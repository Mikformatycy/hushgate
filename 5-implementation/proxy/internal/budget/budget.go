// Package budget tracks token usage per agent and holds the kill switch.
package budget

import (
	"context"
	"errors"
	"sort"
	"strings"
	"sync"

	"github.com/redis/go-redis/v9"
)

type Status struct {
	Used       int64
	Killed     bool
	KillReason string
}

type Store interface {
	Status(ctx context.Context, agent string) (Status, error)
	AddUsage(ctx context.Context, agent string, tokens int64) (int64, error)
	Kill(ctx context.Context, agent, reason string) error
	Reset(ctx context.Context, agent string) error
	Agents(ctx context.Context) ([]string, error)
}

// Redis stores counters under gate:tokens:<agent> and gate:killed:<agent>.
type Redis struct{ c *redis.Client }

func NewRedis(addr string) *Redis { return &Redis{c: redis.NewClient(&redis.Options{Addr: addr})} }

func (r *Redis) Ping(ctx context.Context) error { return r.c.Ping(ctx).Err() }

func (r *Redis) Status(ctx context.Context, agent string) (Status, error) {
	pipe := r.c.Pipeline()
	used := pipe.Get(ctx, "gate:tokens:"+agent)
	killed := pipe.Get(ctx, "gate:killed:"+agent)
	if _, err := pipe.Exec(ctx); err != nil && !errors.Is(err, redis.Nil) {
		return Status{}, err
	}
	s := Status{}
	s.Used, _ = used.Int64()
	if reason, err := killed.Result(); err == nil {
		s.Killed, s.KillReason = true, reason
	}
	return s, nil
}

func (r *Redis) AddUsage(ctx context.Context, agent string, tokens int64) (int64, error) {
	return r.c.IncrBy(ctx, "gate:tokens:"+agent, tokens).Result()
}

func (r *Redis) Kill(ctx context.Context, agent, reason string) error {
	return r.c.Set(ctx, "gate:killed:"+agent, reason, 0).Err()
}

func (r *Redis) Reset(ctx context.Context, agent string) error {
	return r.c.Del(ctx, "gate:tokens:"+agent, "gate:killed:"+agent).Err()
}

func (r *Redis) Agents(ctx context.Context) ([]string, error) {
	seen := map[string]bool{}
	for _, prefix := range []string{"gate:tokens:", "gate:killed:"} {
		iter := r.c.Scan(ctx, 0, prefix+"*", 100).Iterator()
		for iter.Next(ctx) {
			seen[strings.TrimPrefix(iter.Val(), prefix)] = true
		}
		if err := iter.Err(); err != nil {
			return nil, err
		}
	}
	return sortedKeys(seen), nil
}

// Memory is an in-process store for tests and running without Redis.
type Memory struct {
	mu     sync.Mutex
	used   map[string]int64
	killed map[string]string
}

func NewMemory() *Memory { return &Memory{used: map[string]int64{}, killed: map[string]string{}} }

func (m *Memory) Status(_ context.Context, agent string) (Status, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	reason, killed := m.killed[agent]
	return Status{Used: m.used[agent], Killed: killed, KillReason: reason}, nil
}

func (m *Memory) AddUsage(_ context.Context, agent string, tokens int64) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.used[agent] += tokens
	return m.used[agent], nil
}

func (m *Memory) Kill(_ context.Context, agent, reason string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.killed[agent] = reason
	return nil
}

func (m *Memory) Reset(_ context.Context, agent string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.used, agent)
	delete(m.killed, agent)
	return nil
}

func (m *Memory) Agents(_ context.Context) ([]string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	seen := map[string]bool{}
	for a := range m.used {
		seen[a] = true
	}
	for a := range m.killed {
		seen[a] = true
	}
	return sortedKeys(seen), nil
}

func sortedKeys(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
