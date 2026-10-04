// Package engine applies a policy file to the running components. It is the
// only place that knows how config settings map onto policy, vault, reviews
// and the device allowlist.
package engine

import (
	"github.com/Mikformatycy/hushgate/proxy/internal/config"
	"github.com/Mikformatycy/hushgate/proxy/internal/forward"
	"github.com/Mikformatycy/hushgate/proxy/internal/policy"
	"github.com/Mikformatycy/hushgate/proxy/internal/review"
	"github.com/Mikformatycy/hushgate/proxy/internal/shell"
	"github.com/Mikformatycy/hushgate/proxy/internal/signature"
	"github.com/Mikformatycy/hushgate/proxy/internal/vault"
)

type Engine struct {
	Policy     *policy.Policy
	Vault      *vault.Vault
	Reviews    *review.Store
	Nodes      *forward.Registry
	Guard      *shell.Guard
	Signatures *signature.Set
	Feeds      *signature.Loader
}

// Apply is a config.ApplyFunc. The vault is loaded first because it is the
// only step that can fail; nothing changes unless it succeeds.
func (e *Engine) Apply(_, cfg *config.Config) error {
	overrides := map[string]vault.Tier{}
	for name, t := range cfg.Vault.Overrides {
		overrides[name], _ = vault.ParseTier(t)
	}
	vars, err := e.Vault.LoadEnvFiles(cfg.Vault.EnvFiles, overrides)
	if err != nil {
		return err
	}
	maskFrom, _ := vault.ParseTier(cfg.Masking.MaskFrom)
	e.Vault.SetMaskFrom(maskFrom)

	tools := make(map[string]policy.Sink, len(cfg.Tools.Rules))
	for name, s := range cfg.Tools.Rules {
		tools[name] = policy.Sink(s)
	}
	onNetwork := map[vault.Tier]policy.Action{}
	for t, a := range cfg.Masking.OnNetworkTool {
		tier, _ := vault.ParseTier(t)
		onNetwork[tier] = policy.Action(a)
	}
	e.Policy.Replace(tools, policy.Sink(cfg.Tools.Default), onNetwork)

	if e.Guard != nil {
		e.Guard.Replace(cfg.BashGuard.Tools, cfg.BashGuard.NetworkCommands)
	}
	if e.Signatures != nil && e.Feeds != nil {
		// A broken or unreachable feed keeps its last good copy; it never fails the reload.
		feeds, status := e.Feeds.Load(cfg.Signatures.Feeds)
		e.Signatures.Replace(feeds, status, cfg.Signatures.Disabled)
	}

	if e.Nodes != nil {
		nodes := make([]forward.Node, 0, len(cfg.Nodes))
		for _, n := range cfg.Nodes {
			nodes = append(nodes, forward.Node{ID: n.ID, Owner: n.Owner, Token: n.Token})
		}
		e.Nodes.Replace(nodes)
	}

	if e.Reviews != nil {
		for _, x := range vars {
			if x.Reason == vault.FallbackReason {
				val, _ := e.Vault.Value(x.Name)
				e.Reviews.ObserveVariable(x.Name, val, x.Tier.String()+" (masked by default)")
			}
		}
		// A rule written into the file (from the Review page or by hand) settles its review.
		for name, s := range cfg.Tools.Rules {
			e.Reviews.Settle("tool:"+name, s)
		}
		for name, t := range cfg.Vault.Overrides {
			e.Reviews.Settle("var:"+name, t)
		}
	}
	return nil
}
