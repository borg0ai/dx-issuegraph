//go:build cgo

package main

func federationLocalOnlyCases() []localOnlyCase {
	return []localOnlyCase{
		{name: "federation sync", cmd: federationSyncCmd},
		{name: "federation status", cmd: federationStatusCmd},
		{name: "federation add-peer", cmd: federationAddPeerCmd, args: []string{"peer", "file:///tmp/peer"}},
	}
}
