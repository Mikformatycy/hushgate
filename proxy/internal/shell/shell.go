// Package shell is the Bash guard: it finds commands in a shell tool call that
// send data off the machine, so the gate can treat that call as a network
// tool instead of a local one. It is a deterministic first line; network
// egress rules (the sandbox) remain the hard guarantee.
package shell

import (
	"regexp"
	"strings"
	"sync"
)

// Guard holds which tools take shell commands and which programs count as
// network egress. Updated on live policy reload.
type Guard struct {
	mu       sync.RWMutex
	tools    map[string]bool
	commands map[string]bool
}

func NewGuard(tools, commands []string) *Guard {
	g := &Guard{}
	g.Replace(tools, commands)
	return g
}

func (g *Guard) Replace(tools, commands []string) {
	t, c := set(tools), set(commands)
	g.mu.Lock()
	g.tools, g.commands = t, c
	g.mu.Unlock()
}

// IsShellTool reports whether the tool's input is a shell command.
func (g *Guard) IsShellTool(tool string) bool {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.tools[tool]
}

// NetworkCommand returns the first egress program the command runs, if any.
func (g *Guard) NetworkCommand(cmd string) (string, bool) {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return findNetwork(Normalize(cmd), g.commands, 0)
}

var (
	// Separators that start a new command: ; & | && || newline ( ) ` $(
	segmentSplit = regexp.MustCompile("(\\|\\||&&|[;&|\\n()`]|\\$\\()")
	// Wrappers that run the next word as the real command.
	wrappers = set([]string{"sudo", "doas", "env", "time", "nohup", "nice", "ionice", "exec", "command",
		"xargs", "builtin", "stdbuf", "timeout", "watch"})
	shells       = set([]string{"sh", "bash", "zsh", "dash", "ksh", "busybox"})
	pythonNet    = regexp.MustCompile(`\b(urllib|requests|http\.client|httpx|socket|aiohttp|ftplib|smtplib)\b`)
	envAssign    = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*=`)
	pythonBinary = regexp.MustCompile(`^python[0-9.]*$`)
)

// Normalize removes the quoting tricks that hide a program name from a naive
// match: quotes inside the name (c'u'rl), "curl", \curl, and line continuations.
func Normalize(cmd string) string {
	cmd = strings.ReplaceAll(cmd, "\\\n", " ")
	cmd = strings.NewReplacer(`'`, "", `"`, "", `\`, "").Replace(cmd)
	return cmd
}

func findNetwork(cmd string, commands map[string]bool, depth int) (string, bool) {
	if depth > 3 {
		return "", false
	}
	for _, seg := range segmentSplit.Split(cmd, -1) {
		words := strings.Fields(seg)
		i := 0
		for i < len(words) && (wrappers[base(words[i])] || envAssign.MatchString(words[i]) ||
			(i > 0 && wrapperArg(words[i]) && wrappers[base(words[i-1])])) {
			i++
		}
		if i >= len(words) {
			continue
		}
		prog := base(words[i])
		switch {
		case commands[prog]:
			return prog, true
		case shells[prog] && i+1 < len(words) && words[i+1] == "-c":
			// bash -c "curl ..." (quotes already removed): check what follows
			if p, ok := findNetwork(strings.Join(words[i+2:], " "), commands, depth+1); ok {
				return p, true
			}
		case pythonBinary.MatchString(prog) && pythonNet.MatchString(seg):
			return prog + " (network library)", true
		}
	}
	return "", false
}

// wrapperArg is a flag or number given to a wrapper: sudo -E, timeout 5, nice -n.
func wrapperArg(w string) bool {
	return strings.HasPrefix(w, "-") || strings.Trim(w, "0123456789.smhd") == ""
}

// base strips a path: /usr/bin/curl -> curl.
func base(w string) string {
	if i := strings.LastIndexByte(w, '/'); i >= 0 {
		return w[i+1:]
	}
	return w
}

func set(xs []string) map[string]bool {
	m := make(map[string]bool, len(xs))
	for _, x := range xs {
		m[x] = true
	}
	return m
}
