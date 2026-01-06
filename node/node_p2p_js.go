// Copyright 2025 Democratized Data Foundation
//
// Use of this software is governed by the Business Source License
// included in the file licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with
// the Business Source License, use of this software will be governed
// by the Apache License, Version 2.0, included in the file
// licenses/APL.txt.

//go:build js

package node

import (
	"context"
	"fmt"
	"syscall/js"

	"github.com/davecgh/go-spew/spew"
	"github.com/sourcenetwork/corekv"
	"github.com/sourcenetwork/goji"
	"github.com/sourcenetwork/immutable"

	"github.com/sourcenetwork/defradb/internal/db"
	jsDatastore "github.com/sourcenetwork/defradb/js/datastore"
	"github.com/sourcenetwork/defradb/js/host"
)

// startP2P checks for a JavaScript Host and initializes P2P if available.
// The JavaScript Host should be set on window.defraP2PHost before DB initialization.
func (n *Node) startP2P(_ context.Context, store corekv.ReaderWriter, chunkSize immutable.Option[int]) error {
	if n.config.disableP2P {
		return nil
	}

	// Check if a JavaScript Host is available
	global := js.Global()
	jsPeer := global.Get("Peer")

	// If no external host is set, P2P is not available
	if !jsPeer.Truthy() {
		// This is not an error - P2P is optional and can be set later
		return nil
	}

	// Create the blockstore that will be used by the P2P layer
	// This blockstore is passed to the JavaScript Peer.create() method as the first argument
	blockstore := jsDatastore.NewJSBlockstore(store, chunkSize)

	// Call Peer.create(store, options?) - store is the first positional argument
	prom := jsPeer.Call("create", blockstore.JSValue())
	jsHost, err := goji.Await(goji.PromiseValue(prom))
	if err != nil {
		return err
	}

	adapter := host.NewJSHostAdapter(jsHost[0])
	fmt.Println("adding p2p to db")
	spew.Dump(jsHost[0])
	n.options = append(n.options, db.WithP2P(adapter))
	// Add the adapter as a P2P option (same pattern as node_p2p.go)
	n.peer = adapter

	return nil
}
