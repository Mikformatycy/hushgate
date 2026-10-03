package policy

import (
	"testing"

	"github.com/Mikformatycy/hushgate/proxy/internal/vault"
)

func TestDecide(t *testing.T) {
	p := &Policy{Tools: map[string]Sink{"write_file": Local, "send_email": Network, "rm": Deny}, Default: Deny}
	secret := vault.Ref{Tier: vault.Secret}
	pii := vault.Ref{Tier: vault.Confidential}
	internal := vault.Ref{Tier: vault.Internal}
	cases := []struct {
		tool      string
		refs      []vault.Ref
		act       Action
		rehydrate bool
	}{
		{"write_file", []vault.Ref{secret}, Allow, true},
		{"send_email", nil, Allow, false},
		{"send_email", []vault.Ref{internal}, Allow, false},
		{"send_email", []vault.Ref{pii}, Block, false},
		{"send_email", []vault.Ref{pii, secret}, Kill, false},
		{"rm", nil, Block, false},
		{"unknown_tool", nil, Block, false},
	}
	for _, c := range cases {
		act, rh, _ := p.Decide(c.tool, c.refs)
		if act != c.act || rh != c.rehydrate {
			t.Errorf("%s %v: got %s/%v, want %s/%v", c.tool, c.refs, act, rh, c.act, c.rehydrate)
		}
	}
}

func TestTierActionsConfigurable(t *testing.T) {
	p := &Policy{}
	p.Replace(map[string]Sink{"send_email": Network}, Deny,
		map[vault.Tier]Action{vault.Confidential: Allow, vault.Secret: Block})
	if act, _, _ := p.Decide("send_email", []vault.Ref{{Tier: vault.Confidential}}); act != Allow {
		t.Fatalf("C2 should be allowed by policy, got %s", act)
	}
	if act, _, _ := p.Decide("send_email", []vault.Ref{{Tier: vault.Secret}}); act != Block {
		t.Fatalf("C3 should block instead of kill, got %s", act)
	}
	if act, _, _ := p.Decide("unknown", nil); act != Block {
		t.Fatal("default deny not applied")
	}
}
