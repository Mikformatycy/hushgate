// Package forward is the company-network mode: an HTTP(S) forward proxy that
// decrypts TLS with the company CA, so agents need no base-URL configuration.
// Every decrypted request is checked for LLM traffic; only allowlisted nodes
// may reach approved providers, and those requests go through the gateway.
package forward

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/big"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/Mikformatycy/hushgate/proxy/internal/audit"
	"github.com/Mikformatycy/hushgate/proxy/internal/detect"
	"github.com/Mikformatycy/hushgate/proxy/internal/gateway"
)

type Proxy struct {
	Nodes     *Registry
	Approved  map[string]bool // LLM provider hosts nodes may use
	Gateway   *gateway.Gateway
	Transport http.RoundTripper
	Audit     audit.Logger

	ca      tls.Certificate
	leafKey *ecdsa.PrivateKey
	mu      sync.Mutex
	certs   map[string]*tls.Certificate
}

func New(caCertFile, caKeyFile string) (*Proxy, error) {
	ca, err := tls.LoadX509KeyPair(caCertFile, caKeyFile)
	if err != nil {
		return nil, fmt.Errorf("load CA: %w", err)
	}
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, err
	}
	return &Proxy{ca: ca, leafKey: key, certs: map[string]*tls.Certificate{}}, nil
}

var hopHeaders = []string{"Connection", "Keep-Alive", "Proxy-Authorization", "Proxy-Connection",
	"Te", "Trailer", "Transfer-Encoding", "Upgrade"}

func (p *Proxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	node := p.Nodes.Authenticate(r)
	switch {
	case r.Method == http.MethodConnect:
		p.bump(w, r, node)
	case r.URL.IsAbs():
		p.handle(w, r, "http", r.URL.Host, node)
	default:
		http.Error(w, "this is a forward proxy", http.StatusBadRequest)
	}
}

// bump terminates the client's TLS with a certificate minted for the target
// host, then serves the decrypted requests.
func (p *Proxy) bump(w http.ResponseWriter, r *http.Request, node *Node) {
	hj, ok := w.(http.Hijacker)
	if !ok {
		http.Error(w, "hijack unsupported", http.StatusInternalServerError)
		return
	}
	conn, _, err := hj.Hijack()
	if err != nil {
		return
	}
	target := r.Host
	if _, err := io.WriteString(conn, "HTTP/1.1 200 Connection established\r\n\r\n"); err != nil {
		conn.Close()
		return
	}
	tlsConn := tls.Server(conn, &tls.Config{
		NextProtos: []string{"http/1.1"},
		GetCertificate: func(hello *tls.ClientHelloInfo) (*tls.Certificate, error) {
			name := hello.ServerName
			if name == "" {
				name = hostname(target)
			}
			return p.certFor(name)
		},
	})
	if err := tlsConn.HandshakeContext(r.Context()); err != nil {
		// Typically a device without the company CA: it fails closed.
		log.Printf("tls handshake with %s for %s: %v", conn.RemoteAddr(), target, err)
		conn.Close()
		return
	}
	srv := &http.Server{
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			p.handle(w, r, "https", target, node)
		}),
		ErrorLog: log.New(io.Discard, "", 0),
	}
	srv.Serve(newSingleListener(tlsConn))
}

func (p *Proxy) handle(w http.ResponseWriter, r *http.Request, scheme, hostport string, node *Node) {
	body, err := io.ReadAll(io.LimitReader(r.Body, 64<<20))
	if err != nil {
		http.Error(w, "read body", http.StatusBadRequest)
		return
	}
	r.Body = io.NopCloser(bytes.NewReader(body))
	host := hostname(hostport)

	m, isLLM := detect.LLM(r, body)
	switch {
	case !isLLM:
		p.passthrough(w, r, scheme, hostport, body)
	case node == nil:
		p.Audit.Record(audit.Event{Agent: p.describe(r.RemoteAddr), Kind: "node_blocked", Host: host,
			Reason: fmt.Sprintf("%s LLM request from a device not on the agent allowlist (%s)", m.Format, m.Signal)})
		deny(w, m.Format, "this device is not registered as an agent node")
	case !p.Approved[host]:
		p.Audit.Record(audit.Event{Agent: node.ID, Kind: "shadow_ai", Host: host,
			Reason: fmt.Sprintf("%s LLM request to unapproved host (%s)", m.Format, m.Signal)})
		deny(w, m.Format, host+" is not an approved LLM provider")
	default:
		gw := *p.Gateway
		gw.Upstream = &url.URL{Scheme: scheme, Host: trimDefaultPort(scheme, hostport)}
		r.Header.Set("X-Agent-Id", node.ID) // identity comes from the device, not the agent
		gw.ServeHTTP(w, r)
	}
}

// passthrough forwards ordinary web traffic untouched.
func (p *Proxy) passthrough(w http.ResponseWriter, r *http.Request, scheme, hostport string, body []byte) {
	target := scheme + "://" + trimDefaultPort(scheme, hostport) + r.URL.RequestURI()
	out, err := http.NewRequestWithContext(r.Context(), r.Method, target, bytes.NewReader(body))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	out.Header = r.Header.Clone()
	for _, h := range hopHeaders {
		out.Header.Del(h)
	}
	resp, err := p.Transport.RoundTrip(out)
	if err != nil {
		http.Error(w, "upstream: "+err.Error(), http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()
	for k, vs := range resp.Header {
		w.Header()[k] = vs
	}
	for _, h := range hopHeaders {
		w.Header().Del(h)
	}
	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
}

func (p *Proxy) certFor(name string) (*tls.Certificate, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if c, ok := p.certs[name]; ok {
		return c, nil
	}
	serial, _ := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	tmpl := &x509.Certificate{
		SerialNumber: serial,
		Subject:      pkix.Name{CommonName: name},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(7 * 24 * time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}
	if ip := net.ParseIP(name); ip != nil {
		tmpl.IPAddresses = []net.IP{ip}
	} else {
		tmpl.DNSNames = []string{name}
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, p.ca.Leaf, &p.leafKey.PublicKey, p.ca.PrivateKey)
	if err != nil {
		return nil, err
	}
	c := &tls.Certificate{Certificate: [][]byte{der, p.ca.Leaf.Raw}, PrivateKey: p.leafKey}
	p.certs[name] = c
	return c, nil
}

// describe names an unregistered device: reverse DNS (container name in the
// demo, hostname in a real network) plus IP.
func (p *Proxy) describe(remoteAddr string) string {
	ip, _, err := net.SplitHostPort(remoteAddr)
	if err != nil {
		ip = remoteAddr
	}
	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancel()
	if names, err := net.DefaultResolver.LookupAddr(ctx, ip); err == nil && len(names) > 0 {
		name, _, _ := strings.Cut(names[0], ".")
		return fmt.Sprintf("unregistered: %s (%s)", name, ip)
	}
	return "unregistered: " + ip
}

// deny answers in the error format the client's SDK expects.
func deny(w http.ResponseWriter, format, msg string) {
	msg = "hushgate: " + msg
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusForbidden)
	if format == "anthropic" {
		json.NewEncoder(w).Encode(map[string]any{"type": "error",
			"error": map[string]string{"type": "permission_error", "message": msg}})
		return
	}
	json.NewEncoder(w).Encode(map[string]any{
		"error": map[string]string{"message": msg, "type": "permission_error", "code": "blocked_by_gate"}})
}

func hostname(hostport string) string {
	if h, _, err := net.SplitHostPort(hostport); err == nil {
		return h
	}
	return hostport
}

func trimDefaultPort(scheme, hostport string) string {
	h, port, err := net.SplitHostPort(hostport)
	if err != nil {
		return hostport
	}
	if (scheme == "https" && port == "443") || (scheme == "http" && port == "80") {
		return h
	}
	return hostport
}

// singleListener lets http.Server serve exactly one already-accepted conn.
type singleListener struct {
	mu   sync.Mutex
	conn net.Conn
	done chan struct{}
	once sync.Once
}

func newSingleListener(c net.Conn) *singleListener {
	return &singleListener{conn: c, done: make(chan struct{})}
}

func (l *singleListener) Accept() (net.Conn, error) {
	l.mu.Lock()
	c := l.conn
	l.conn = nil
	l.mu.Unlock()
	if c != nil {
		return &closeNotifyConn{Conn: c, l: l}, nil
	}
	<-l.done
	return nil, net.ErrClosed
}

func (l *singleListener) Close() error   { return nil }
func (l *singleListener) Addr() net.Addr { return &net.TCPAddr{} }
func (l *singleListener) finish()        { l.once.Do(func() { close(l.done) }) }

type closeNotifyConn struct {
	net.Conn
	l *singleListener
}

func (c *closeNotifyConn) Close() error {
	err := c.Conn.Close()
	c.l.finish()
	return err
}
