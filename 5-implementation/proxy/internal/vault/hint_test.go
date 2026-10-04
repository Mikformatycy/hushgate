package vault

import (
	"math/rand"
	"reflect"
	"strings"
	"testing"
)

// The windowed search must find exactly what a search of the whole text finds.
func TestHintsMatchFullSearch(t *testing.T) {
	pieces := []string{
		"0", "1", "4", "5", "9", "44051401359", "4111111111111111", "4111 1111 1111 1111", "5500-0000-0000-0004",
		"PL61109010140000071219812874", "PL61 1090 1014 0000 0712 1981 2874", "DE89370400440532013000",
		"123-456-32-18", "123-45-63-218", "1234563218", "NIP", "nip:", "Nip ", " ", "-", ":", "\n", "\\n", "\"",
		"A", "PL", "X", "sk_live_", "rk_live_", "k_live_", "abc123XYZ7890abcdefGHIJKL", "s", "r", ",", "IBAN ",
	}
	r := rand.New(rand.NewSource(1))
	cases := []string{"", "x", "44051401359", "a44051401359b", "PL61109010140000071219812874", "NIP: 1234563218"}
	for n := 0; n < 4000; n++ {
		var b strings.Builder
		for k := 0; k < 1+r.Intn(12); k++ {
			b.WriteString(pieces[r.Intn(len(pieces))])
		}
		cases = append(cases, b.String())
	}
	for i := range Detectors {
		d := &Detectors[i]
		if d.hint == nil {
			continue
		}
		for _, c := range cases {
			got, want := d.find([]byte(c)), d.findIn([]byte(c), 0)
			if !reflect.DeepEqual(got, want) && !(len(got) == 0 && len(want) == 0) {
				t.Fatalf("%s on %q: windowed %v, full %v", d.Name, c, got, want)
			}
		}
	}
}
