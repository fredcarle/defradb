# DefraDB P2P Example

This example demonstrates how DefraDB's P2P functionality works in the browser using the JavaScript P2P implementation (Helia + js-libp2p) to sync documents between peers.

## What This Example Shows

- Loading DefraDB compiled to WebAssembly in the browser
- Creating two independent DefraDB instances with P2P support
- Using the JavaScript P2P implementation (js-libp2p + Helia) instead of Go's libp2p
- Creating schemas and documents in one peer
- Connecting peers together
- Syncing documents between peers using DefraDB's P2P protocol and Helia's bitswap
- How the JavaScript blockstore integrates with DefraDB's IPLD storage

## Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                        Browser (WASM)                           │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  Peer 1                           Peer 2                       │
│  ┌──────────────┐                 ┌──────────────┐            │
│  │ DefraDB WASM │                 │ DefraDB WASM │            │
│  │   Client     │                 │   Client     │            │
│  └──────┬───────┘                 └──────┬───────┘            │
│         │                                │                     │
│         ▼                                ▼                     │
│  ┌──────────────┐                 ┌──────────────┐            │
│  │ JSHostAdapter│                 │ JSHostAdapter│            │
│  │ (Go→JS       │                 │ (Go→JS       │            │
│  │  Bridge)     │                 │  Bridge)     │            │
│  └──────┬───────┘                 └──────┬───────┘            │
│         │                                │                     │
│         ▼                                ▼                     │
│  ┌──────────────┐                 ┌──────────────┐            │
│  │  Peer (TS)   │◄──WebRTC/WS────►│  Peer (TS)   │            │
│  │  + Helia     │                 │  + Helia     │            │
│  │  + Bitswap   │                 │  + Bitswap   │            │
│  └──────────────┘                 └──────────────┘            │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

## Prerequisites

### 1. Build DefraDB WASM Binary

From the DefraDB repo root:

```bash
GOOS=js GOARCH=wasm go build -tags js -o js/p2p/example/defradb.wasm ./cmd/defradb
```

This creates the `defradb.wasm` binary that runs in the browser.

### 2. Install Dependencies

The example has its own package.json with dependencies:

```bash
cd js/p2p/example
npm install
```

## Running the Example

```bash
npm run dev
```

This will:
1. Build the parent P2P library (`@defradb/browser-p2p`)
2. Bundle the example TypeScript code with esbuild
3. Start an HTTP server at http://localhost:8080
4. Open the example in your browser

## How to Use

### Step-by-Step Guide

1. **Load WASM** - Click "Load DefraDB WASM" to initialize the DefraDB runtime
   - This loads the WASM binary and sets up the `window.defradb` API

2. **Initialize Peer 1** - Click "Initialize Peer 1 with DefraDB"
   - Creates a JavaScript P2P peer (Helia + js-libp2p)
   - Creates a DefraDB instance that uses this peer for P2P
   - The peer is injected via `window.defraP2PHost`

3. **Initialize Peer 2** - Click "Initialize Peer 2 with DefraDB"
   - Creates a second independent peer and DefraDB instance

4. **Create Schema (Peer 1)** - Click "Create User Schema" on Peer 1
   - Adds a User type schema to Peer 1's database

5. **Create Document (Peer 1)** - Enter a name and age, then click "Create Document"
   - Creates a User document in Peer 1's database
   - The document key is automatically copied to Peer 2

6. **Connect Peers** - Click "Connect to Peer 1" on Peer 2
   - Establishes a P2P connection between the two DefraDB instances
   - Required for document syncing

7. **Create Schema (Peer 2)** - Click "Create User Schema" on Peer 2
   - Both peers need the same schema to work with the same document types

8. **Sync Document (Peer 2)** - Click "Sync and Query Document"
   - Peer 2 calls `p2pSyncDocuments()` to fetch the document from Peer 1
   - The document is retrieved via Helia's bitswap protocol
   - Once synced, Peer 2 queries and displays the document

## What's Happening Under the Hood

### DefraDB WASM Initialization

1. The `@sourcenetwork/acp-js` library's `instantiate()` function loads the WASM binary
2. This sets up `window.defradb` with an `open()` function
3. `open()` creates a DefraDB node and returns a client

### P2P Integration

1. **Peer Creation**: A TypeScript `Peer` is created with js-libp2p + Helia
2. **Host Injection**: The peer is set to `window.defraP2PHost`
3. **DefraDB Initialization**: When `defradb.open()` is called, [node_p2p_js.go](../../../node/node_p2p_js.go) checks for `window.defraP2PHost`
4. **JSHostAdapter**: The JavaScript peer is wrapped in a Go adapter implementing the `client.Host` interface
5. **Blockstore Bridge**: The Go blockstore is exposed to JavaScript, and Helia wraps it with bitswap

### Document Syncing

1. **Create**: Peer 1 creates a document, storing it as IPLD blocks in its blockstore
2. **Connect**: Peers establish a libp2p connection
3. **Sync**: Peer 2 calls `p2pSyncDocuments([docKey])`
4. **Fetch**: DefraDB requests the IPLD blocks for that document
5. **Bitswap**: Helia's bitswap protocol fetches blocks from Peer 1
6. **Store**: Blocks are stored in Peer 2's local blockstore
7. **Query**: Peer 2 can now query the document locally

## Files

- [index.html](./index.html) - UI for the two-peer demonstration
- [example.ts](./example.ts) - TypeScript code implementing the P2P logic
- [package.json](./package.json) - Build configuration and dependencies
- [README.md](./README.md) - This file

## Key Technologies

- **DefraDB**: Document database with P2P replication
- **WebAssembly**: Allows running Go code in the browser
- **js-libp2p**: JavaScript implementation of libp2p networking stack
- **Helia**: JavaScript IPFS implementation with bitswap protocol
- **Bitswap**: IPFS's block exchange protocol for P2P data transfer

## Troubleshooting

**WASM fails to load:**
- Make sure you've built the DefraDB WASM binary (see Prerequisites)
- Check that `defradb.wasm` exists in the example directory
- Look at browser console for detailed errors

**Peers can't connect:**
- Make sure both peers are initialized before trying to connect
- Check the browser console for detailed error messages
- Ensure you're running the example via HTTP (not file://)

**Document sync fails:**
- Ensure peers are connected first
- Verify both peers have the same schema
- Check that Peer 1 has actually created the document
- Look for errors in the browser console

**Build errors:**
- Run `npm install` in both the parent directory (`js/p2p`) and example directory
- Rebuild the parent library: `cd .. && npm run build`

## Integration with DefraDB

This example shows the complete integration between:
- DefraDB's Go codebase (compiled to WASM)
- The JavaScript P2P library (`@defradb/browser-p2p`)
- DefraDB's P2P document sync protocol
- Helia's bitswap protocol for block exchange

The key integration points are:
1. [js/js.go](../../../js/js.go) - Exposes `defradb.open()` to JavaScript
2. [node/node_p2p_js.go](../../../node/node_p2p_js.go) - Initializes P2P with JavaScript host
3. [js/host/adapter.go](../../../js/host/adapter.go) - Bridges JavaScript peer to Go's Host interface
4. [js/datastore/blockstore_js.go](../../../js/datastore/blockstore_js.go) - Exposes Go blockstore to JavaScript
5. [js/p2p/src/peer.ts](../src/peer.ts) - JavaScript P2P peer implementation
