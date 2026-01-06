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
import { createHelia } from 'helia';
import { createBitswap } from '@helia/bitswap';
import { kadDHT } from '@libp2p/kad-dht';
import { ping } from '@libp2p/ping';
import { identify } from '@libp2p/identify';
import { gossipsub } from '@libp2p/gossipsub';
import { noise } from '@chainsafe/libp2p-noise';
import { yamux } from '@chainsafe/libp2p-yamux';
import { webSockets } from '@libp2p/websockets';
import { webRTC } from '@libp2p/webrtc';
import { circuitRelayTransport } from '@libp2p/circuit-relay-v2';
import { multiaddr } from '@multiformats/multiaddr';
import { privateKeyFromProtobuf, // For protobuf-encoded keys
privateKeyFromRaw, // For raw private key bytes
 } from '@libp2p/crypto/keys';
/**
 * Peer implements the DefraDB Host interface using js-libp2p for browser-based P2P networking.
 * This allows DefraDB to run in the browser with full P2P capabilities via WebRTC and WebSockets.
 */
export class Peer {
    constructor(helia) {
        this.pubsubEventListeners = new Map();
        this.helia = helia;
    }
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
    static async create(store, options) {
        console.log('!!! PEER.CREATE CALLED !!!');
        console.log('Creating Peer with options:', options);
        // Convert private key from protobuf format if provided
        let privateKey;
        if (options?.privateKey) {
            if (options.isPrivateKeyProtobuf) {
                privateKey = privateKeyFromProtobuf(options.privateKey);
            }
            else {
                // By default, assume it's a raw key
                privateKey = privateKeyFromRaw(options.privateKey);
            }
        }
        console.log('Creating Helia with bitswap blockBroker...');
        // Enable debug logging for Helia, bitswap, and libp2p
        if (typeof window !== 'undefined' && window.localStorage) {
            // Enable all debug logs - you can filter to specific modules if needed
            window.localStorage.setItem('debug', '*');
            console.log('🐛 Debug logging enabled for all Helia/libp2p/bitswap components');
        }
        // Pass the raw blockstore to Helia
        const helia = await createHelia({
            blockstore: store,
            blockBrokers: [
                (components) => {
                    console.log('Bitswap blockBroker factory called');
                    // CRITICAL: Use our original store directly for bitswap, not Helia's wrapper
                    const bitswapComponents = {
                        ...components,
                        blockstore: store
                    };
                    const bitswapInstance = createBitswap(bitswapComponents, {
                        // Prevent stream timeouts that cause half-closed stream issues
                        // Set to ~24 days (max safe timeout value)
                        // @ts-ignore
                        messageReceiveTimeout: 2147483647,
                    });
                    console.log('Bitswap instance created successfully');
                    // Start bitswap
                    const startResult = bitswapInstance.start();
                    if (startResult && typeof startResult.then === 'function') {
                        startResult.then(() => {
                            console.log('✓ Bitswap started and ready');
                        }).catch((err) => {
                            console.error('✗ Bitswap start failed:', err);
                        });
                    }
                    return bitswapInstance;
                }
            ],
            libp2p: {
                // Addresses to listen on - empty by default to prevent automatic relay connections
                addresses: {
                    listen: options?.listenAddresses || []
                },
                // Transports: WebRTC and WebSockets for browser, plus circuit relay
                transports: [
                    webSockets(),
                    webRTC(),
                    // support dialing/listening on Circuit Relay addresses (manual connections only)
                    circuitRelayTransport(),
                ],
                // Connection encryption (Noise protocol)
                connectionEncrypters: [noise()],
                // Stream multiplexing (Yamux)
                streamMuxers: [yamux()],
                connectionGater: {
                    denyDialMultiaddr: () => {
                        // by default we refuse to dial local addresses from browsers since they
                        // are usually sent by remote peers broadcasting undialable multiaddrs and
                        // cause errors to appear in the console but in this example we are
                        // explicitly connecting to a local node so allow all addresses
                        return false;
                    }
                },
                connectionManager: {
                    maxConnections: 400,
                    maxParallelDials: 100,
                    dialTimeout: 30000, // 30 seconds
                },
                // Disable automatic peer discovery to prevent connecting to random nodes
                peerDiscovery: [],
                services: {
                    dht: kadDHT({
                        clientMode: true, // Client mode only - prevents automatic bootstrapping to public DHT nodes
                    }),
                    identify: identify(),
                    ping: ping(),
                    pubsub: gossipsub({
                        allowPublishToZeroTopicPeers: true,
                        emitSelf: false,
                    })
                },
            },
            // Use provided private key if available
            ...(privateKey && { privateKey })
        });
        // Note: createLibp2p starts the node by default (unless start: false is passed)
        console.log('Libp2p started with peer ID:', helia.libp2p.peerId.toString());
        // Log registered protocols
        console.log('Registered libp2p protocols:', helia.libp2p.getProtocols());
        return new Peer(helia);
        // Return a plain object with all methods bound to the peer instance
        // This is necessary for Go's syscall/js to properly access the methods
        // since it can't traverse the prototype chain
        //   return {
        //     close: peer.close.bind(peer),
        //     id: peer.id.bind(peer),
        //     addresses: peer.addresses.bind(peer),
        //     pubkey: peer.pubkey.bind(peer),
        //     connect: peer.connect.bind(peer),
        //     disconnect: peer.disconnect.bind(peer),
        //     send: peer.send.bind(peer),
        //     sign: peer.sign.bind(peer),
        //     setStreamHandler: peer.setStreamHandler.bind(peer),
        //     addPubSubTopic: peer.addPubSubTopic.bind(peer),
        //     removePubSubTopic: peer.removePubSubTopic.bind(peer),
        //     publishToTopicAsync: peer.publishToTopicAsync.bind(peer),
        //     publishToTopic: peer.publishToTopic.bind(peer),
        //     ipldStore: peer.ipldStore.bind(peer),
        //     contextWithSession: peer.contextWithSession.bind(peer),
        //     setBlockAccessFunc: peer.setBlockAccessFunc.bind(peer),
        //   } as any as Peer;
    }
    /**
     * Closes the host and stops the underlying libp2p node.
     * Cleans up all pubsub subscriptions and event listeners.
     */
    async close() {
        // Clean up all pubsub listeners
        const pubsub = this.helia.libp2p.services.pubsub;
        if (pubsub) {
            for (const [topic, listener] of this.pubsubEventListeners.entries()) {
                pubsub.removeEventListener('message', listener);
                try {
                    pubsub.unsubscribe(topic);
                }
                catch (err) {
                    console.warn(`Failed to unsubscribe from topic ${topic}:`, err);
                }
            }
        }
        this.pubsubEventListeners.clear();
        await this.helia.stop();
    }
    /**
     * Returns the peer ID of this host.
     */
    id() {
        return this.helia.libp2p.peerId.toString();
    }
    /**
     * Returns the multiaddresses this host is listening on.
     */
    addresses() {
        console.log('Getting listening addresses');
        return this.helia.libp2p.getMultiaddrs().map((ma) => ma.toString());
    }
    /**
     * Returns the multiaddresses of the peers this host is connected to.
     */
    activePeers() {
        console.log('Getting peer addresses');
        return this.helia.libp2p.getConnections().map((conn) => conn.remoteAddr.toString());
    }
    /**
     * Returns the public key of this host as a byte array.
     */
    pubkey() {
        const pubkey = this.helia.libp2p.peerId.publicKey;
        if (!pubkey) {
            throw new Error('Public key not available');
        }
        return pubkey.raw;
    }
    /**
     * Connects to peers at the given multiaddresses.
     * @param addresses - Array of multiaddresses to connect to
     */
    async connect(addresses) {
        if (!addresses || addresses.length === 0) {
            throw new Error('No addresses provided');
        }
        const errors = [];
        for (const addr of addresses) {
            try {
                console.log(`Dialing peer at address: ${addr}`);
                await this.helia.libp2p.dial(multiaddr(addr));
            }
            catch (err) {
                errors.push(err instanceof Error ? err : new Error(String(err)));
            }
        }
        // If all connections failed, throw the first error
        if (errors.length === addresses.length && errors.length > 0) {
            throw new Error(`Failed to connect to any peer: ${errors[0].message}`);
        }
    }
    /**
     * Disconnects from a peer with the given peer ID.
     * @param peerID - The peer ID to disconnect from
     */
    async disconnect(peerID) {
        if (!peerID) {
            throw new Error('Peer ID is required');
        }
        const connections = this.helia.libp2p.getConnections();
        let disconnected = false;
        for (const conn of connections) {
            if (conn.remotePeer.toString() === peerID) {
                await conn.close();
                disconnected = true;
            }
        }
        if (!disconnected) {
            throw new Error(`No connection found for peer ${peerID}`);
        }
    }
    /**
     * Sends data to a peer using the specified protocol.
     * @param data - The data to send
     * @param peerID - The peer ID to send to
     * @param protocolID - The protocol ID to use
     */
    async send(data, peerID, protocolID) {
        if (!data || data.length === 0) {
            throw new Error('Data is required');
        }
        if (!peerID) {
            throw new Error('Peer ID is required');
        }
        if (!protocolID) {
            throw new Error('Protocol ID is required');
        }
        const peer = this.helia.libp2p.getPeers().find(p => p.toString() === peerID);
        if (!peer) {
            throw new Error(`Peer not found: ${peerID}`);
        }
        const stream = await this.helia.libp2p.dialProtocol(peer, protocolID);
        stream.send(data);
    }
    /**
     * Signs data using this host's private key.
     * @param data - The data to sign
     * @returns The signature
     */
    async sign(data) {
        if (!data || data.length === 0) {
            throw new Error('Data is required for signing');
        }
        // Access the private key from libp2p components
        // This matches the Go implementation: p.host.Peerstore().PrivKey(p.host.ID()).Sign(data)
        // Note: components is available but not exposed in the public interface, so we cast
        const privateKey = this.helia.libp2p.components?.privateKey;
        if (!privateKey) {
            throw new Error('Private key not available for signing');
        }
        return await privateKey.sign(data);
    }
    /**
     * Sets a handler for incoming streams on a protocol.
     * @param protocolID - The protocol ID to handle
     * @param handler - The handler function
     */
    setStreamHandler(protocolID, handler) {
        console.log(`Setting stream handler for protocol: ${protocolID}`);
        if (!protocolID) {
            throw new Error('Protocol ID is required');
        }
        if (!handler) {
            throw new Error('Handler is required');
        }
        // Register with libp2p
        this.helia.libp2p.handle(protocolID, (stream, connection) => {
            const remotePeer = connection.remotePeer.toString();
            stream.addEventListener('message', (evt) => {
                // Call the handler
                try {
                    handler(evt.data.subarray(), remotePeer);
                }
                catch (err) {
                    console.error(`Error in stream handler for protocol ${protocolID}:`, err);
                }
            });
        });
    }
    /**
     * Adds a pubsub topic with an optional subscription and message handler.
     * @param topicName - The topic name
     * @param subscribe - Whether to subscribe to the topic
     * @param handler - The message handler function
     */
    async addPubSubTopic(topicName, subscribe, handler) {
        if (!topicName) {
            throw new Error('Topic name is required');
        }
        if (!handler) {
            throw new Error('Handler is required');
        }
        const pubsub = this.helia.libp2p.services.pubsub;
        if (!pubsub) {
            throw new Error('PubSub service not configured');
        }
        if (subscribe) {
            // Create event listener
            const listener = async (evt) => {
                if (evt.detail.topic !== topicName) {
                    return;
                }
                const from = evt.detail.from ? evt.detail.from.toString() : '';
                const data = evt.detail.data;
                try {
                    await handler(from, topicName, data);
                }
                catch (error) {
                    console.error(`Error in pubsub handler for topic ${topicName}:`, error);
                }
            };
            this.pubsubEventListeners.set(topicName, listener);
            // Subscribe and add listener
            pubsub.subscribe(topicName);
            pubsub.addEventListener('message', listener);
        }
    }
    /**
     * Removes a pubsub topic and unsubscribes if subscribed.
     * @param topic - The topic to remove
     */
    async removePubSubTopic(topic) {
        if (!topic) {
            throw new Error('Topic name is required');
        }
        const pubsub = this.helia.libp2p.services.pubsub;
        if (!pubsub) {
            throw new Error('PubSub service not configured');
        }
        // Remove event listener if it exists
        const listener = this.pubsubEventListeners.get(topic);
        if (listener) {
            pubsub.removeEventListener('message', listener);
            this.pubsubEventListeners.delete(topic);
        }
        // Unsubscribe
        try {
            pubsub.unsubscribe(topic);
        }
        catch (err) {
            console.warn(`Failed to unsubscribe from topic ${topic}:`, err);
        }
    }
    /**
     * Publishes data to a topic without waiting for responses.
     * @param topic - The topic to publish to
     * @param data - The data to publish
     */
    async publishToTopicAsync(topic, data) {
        console.log(`Publishing to topic asynchronously: ${topic}`);
        if (!topic) {
            throw new Error('Topic name is required');
        }
        if (!data) {
            throw new Error('Data is required');
        }
        const pubsub = this.helia.libp2p.services.pubsub;
        if (!pubsub) {
            throw new Error('PubSub service not configured');
        }
        await pubsub.publish(topic, data);
    }
    /**
     * Publishes data to a topic and returns an async iterator for responses.
     * @param topic - The topic to publish to
     * @param data - The data to publish
     * @param withMultiResponse - Whether to wait for multiple responses
     * @returns An async iterator of responses
     */
    async publishToTopic(topic, data, withMultiResponse) {
        console.log(`Publishing to topic: ${topic}, withMultiResponse: ${withMultiResponse}`);
        if (!topic) {
            throw new Error('Topic name is required');
        }
        if (!data) {
            throw new Error('Data is required');
        }
        const pubsub = this.helia.libp2p.services.pubsub;
        if (!pubsub) {
            throw new Error('PubSub service not configured');
        }
        // Create response queue and state
        const responseQueue = [];
        let isDone = false;
        let resolveNext = null;
        // Create unique response topic
        const responseTopic = `${topic}/response/${this.id()}-${Date.now()}-${Math.random()}`;
        // Response listener
        const responseListener = (evt) => {
            if (evt.detail.topic !== responseTopic) {
                return;
            }
            const response = {
                id: evt.detail.id || '',
                from: evt.detail.from ? evt.detail.from.toString() : '',
                data: evt.detail.data,
            };
            responseQueue.push(response);
            if (resolveNext) {
                const resolve = resolveNext;
                resolveNext = null;
                resolve({ value: response, done: false });
            }
            if (!withMultiResponse) {
                cleanup();
            }
        };
        // Cleanup function
        const cleanup = () => {
            if (isDone)
                return;
            isDone = true;
            try {
                pubsub.removeEventListener('message', responseListener);
                pubsub.unsubscribe(responseTopic);
            }
            catch (err) {
                console.warn('Error during pubsub cleanup:', err);
            }
            if (resolveNext) {
                resolveNext({ value: undefined, done: true });
                resolveNext = null;
            }
        };
        // Subscribe to response topic
        pubsub.subscribe(responseTopic);
        pubsub.addEventListener('message', responseListener);
        // Publish the message
        try {
            await pubsub.publish(topic, data);
        }
        catch (err) {
            cleanup();
            throw err;
        }
        // Set timeout (30 seconds)
        const timeoutId = setTimeout(cleanup, 30000);
        // Return async iterator
        return {
            next: async () => {
                if (responseQueue.length > 0) {
                    return { value: responseQueue.shift(), done: false };
                }
                if (isDone) {
                    clearTimeout(timeoutId);
                    return { value: undefined, done: true };
                }
                return new Promise((resolve) => {
                    resolveNext = resolve;
                });
            },
            // Optional return method for cleanup when iteration stops early
            return: async () => {
                clearTimeout(timeoutId);
                cleanup();
                return { value: undefined, done: true };
            }
        };
    }
    /**
     * Returns the IPLD store instance with bitswap support.
     * This blockstore is backed by Helia's bitswap, which means it can
     * fetch blocks from remote peers over the network, similar to Go's
     * blockservice.IPLDStore wrapper.
     *
     * @returns A bitswap-enabled blockstore that can fetch from the network
     */
    ipldStore() {
        // Return Helia's blockstore, which is the local store wrapped with bitswap.
        // When a block is not found locally, Helia will automatically fetch it
        // from remote peers using the bitswap protocol.
        return this.helia.blockstore;
    }
    /**
     * Creates a new context with a session ID for block service operations.
     * @param ctx - The parent context
     * @returns A new context with session information
     */
    contextWithSession(ctx) {
        return {
            ...ctx,
            values: {
                ...(ctx.values || {}),
                sessionId: `session-${this.id()}-${Date.now()}`,
            },
        };
    }
    /**
     * Sets the function to use for determining block access permissions.
     * @param accessFunc - The access control function
     */
    setBlockAccessFunc(accessFunc) {
        this.blockAccessFunc = accessFunc;
    }
    /**
     * Checks if a peer has access to a block (internal helper).
     * @param ctx - The context
     * @param peerID - The peer ID requesting access
     * @param cid - The CID of the block
     * @returns true if access is allowed, false otherwise
     */
    hasBlockAccess(peerID, cid) {
        if (!this.blockAccessFunc) {
            return true; // Default: allow all access
        }
        return this.blockAccessFunc(peerID, cid);
    }
}
