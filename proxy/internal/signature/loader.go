package signature

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

// Loader reads feeds from files and URLs. URL feeds are fetched in the
// background on a schedule; the last successfully parsed copy of every
// source is kept, so a broken or unreachable feed never removes protection.
type Loader struct {
	Client *http.Client

	mu       sync.Mutex
	raw      map[string][]byte // latest fetched bytes per URL
	fetchErr map[string]string
	lastGood map[string]*Feed
}

func NewLoader() *Loader {
	return &Loader{Client: &http.Client{Timeout: 10 * time.Second}, raw: map[string][]byte{},
		fetchErr: map[string]string{}, lastGood: map[string]*Feed{}}
}

func isURL(src string) bool {
	return strings.HasPrefix(src, "https://") || strings.HasPrefix(src, "http://")
}

// Fetch downloads every URL source now. Errors are recorded per source.
func (l *Loader) Fetch(ctx context.Context, sources []string) {
	for _, src := range sources {
		if !isURL(src) {
			continue
		}
		b, err := l.get(ctx, src)
		l.mu.Lock()
		if err != nil {
			l.fetchErr[src] = err.Error()
		} else {
			l.raw[src], l.fetchErr[src] = b, ""
		}
		l.mu.Unlock()
	}
}

func (l *Loader) get(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := l.Client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	return io.ReadAll(io.LimitReader(resp.Body, 4<<20))
}

// Raw returns the current bytes of every source (files read now, URLs from the
// last fetch). The policy watcher fingerprints these to notice feed changes.
func (l *Loader) Raw(sources []string) [][]byte {
	out := make([][]byte, 0, len(sources))
	for _, src := range sources {
		if isURL(src) {
			l.mu.Lock()
			out = append(out, append([]byte(src), l.raw[src]...))
			l.mu.Unlock()
			continue
		}
		b, _ := os.ReadFile(src)
		out = append(out, append([]byte(src), b...))
	}
	return out
}

// Load parses every source, falling back to its last good copy on failure.
func (l *Loader) Load(sources []string) ([]*Feed, []FeedStatus) {
	l.mu.Lock()
	defer l.mu.Unlock()
	var feeds []*Feed
	var status []FeedStatus
	for _, src := range sources {
		st := FeedStatus{Source: src}
		f, err := l.parse(src)
		if err == nil {
			l.lastGood[src] = f
			if msg := l.fetchErr[src]; msg != "" { // parsed an older download; the newest fetch failed
				st.Error = "latest fetch failed: " + msg + "; using the copy fetched earlier"
			}
		} else {
			st.Error = err.Error()
			if f = l.lastGood[src]; f != nil {
				st.Error += "; previous version still active"
			}
		}
		if f != nil {
			feeds = append(feeds, f)
			st.Name, st.Version, st.Count = f.Name, f.Version, len(f.Signatures)
		}
		status = append(status, st)
	}
	return feeds, status
}

// parse reads one source; caller holds l.mu.
func (l *Loader) parse(src string) (*Feed, error) {
	if !isURL(src) {
		data, err := os.ReadFile(src)
		if err != nil {
			return nil, err
		}
		return ParseFeed(data)
	}
	data := l.raw[src]
	if data == nil {
		if msg := l.fetchErr[src]; msg != "" {
			return nil, fmt.Errorf("fetch failed: %s", msg)
		}
		return nil, fmt.Errorf("not fetched yet")
	}
	return ParseFeed(data)
}
