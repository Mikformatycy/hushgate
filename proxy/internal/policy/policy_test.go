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
