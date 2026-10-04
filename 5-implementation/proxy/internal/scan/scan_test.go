package scan

import "testing"

func TestObserveDedupesAndResolves(t *testing.T) {
	s := NewStore(10)
	if !s.Observe("a1", "read_file", "hello") || s.Observe("a1", "read_file", "hello") {
		t.Fatal("same agent+text should be queued once")
	}
	if !s.Observe("a2", "read_file", "hello") {
		t.Fatal("another agent reading the same text is a separate event")
	}
	p := s.Pending(10)
	if len(p) != 2 {
		t.Fatalf("pending = %d", len(p))
	}
	if _, err := s.Resolve(p[0].ID, 1.5, "m"); err == nil {
		t.Fatal("out-of-range probability accepted")
	}
	it, err := s.Resolve(p[0].ID, 0.97, "jev-1.13.0")
	if err != nil || *it.Injection != 0.97 {
		t.Fatalf("%+v %v", it, err)
	}
	if _, err := s.Resolve(p[0].ID, 0.1, "m"); err == nil {
		t.Fatal("resolved twice")
	}
	if len(s.Pending(10)) != 1 {
		t.Fatal("resolved item still pending")
	}
}

func TestBounded(t *testing.T) {
	s := NewStore(2)
	s.Observe("a", "x", "1")
	s.Observe("a", "x", "2")
	s.Observe("a", "x", "3")
	if p := s.Pending(10); len(p) != 2 || p[0].Text != "2" {
		t.Fatalf("pending = %+v", p)
	}
}
