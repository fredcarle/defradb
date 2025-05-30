// Copyright 2025 Democratized Data Foundation
//
// Use of this software is governed by the Business Source License
// included in the file licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with
// the Business Source License, use of this software will be governed
// by the Apache License, Version 2.0, included in the file
// licenses/APL.txt.

// P2P networking stack does not work in JS builds.
//
//go:build !js

package node

import (
	"context"

	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/sourcenetwork/defradb/client"
	"github.com/sourcenetwork/defradb/datastore"
	"github.com/sourcenetwork/defradb/internal/db"
	"github.com/sourcenetwork/defradb/internal/kms"
	"github.com/sourcenetwork/defradb/net"
	netConfig "github.com/sourcenetwork/defradb/net/config"
)

func (n *Node) startP2P(ctx context.Context) error {
	if n.config.disableP2P {
		return nil
	}
	var err error
	n.peer, err = net.NewPeer(
		ctx,
		n.Events(),
		n.db.DocumentACP(),
		n.db,
		filterOptions[netConfig.NodeOpt](n.options)...,
	)
	if err != nil {
		return err
	}

	ident, err := n.db.GetNodeIdentity(ctx)
	if err != nil {
		return err
	}
	if n.config.kmsType.HasValue() {
		switch n.config.kmsType.Value() {
		case kms.PubSubServiceType:
			n.kmsService, err = kms.NewPubSubService(
				ctx,
				n.peer.PeerID(),
				n.peer.Server(),
				n.Events(),
				datastore.EncstoreFrom(n.db.Rootstore()),
				n.db.DocumentACP(),
				db.NewCollectionRetriever(n.db),
				ident.Value().DID,
			)
		}
		if err != nil {
			return err
		}
	}
	return nil
}

func (n *Node) Connect(ctx context.Context, addr peer.AddrInfo) error {
	return n.peer.Connect(ctx, addr)
}

// The methods below implement the `client.P2P` interface

var _ client.P2P = (*Node)(nil)

// TODO: check that peer is not nil
func (n *Node) PeerInfo() peer.AddrInfo {
	return n.peer.PeerInfo()
}

func (n *Node) SetReplicator(ctx context.Context, info peer.AddrInfo, collections ...string) error {
	return n.peer.SetReplicator(ctx, info, collections...)
}

func (n *Node) DeleteReplicator(ctx context.Context, info peer.AddrInfo, collections ...string) error {
	return n.peer.DeleteReplicator(ctx, info, collections...)
}

func (n *Node) GetAllReplicators(ctx context.Context) ([]client.Replicator, error) {
	return n.peer.GetAllReplicators(ctx)
}

func (n *Node) AddP2PCollections(ctx context.Context, collectionIDs []string) error {
	return n.peer.AddP2PCollections(ctx, collectionIDs)
}

func (n *Node) RemoveP2PCollections(ctx context.Context, collectionIDs []string) error {
	return n.peer.RemoveP2PCollections(ctx, collectionIDs)
}

func (n *Node) GetAllP2PCollections(ctx context.Context) ([]string, error) {
	return n.peer.GetAllP2PCollections(ctx)
}
