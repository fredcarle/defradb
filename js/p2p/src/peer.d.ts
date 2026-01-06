/**
 * Copyright 2025 Democratized Data Foundation
 *
 * Use of this software is governed by the Business Source License
 * included in the file licenses/BSL.txt.
 *
 * As of the Change Date specified in that file, in accordance with
 * the Business Source License, use of this software will be governed
 * by the Apache License, Version 2.0, included in the file
 * licenses/APL.txt.
 *
 * Peer implementation for DefraDB P2P in the browser.
 * Implements the Host interface from client/p2p.go using js-libp2p v3.
 */
import type { Blockstore } from 'interface-blockstore';
import type { Host, Context, StreamHandler, PubsubMessageHandler, BlockAccessFunc, PubsubResponse } from './host-interface';
/**
 * Peer implements the DefraDB Host interface using js-libp2p for browser-based P2P networking.
 * This allows DefraDB to run in the browser with full P2P capabilities via WebRTC and WebSockets.
 */
export declare class Peer implements Host {
    private helia;
    private pubsubEventListeners;
    private blockAccessFunc?;
    private constructor();
    /**
     * Creates and starts a new Peer instance.
     * This method configures libp2p similar to the Go implementation in go-p2p/host.go:
     * - Connection manager with limits (100-400 connections, 20s grace period)
     * - Dual DHT (WAN/LAN) with public key validator, concurrency 10, auto mode
     * - WebRTC and WebSocket transports for browser compatibility
     * - Circuit relay for NAT traversal
     * - Noise encryption and Yamux multiplexing
     * - GossipSub for pubsub messaging
     *
     * @param store - IPLD store implementation
     * @param options - Optional configuration (listen addresses, private key, relay settings)
     * @returns A started Peer instance
     */
    static create(store: Blockstore, options?: {
        listenAddresses?: string[];
        privateKey?: Uint8Array;
        isPrivateKeyProtobuf?: boolean;
        enableRelay?: boolean;
        client?: any;
    }): Promise<Peer>;
    /**
     * Closes the host and stops the underlying libp2p node.
     * Cleans up all pubsub subscriptions and event listeners.
     */
    close(): Promise<void>;
    /**
     * Returns the peer ID of this host.
     */
    id(): string;
    /**
     * Returns the multiaddresses this host is listening on.
     */
    addresses(): string[];
    /**
     * Returns the multiaddresses of the peers this host is connected to.
     */
    activePeers(): string[];
    /**
     * Returns the public key of this host as a byte array.
     */
    pubkey(): Uint8Array;
    /**
     * Connects to peers at the given multiaddresses.
     * @param addresses - Array of multiaddresses to connect to
     */
    connect(addresses: string[]): Promise<void>;
    /**
     * Disconnects from a peer with the given peer ID.
     * @param peerID - The peer ID to disconnect from
     */
    disconnect(peerID: string): Promise<void>;
    /**
     * Sends data to a peer using the specified protocol.
     * @param data - The data to send
     * @param peerID - The peer ID to send to
     * @param protocolID - The protocol ID to use
     */
    send(data: Uint8Array, peerID: string, protocolID: string): Promise<void>;
    /**
     * Signs data using this host's private key.
     * @param data - The data to sign
     * @returns The signature
     */
    sign(data: Uint8Array): Promise<Uint8Array>;
    /**
     * Sets a handler for incoming streams on a protocol.
     * @param protocolID - The protocol ID to handle
     * @param handler - The handler function
     */
    setStreamHandler(protocolID: string, handler: StreamHandler): void;
    /**
     * Adds a pubsub topic with an optional subscription and message handler.
     * @param topicName - The topic name
     * @param subscribe - Whether to subscribe to the topic
     * @param handler - The message handler function
     */
    addPubSubTopic(topicName: string, subscribe: boolean, handler: PubsubMessageHandler): Promise<void>;
    /**
     * Removes a pubsub topic and unsubscribes if subscribed.
     * @param topic - The topic to remove
     */
    removePubSubTopic(topic: string): Promise<void>;
    /**
     * Publishes data to a topic without waiting for responses.
     * @param topic - The topic to publish to
     * @param data - The data to publish
     */
    publishToTopicAsync(topic: string, data: Uint8Array): Promise<void>;
    /**
     * Publishes data to a topic and returns an async iterator for responses.
     * @param topic - The topic to publish to
     * @param data - The data to publish
     * @param withMultiResponse - Whether to wait for multiple responses
     * @returns An async iterator of responses
     */
    publishToTopic(topic: string, data: Uint8Array, withMultiResponse: boolean): Promise<AsyncIterator<PubsubResponse>>;
    /**
     * Returns the IPLD store instance with bitswap support.
     * This blockstore is backed by Helia's bitswap, which means it can
     * fetch blocks from remote peers over the network, similar to Go's
     * blockservice.IPLDStore wrapper.
     *
     * @returns A bitswap-enabled blockstore that can fetch from the network
     */
    ipldStore(): Blockstore;
    /**
     * Creates a new context with a session ID for block service operations.
     * @param ctx - The parent context
     * @returns A new context with session information
     */
    contextWithSession(ctx: Context): Context;
    /**
     * Sets the function to use for determining block access permissions.
     * @param accessFunc - The access control function
     */
    setBlockAccessFunc(accessFunc: BlockAccessFunc): void;
    /**
     * Checks if a peer has access to a block (internal helper).
     * @param ctx - The context
     * @param peerID - The peer ID requesting access
     * @param cid - The CID of the block
     * @returns true if access is allowed, false otherwise
     */
    hasBlockAccess(peerID: string, cid: string): boolean;
}
