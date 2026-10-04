package vault

import (
	"strings"
	"testing"
)

func TestPersonalDataMasked(t *testing.T) {
	cases := []struct {
		name, text, secret, kind string
	}{
		{"iban compact", `pay to PL61109010140000071219812874 today`, "PL61109010140000071219812874", "IBAN"},
		{"iban grouped", `IBAN: PL61 1090 1014 0000 0712 1981 2874, thanks`, "PL61 1090 1014 0000 0712 1981 2874", "IBAN"},
		{"iban followed by caps word", `GB82WEST12345698765432 ABCD`, "GB82WEST12345698765432", "IBAN"},
		{"card", `card 4111 1111 1111 1111 exp 12/27`, "4111 1111 1111 1111", "CARD"},
		{"card then year", `5555555555554444 2027`, "5555555555554444", "CARD"},
		{"amex", `amex 378282246310005`, "378282246310005", "CARD"},
		{"pesel after json newline", `client:\n90031501238`, "90031501238", "PESEL"},
		{"nip formatted", `NIP 123-456-32-18`, "123-456-32-18", "NIP"},
		{"nip keyword", `nip:5260250274`, "5260250274", "NIP"},
	}
	for _, c := range cases {
		v := New(Confidential)
		body := []byte(`{"content":"` + c.text + `"}`)
		masked, refs := v.Mask(body)
		s := string(masked)
		if strings.Contains(s, c.secret) || !strings.Contains(s, "{{VAULT_"+c.kind+"_") {
			t.Errorf("%s: not masked: %s", c.name, s)
			continue
		}
		if len(refs) != 1 || refs[0].Tier != Confidential {
			t.Errorf("%s: refs = %+v", c.name, refs)
		}
		if v.RehydrateJSON(s) != string(body) {
			t.Errorf("%s: rehydrate mismatch: %s", c.name, v.RehydrateJSON(s))
		}
	}
}

func TestPersonalDataNoFalsePositives(t *testing.T) {
	for _, text := range []string{
		`PL61109010140000071219812875`, // bad IBAN checksum
		`4111111111111112`,             // bad Luhn
		`1696334400000`,                // unix ms timestamp
		`44051401358`,                  // bad PESEL check digit
		`order 12345678901`,            // 11 digits, invalid PESEL
		`phone 1234567890`,             // 10 digits without NIP keyword
		`NIP 1234567890`,               // bad NIP checksum
		`build 9003150123812`,          // valid PESEL inside a longer number
		`version 2.4.1, port 8080`,
	} {
		v := New(Confidential)
		masked, refs := v.Mask([]byte(`{"content":"` + text + `"}`))
		if len(refs) != 0 {
			t.Errorf("false positive on %q: %s", text, masked)
		}
	}
}

func TestAdjacentNumbersBothMasked(t *testing.T) {
	v := New(Confidential)
	masked, refs := v.Mask([]byte(`{"content":"90031501238,44051401359"}`))
	if len(refs) != 2 || strings.Contains(string(masked), "90031501238") || strings.Contains(string(masked), "44051401359") {
		t.Fatalf("second PESEL leaked: %s", masked)
	}
}

func TestClassifyWhy(t *testing.T) {
	cases := []struct {
		name, value string
		tier        Tier
		reason      string
	}{
		{"DB_PASSWORD", "hunter2", Secret, "name contains PASSWORD"},
		{"X", "AKIAIOSFODNN7EXAMPLE", Secret, "value matches AWS_KEY format"},
		{"SETTLEMENT_ACCOUNT", "PL61109010140000071219812874", Confidential, "value is a valid IBAN"},
		{"API_HOST", "internal.corp", Internal, "name contains HOST"},
		{"FLAG", "true", Public, "empty or boolean value"},
		{"TEAM_CHANNEL", "payments-oncall", Confidential, FallbackReason},
	}
	for _, c := range cases {
		tier, reason := ClassifyWhy(c.name, c.value)
		if tier != c.tier || !strings.HasPrefix(reason, c.reason) {
			t.Errorf("%s: got %s %q, want %s %q", c.name, tier, reason, c.tier, c.reason)
		}
	}
}
