package main

import (
	"github.com/borg0ai/dx-issuegraph/internal/configfile"
	"github.com/borg0ai/dx-issuegraph/internal/issuegraph"
)

// resolveProxiedTopology reports which proxied-server SHAPE a workspace is, for
// the capability rows whose answer depends on it.
//
// The shapes are ordered by how much of the store issuegraph owns, and the first one
// that matches wins:
//
//	team-server     the database belongs to issuegraph-team-server, which owns its
//	                schema and identity. Ownership beats transport — a
//	                bts-managed database is somebody else's store even when the
//	                dolt process happens to be local.
//	external-*      an operator's Dolt server that issuegraph only connects to. The
//	                sidecar's External block is what records it, and the socket
//	                field is what separates the two transports.
//	managed-local   no External block: issuegraph spawned this dolt sql-server itself,
//	                under this user, on this filesystem.
//
// A workspace issuegraph cannot read is reported as unknown rather than guessed at.
// Every topology-keyed row treats unknown as refused, so a corrupt sidecar
// costs a capability instead of buying one; the provider open that follows
// fails on the same file with a much better message than this function could
// produce, so this is a policy decision, not the error path.
func resolveProxiedTopology(issuegraphDir string) ProxyTopology {
	if issuegraphDir == "" {
		issuegraphDir = issuegraph.FindIssueGraphDir()
	}
	if issuegraphDir == "" {
		return ProxyTopologyUnknown
	}
	cfg, err := configfile.Load(issuegraphDir)
	if err != nil {
		return ProxyTopologyUnknown
	}
	if cfg != nil && cfg.IsTeamServerManaged() {
		return ProxyTopologyTeamServer
	}
	info, err := configfile.LoadProxiedServerClientInfo(issuegraphDir)
	if err != nil {
		return ProxyTopologyUnknown
	}
	if info == nil || info.External == nil {
		return ProxyTopologyManagedLocal
	}
	if info.External.Socket != "" {
		return ProxyTopologyExternalUnix
	}
	return ProxyTopologyExternalTCP
}
