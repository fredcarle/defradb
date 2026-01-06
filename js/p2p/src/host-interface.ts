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
 */

import { Block } from "@helia/bitswap/dist/src/pb/message";
import { Blockstore } from "interface-blockstore";

/**
 * Context represents execution context with optional timeout and cancellation.
 */
export interface Context {
  /** Deadline timestamp in milliseconds (Unix epoch) */
  deadline?: number;
  /** Cancellation signal */
  signal?: AbortSignal;
  /** Custom context values */
  values?: Record<string, any>;
}

/**
 * StreamHandler is called when a new stream is opened by a peer.
 * @param stream - Readable stream of data from the peer
 * @param peerID - ID of the peer that opened the stream
 */
export type StreamHandler = (stream: Uint8Array, peerID: string) => void;

/**
 * PubsubMessageHandler is called when a pubsub message is received.
 * @param from - Peer ID of the message sender
 * @param topic - Topic name
 * @param msg - Message data
 * @returns Response data or a Promise that resolves to response data
 */
export type PubsubMessageHandler = (
  from: string,
  topic: string,
  msg: Uint8Array
) => Promise<Uint8Array> | Uint8Array;

/**
 * BlockAccessFunc determines if a peer has access to a requested block.
 * @param peerID - ID of the requesting peer
 * @param cid - Content identifier of the requested block
 * @returns true if access is granted, false otherwise
 */
export type BlockAccessFunc = (peerID: string, cid: string) => boolean;

/**
 * PubsubResponse represents a response received from a pubsub message.
 */
export interface PubsubResponse {
  /** CID of the received message */
  id: string;
  /** ID of the sender */
  from: string;
  /** Message data */
  data: Uint8Array;
  /** Error from the sender, if any */
  error?: Error;
}

/**
 * IPLDStore interface for IPLD block storage.
 * This is a minimal interface - the actual implementation should
 * match the blockstore.IPLDStore interface from Go.
 */
export interface IPLDStore {
  // Add methods as needed based on blockstore.IPLDStore
  has(cid: string): Promise<boolean>;
  get(cid: string): Promise<Uint8Array | null>;
  put(cid: string, data: Uint8Array): Promise<void>;
}

/**
 * Host interface represents a P2P networking host.
 * This interface matches the client.Host interface from Go.
 */
export interface Host {
  /**
   * Returns the peer ID of the host.
   */
  id(): string;

  /**
   * Returns the host's list of multiaddresses.
   */
  addresses(): string[];

  /**
   * Returns the byte slice representation of the host's public key.
   */
  pubkey(): Uint8Array;

  /**
   * Tries to connect to the peer with the given addresses.
   * @param ctx - Execution context
   * @param addresses - List of multiaddresses to connect to
   */
  connect(addresses: string[]): Promise<void>;

  /**
   * Tries to disconnect from the peer with the given ID.
   * @param ctx - Execution context
   * @param peerID - ID of the peer to disconnect from
   */
  disconnect(peerID: string): Promise<void>;

  /**
   * Tries to send the given data to a peer.
   * @param ctx - Execution context
   * @param data - Data to send
   * @param peerID - ID of the peer to send to
   * @param protocolID - Protocol identifier
   */
  send(data: Uint8Array, peerID: string, protocolID: string): Promise<void>;

  /**
   * Returns a hash of the provided data signed with the private key of the host.
   * @param data - Data to sign
   */
  sign(data: Uint8Array): Promise<Uint8Array>;

  /**
   * Tells the host to listen for messages of the provided protocol ID and
   * handle them with the given handler.
   * @param protocolID - Protocol identifier
   * @param handler - Handler function for incoming streams
   */
  setStreamHandler(protocolID: string, handler: StreamHandler): void;

  /**
   * Adds a pubsub topic to the host.
   * @param topicName - Name of the topic
   * @param subscribe - Whether to subscribe to the topic
   * @param handler - Handler function for incoming messages
   */
  addPubSubTopic(topicName: string, subscribe: boolean, handler: PubsubMessageHandler): Promise<void>;

  /**
   * Removes the given topic from the host.
   * @param topic - Name of the topic to remove
   */
  removePubSubTopic(topic: string): Promise<void>;

  /**
   * Sends a new message on the given topic without waiting for a response.
   * @param ctx - Execution context
   * @param topic - Topic name
   * @param data - Message data
   */
  publishToTopicAsync(topic: string, data: Uint8Array): Promise<void>;

  /**
   * Sends a new message on the given topic, returning an async iterator of responses.
   * @param ctx - Execution context
   * @param topic - Topic name
   * @param data - Message data
   * @param withMultiResponse - Whether to allow responses from multiple peers
   */
  publishToTopic(
    topic: string,
    data: Uint8Array,
    withMultiResponse: boolean
  ): Promise<AsyncIterator<PubsubResponse>>;

  /**
   * Returns the host's IPLD store implementation.
   */
  ipldStore(): Blockstore;

  /**
   * Returns a new context with a session for the underlying block service.
   * @param ctx - Original context
   */
  contextWithSession(ctx: Context): Context;

  /**
   * Sets the function to use to determine if a peer has access to
   * the requested blocks on the block service.
   * @param accessFunc - Function to check block access permissions
   */
  setBlockAccessFunc(accessFunc: BlockAccessFunc): void;
}
