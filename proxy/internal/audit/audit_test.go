package audit

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFileRoundTripAndRestore(t *testing.T) {
	path := filepath.Join(t.TempDir(), "logs", "audit.jsonl")
	fl, err := OpenFile(path)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 5; i++ {
		fl.Record(Event{Agent: "a1", Kind: "tool_call", Action: "allow", Tool: "Read"})
	}
	// a crash mid-write leaves a damaged last line
	f, _ := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0)
	f.WriteString(`{"agent":"a1","kind":"tool_c`)
	f.Close()

	events, err := ReadFile(path)
	if err != nil || len(events) != 5 || events[0].Time.IsZero() {
		t.Fatalf("read %d events, err %v", len(events), err)
	}
	r := NewRing(3)
	r.Restore(events)
	got := r.After(0)
	if len(got) != 3 || got[0].ID != 1 || got[2].ID != 3 {
		t.Fatalf("restore kept %+v", got)
	}
}
