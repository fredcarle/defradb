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

package js

import (
	"fmt"
	"syscall/js"

	"github.com/davecgh/go-spew/spew"
	"github.com/sourcenetwork/defradb/js/utils"
	"github.com/sourcenetwork/goji"
)

// setExternalHost injects a JavaScript-based Host implementation into the DefraDB node.
// This allows browser-based P2P implementations (using js-libp2p) to be used instead
// of the default Go libp2p implementation.
//
// The jsHost object must implement all methods defined in the client.Host interface.
// See js/types/host.ts for the TypeScript interface definition.
func (c *Client) verifyHost(this js.Value, args []js.Value) (js.Value, error) {
	if len(args) < 1 {
		return js.Undefined(), ErrInvalidHostObject
	}

	jsHost := args[0]
	if !jsHost.Truthy() {
		return js.Undefined(), ErrInvalidHostObject
	}

	// Validate that the jsHost object has the required methods
	requiredMethods := []string{
		"id",
		"addresses",
		"pubkey",
		"connect",
		"disconnect",
		"send",
		"sign",
		"setStreamHandler",
		"addPubSubTopic",
		"removePubSubTopic",
		"publishToTopicAsync",
		"publishToTopic",
		"ipldStore",
		"contextWithSession",
		"setBlockAccessFunc",
	}

	for _, method := range requiredMethods {
		if !jsHost.Get(method).Truthy() {
			return js.Undefined(), ErrMissingHostMethod
		}
	}

	// Set the jsHost as a global so node_p2p_js.go can pick it up during startP2P
	js.Global().Set("defraP2PHost", jsHost)

	// Note: The actual JSHostAdapter will be created in node_p2p_js.go
	// when startP2P is called during database initialization.
	// If the DB is already created when this is called, the P2P system
	// won't be initialized until the next database creation.

	return js.Undefined(), nil
}

// getPeerInfo returns the P2P peer information if available.
func (c *Client) getPeerInfo(this js.Value, args []js.Value) (js.Value, error) {
	info, err := c.node.DB.PeerInfo()
	fmt.Println("getPeerInfo called", err, info)
	if err != nil {
		return js.Undefined(), err
	}

	return goji.MarshalJS(info)
}

// p2pConnect connects to a peer with the given addresses.
func (c *Client) p2pConnect(this js.Value, args []js.Value) (js.Value, error) {
	ctx, err := utils.ContextArg(args, 0, c.txns)
	if err != nil {
		return js.Undefined(), err
	}
	spew.Dump(args)
	var addresses []string
	if err := goji.UnmarshalJS(args[1], &addresses); err != nil {
		return js.Undefined(), err
	}
	spew.Dump(addresses)
	return js.Undefined(), c.node.DB.Connect(ctx, addresses)
}

// p2pSetReplicator adds a replicator for the specified collections.
func (c *Client) p2pSetReplicator(this js.Value, args []js.Value) (js.Value, error) {
	ctx, err := utils.ContextArg(args, 0, c.txns)
	if err != nil {
		return js.Undefined(), err
	}

	var addresses []string
	if err := goji.UnmarshalJS(args[1], &addresses); err != nil {
		return js.Undefined(), err
	}

	var collectionNames []string
	if len(args) > 2 {
		if err := goji.UnmarshalJS(args[2], &collectionNames); err != nil {
			return js.Undefined(), err
		}
	}

	return js.Undefined(), c.node.DB.SetReplicator(ctx, addresses, collectionNames...)
}

// p2pDeleteReplicator removes a replicator.
func (c *Client) p2pDeleteReplicator(this js.Value, args []js.Value) (js.Value, error) {
	ctx, err := utils.ContextArg(args, 0, c.txns)
	if err != nil {
		return js.Undefined(), err
	}

	replicatorID, err := utils.StringArg(args, 1, "replicatorID")
	if err != nil {
		return js.Undefined(), err
	}

	var collectionNames []string
	if len(args) > 2 {
		if err := goji.UnmarshalJS(args[2], &collectionNames); err != nil {
			return js.Undefined(), err
		}
	}

	return js.Undefined(), c.node.DB.DeleteReplicator(ctx, replicatorID, collectionNames...)
}

// p2pGetAllReplicators returns all configured replicators.
func (c *Client) p2pGetAllReplicators(this js.Value, args []js.Value) (js.Value, error) {
	ctx, err := utils.ContextArg(args, 0, c.txns)
	if err != nil {
		return js.Undefined(), err
	}

	replicators, err := c.node.DB.GetAllReplicators(ctx)
	if err != nil {
		return js.Undefined(), err
	}

	return goji.MarshalJS(replicators)
}

// p2pAddCollections adds collections to the P2P system.
func (c *Client) p2pAddCollections(this js.Value, args []js.Value) (js.Value, error) {
	ctx, err := utils.ContextArg(args, 0, c.txns)
	if err != nil {
		return js.Undefined(), err
	}

	var collectionNames []string
	if err := goji.UnmarshalJS(args[1], &collectionNames); err != nil {
		return js.Undefined(), err
	}

	return js.Undefined(), c.node.DB.AddP2PCollections(ctx, collectionNames...)
}

// p2pRemoveCollections removes collections from the P2P system.
func (c *Client) p2pRemoveCollections(this js.Value, args []js.Value) (js.Value, error) {
	ctx, err := utils.ContextArg(args, 0, c.txns)
	if err != nil {
		return js.Undefined(), err
	}

	var collectionNames []string
	if err := goji.UnmarshalJS(args[1], &collectionNames); err != nil {
		return js.Undefined(), err
	}

	return js.Undefined(), c.node.DB.RemoveP2PCollections(ctx, collectionNames...)
}

// p2pGetAllCollections returns all P2P-enabled collections.
func (c *Client) p2pGetAllCollections(this js.Value, args []js.Value) (js.Value, error) {
	ctx, err := utils.ContextArg(args, 0, c.txns)
	if err != nil {
		return js.Undefined(), err
	}

	collections, err := c.node.DB.GetAllP2PCollections(ctx)
	if err != nil {
		return js.Undefined(), err
	}

	return goji.MarshalJS(collections)
}

// p2pAddDocuments adds documents to the P2P system.
func (c *Client) p2pAddDocuments(this js.Value, args []js.Value) (js.Value, error) {
	ctx, err := utils.ContextArg(args, 0, c.txns)
	if err != nil {
		return js.Undefined(), err
	}

	var docIDs []string
	if err := goji.UnmarshalJS(args[1], &docIDs); err != nil {
		return js.Undefined(), err
	}

	return js.Undefined(), c.node.DB.AddP2PDocuments(ctx, docIDs...)
}

// p2pRemoveDocuments removes documents from the P2P system.
func (c *Client) p2pRemoveDocuments(this js.Value, args []js.Value) (js.Value, error) {
	ctx, err := utils.ContextArg(args, 0, c.txns)
	if err != nil {
		return js.Undefined(), err
	}

	var docIDs []string
	if err := goji.UnmarshalJS(args[1], &docIDs); err != nil {
		return js.Undefined(), err
	}

	return js.Undefined(), c.node.DB.RemoveP2PDocuments(ctx, docIDs...)
}

// p2pGetAllDocuments returns all P2P-enabled documents.
func (c *Client) p2pGetAllDocuments(this js.Value, args []js.Value) (js.Value, error) {
	ctx, err := utils.ContextArg(args, 0, c.txns)
	if err != nil {
		return js.Undefined(), err
	}

	docIDs, err := c.node.DB.GetAllP2PDocuments(ctx)
	if err != nil {
		return js.Undefined(), err
	}

	return goji.MarshalJS(docIDs)
}

// p2pSyncDocuments synchronizes specific documents from the network.
func (c *Client) p2pSyncDocuments(this js.Value, args []js.Value) (js.Value, error) {
	ctx, err := utils.ContextArg(args, 0, c.txns)
	if err != nil {
		return js.Undefined(), err
	}

	collectionName, err := utils.StringArg(args, 1, "collectionName")
	if err != nil {
		return js.Undefined(), err
	}

	var docIDs []string
	if err := goji.UnmarshalJS(args[2], &docIDs); err != nil {
		return js.Undefined(), err
	}

	return js.Undefined(), c.node.DB.SyncDocuments(ctx, collectionName, docIDs)
}

// p2pSyncCollectionVersions synchronizes specific collection versions from the network.
func (c *Client) p2pSyncCollectionVersions(this js.Value, args []js.Value) (js.Value, error) {
	ctx, err := utils.ContextArg(args, 0, c.txns)
	if err != nil {
		return js.Undefined(), err
	}

	var versionIDs []string
	if err := goji.UnmarshalJS(args[1], &versionIDs); err != nil {
		return js.Undefined(), err
	}

	return js.Undefined(), c.node.DB.SyncCollectionVersions(ctx, versionIDs...)
}
