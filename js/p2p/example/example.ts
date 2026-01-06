// Copyright 2025 Democratized Data Foundation
//
// Use of this software is governed by the Business Source License
// included in the file licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with
// the Business Source License, use of this software will be governed
// by the Apache License, Version 2.0, included in the file
// licenses/APL.txt.

import { Peer } from '../src/peer.js';
import { instantiate } from '@sourcenetwork/acp-js';

// Add declarations for globals
declare global {
  interface Window {
    Peer: typeof Peer;
    defradb: {
      open: () => Promise<any>;
    };
  }
}

// Global DefraDB clients
let db1: any = null;
let db2: any = null;

// Status update helpers
function updateStatus(peerId: string, message: string, type: 'info' | 'success' | 'error' = 'info') {
  const statusEl = document.getElementById(`${peerId}-status`);
  if (statusEl) {
    statusEl.textContent = message;
    statusEl.className = `status ${type}`;
  }
  console.log(`[${peerId}] ${message}`);
}

function updatePeerInfo(peerId: string, dbClient: any) {
  const infoEl = document.getElementById(`${peerId}-info`);
  if (infoEl) {
    // Get peer info from DefraDB
    dbClient.getPeerInfo().then((peerInfo: any) => {
      console.log(peerInfo)
      infoEl.innerHTML = `
        <strong>Peer ID:</strong> ${peerInfo.ID}<br>
        <strong>Addresses:</strong><br>
        ${peerInfo.map((addr: string) => `  ${addr}`).join('<br>')}<br>
        <strong>DefraDB:</strong> Ready
      `;
      infoEl.style.display = 'block';
    }).catch((err: any) => {
      console.error('Failed to get peer info:', err);
    });
  }
}

// Load WASM and expose Peer class
async function loadWasm() {
  updateStatus('peer1', 'Loading DefraDB WASM...');
  updateStatus('peer2', 'Loading DefraDB WASM...');

  try {
    // Expose the Peer class globally so DefraDB's Go code can call Peer.create()
    window.Peer = Peer;

    // Use the acp-js instantiate function to load DefraDB WASM
    await instantiate('defradb.wasm');

    // Wait a bit for DefraDB to initialize
    await new Promise(resolve => setTimeout(resolve, 100));

    if (!window.defradb) {
      throw new Error('DefraDB WASM failed to initialize');
    }

    updateStatus('peer1', 'DefraDB WASM loaded successfully', 'success');
    updateStatus('peer2', 'DefraDB WASM loaded successfully', 'success');

    // Enable init buttons
    const init1Btn = document.getElementById('peer1-init') as HTMLButtonElement;
    const init2Btn = document.getElementById('peer2-init') as HTMLButtonElement;
    if (init1Btn) init1Btn.disabled = false;
    if (init2Btn) init2Btn.disabled = false;

  } catch (error) {
    updateStatus('peer1', `WASM load error: ${error}`, 'error');
    updateStatus('peer2', `WASM load error: ${error}`, 'error');
    throw error;
  }
}

// Initialize Peer 1 with DefraDB
async function initPeer1() {
  try {
    updateStatus('peer1', 'Initializing DefraDB with P2P support...');

    // Open DefraDB - the Go code will automatically call Peer.create() from window.Peer
    // and inject the P2P host
    db1 = await window.defradb.open();

    updateStatus('peer1', 'Peer 1 and DefraDB initialized successfully', 'success');
    updatePeerInfo('peer1', db1);

    // Enable buttons
    const schemaBtn = document.getElementById('peer1-schema') as HTMLButtonElement;
    if (schemaBtn) schemaBtn.disabled = false;

    // Enable connect button if peer2 exists
    const connectBtn = document.getElementById('peer2-connect') as HTMLButtonElement;
    if (connectBtn && db2) connectBtn.disabled = false;

    // Enable relay connection and peer info buttons
    const relayBtn = document.getElementById('peer1-connect-relay') as HTMLButtonElement;
    const infoBtn = document.getElementById('peer1-info-btn') as HTMLButtonElement;
    if (relayBtn) relayBtn.disabled = false;
    if (infoBtn) infoBtn.disabled = false;

  } catch (error) {
    updateStatus('peer1', `Error: ${error}`, 'error');
    console.error('Peer 1 initialization error:', error);
  }
}

// Connect Peer 1 to relay
async function connectRelay1() {
  if (!db1) {
    updateStatus('peer1', 'DefraDB not initialized', 'error');
    return;
  }

  try {
    const relayInput = document.getElementById('peer1-relay') as HTMLInputElement;
    const relayAddr = relayInput?.value;

    if (!relayAddr) {
      updateStatus('peer1', 'Please enter a relay address', 'error');
      return;
    }

    updateStatus('peer1', 'Connecting to relay...');

    // Connect to the relay peer
    await db1.p2pConnect(null, [relayAddr]);

    updateStatus('peer1', 'Connected to relay successfully!', 'success');

  } catch (error) {
    updateStatus('peer1', `Relay connection error: ${error}`, 'error');
    console.error('Relay connection error:', error);
  }
}

// Get Peer 1 info
async function getPeerInfo1() {
  if (!db1) {
    updateStatus('peer1', 'DefraDB not initialized', 'error');
    return;
  }

  try {
    updateStatus('peer1', 'Fetching peer info...');

    const peerInfo = await db1.getPeerInfo();

    updateStatus('peer1', 'Peer info retrieved successfully!', 'success');

    // Update the peer info display
    updatePeerInfo('peer1', db1);

  } catch (error) {
    updateStatus('peer1', `Error fetching peer info: ${error}`, 'error');
    console.error('Peer info error:', error);
  }
}

// Initialize Peer 2 with DefraDB
async function initPeer2() {
  try {
    updateStatus('peer2', 'Initializing DefraDB with P2P support...');

    // Open DefraDB - the Go code will automatically call Peer.create()
    db2 = await window.defradb.open();

    updateStatus('peer2', 'Peer 2 and DefraDB initialized successfully', 'success');
    updatePeerInfo('peer2', db2);

    // Enable connect button if peer1 exists
    const connectBtn = document.getElementById('peer2-connect') as HTMLButtonElement;
    if (connectBtn && db1) connectBtn.disabled = false;

    // Enable relay connection and peer info buttons
    const relayBtn = document.getElementById('peer2-connect-relay') as HTMLButtonElement;
    const infoBtn = document.getElementById('peer2-info-btn') as HTMLButtonElement;
    if (relayBtn) relayBtn.disabled = false;
    if (infoBtn) infoBtn.disabled = false;

  } catch (error) {
    updateStatus('peer2', `Error: ${error}`, 'error');
    console.error('Peer 2 initialization error:', error);
  }
}

// Connect Peer 2 to relay
async function connectRelay2() {
  if (!db2) {
    updateStatus('peer2', 'DefraDB not initialized', 'error');
    return;
  }

  try {
    const relayInput = document.getElementById('peer2-relay') as HTMLInputElement;
    const relayAddr = relayInput?.value;

    if (!relayAddr) {
      updateStatus('peer2', 'Please enter a relay address', 'error');
      return;
    }

    updateStatus('peer2', 'Connecting to relay...');

    // Connect to the relay peer
    await db2.p2pConnect(null, [relayAddr]);

    updateStatus('peer2', 'Connected to relay successfully!', 'success');

  } catch (error) {
    updateStatus('peer2', `Relay connection error: ${error}`, 'error');
    console.error('Relay connection error:', error);
  }
}

// Get Peer 2 info
async function getPeerInfo2() {
  if (!db2) {
    updateStatus('peer2', 'DefraDB not initialized', 'error');
    return;
  }

  try {
    updateStatus('peer2', 'Fetching peer info...');

    const peerInfo = await db2.getPeerInfo();

    updateStatus('peer2', 'Peer info retrieved successfully!', 'success');

    // Update the peer info display
    updatePeerInfo('peer2', db2);

  } catch (error) {
    updateStatus('peer2', `Error fetching peer info: ${error}`, 'error');
    console.error('Peer info error:', error);
  }
}

// Create schema on Peer 1
async function createSchema1() {
  if (!db1) {
    updateStatus('peer1', 'DefraDB not initialized', 'error');
    return;
  }

  try {
    updateStatus('peer1', 'Creating schema...');

    const schema = `
      type User {
        name: String
        age: Int
        verified: Boolean
        points: Int @crdt(type: pncounter)
      }
    `;

    await db1.addSchema(schema);

    updateStatus('peer1', 'Schema created successfully!', 'success');

    // Enable create doc button
    const createBtn = document.getElementById('peer1-create') as HTMLButtonElement;
    if (createBtn) createBtn.disabled = false;

  } catch (error) {
    updateStatus('peer1', `Schema error: ${error}`, 'error');
    console.error('Schema error:', error);
  }
}

// Create document on Peer 1
async function createDocument1() {
  if (!db1) {
    updateStatus('peer1', 'DefraDB not initialized', 'error');
    return;
  }

  try {
    const nameEl = document.getElementById('peer1-name') as HTMLInputElement;
    const ageEl = document.getElementById('peer1-age') as HTMLInputElement;

    const name = nameEl?.value || 'Alice';
    const age = parseInt(ageEl?.value || '30');

    updateStatus('peer1', 'Creating document...');

    // Create the document
    const mutation = `
      mutation {
        create_User(input: {name: "${name}", age: ${age}}) {
          _docID
          name
          age
        }
      }
    `;

    const result = await db1.execRequest(mutation);

    if (result.gql.errors && result.gql.errors.length > 0) {
      throw new Error(result.gql.errors[0].message);
    }
    console.log(result.gql)
    const doc = result.gql.data.create_User[0];

    updateStatus('peer1', `Document created successfully!`, 'success');

    // Display document info
    const docEl = document.getElementById('peer1-doc');
    if (docEl) {
      docEl.innerHTML = `
        <strong>Key:</strong> ${doc._docID}<br>
        <strong>Name:</strong> ${doc.name}<br>
        <strong>Age:</strong> ${doc.age}
      `;
      docEl.style.display = 'block';
    }

    // Auto-fill key in peer2
    const keyInput = document.getElementById('peer2-key') as HTMLInputElement;
    if (keyInput) {
      keyInput.value = doc._docID;
    }

  } catch (error) {
    updateStatus('peer1', `Error: ${error}`, 'error');
    console.error('Create document error:', error);
  }
}

// Connect peers
async function connectPeers() {
  if (!db1 || !db2) {
    updateStatus('peer2', 'Both DBs must be initialized', 'error');
    return;
  }

  try {
    updateStatus('peer2', 'Connecting to Peer 1...');

    // Get peer1 info to get addresses
    const peer1Info = await db1.getPeerInfo();

    if (!peer1Info.Addrs || peer1Info.Addrs.length === 0) {
      updateStatus('peer2', 'Peer 1 has no addresses', 'error');
      return;
    }

    // Connect peer2 to peer1 via DefraDB's P2P layer
    await db2.p2pConnect(null, peer1Info.Addrs);

    updateStatus('peer2', 'Connected to Peer 1 successfully!', 'success');

    // Enable schema and query buttons
    const schemaBtn = document.getElementById('peer2-schema') as HTMLButtonElement;
    const queryBtn = document.getElementById('peer2-query') as HTMLButtonElement;
    if (schemaBtn) schemaBtn.disabled = false;
    if (queryBtn) queryBtn.disabled = false;

  } catch (error) {
    updateStatus('peer2', `Connection error: ${error}`, 'error');
    console.error('Connection error:', error);
  }
}

// Add schema to Peer 2
async function createSchema2() {
  if (!db2) {
    updateStatus('peer2', 'DefraDB not initialized', 'error');
    return;
  }

  try {
    updateStatus('peer2', 'Creating schema...');

    const schema = `
      type User {
        name: String
        age: Int
        verified: Boolean
        points: Int @crdt(type: pncounter)
      }
    `;

    await db2.addSchema(schema);

    updateStatus('peer2', 'Schema created successfully!', 'success');

  } catch (error) {
    updateStatus('peer2', `Schema error: ${error}`, 'error');
    console.error('Schema error:', error);
  }
}

// Query document on Peer 2
async function queryDocument2() {
  if (!db2) {
    updateStatus('peer2', 'DefraDB not initialized', 'error');
    return;
  }

  try {
    const keyInput = document.getElementById('peer2-key') as HTMLInputElement;
    const key = keyInput?.value;

    if (!key) {
      updateStatus('peer2', 'Please enter a document key', 'error');
      return;
    }

    updateStatus('peer2', 'Syncing document from network...');

    // First, sync the document from the network
    await db2.p2pSyncDocuments(null, 'User', [key]);

    updateStatus('peer2', 'Querying synced document...');

    // Now query it
    const query = `
      query {
        User(filter: {_docID: {_eq: "${key}"}}) {
          _docID
          name
          age
        }
      }
    `;

    const result = await db2.execRequest(query);

    if (result.errors && result.errors.length > 0) {
      throw new Error(result.errors[0].message);
    }

    const docs = result.data.User;

    if (docs.length === 0) {
      updateStatus('peer2', 'Document not found', 'error');
      return;
    }

    const doc = docs[0];

    updateStatus('peer2', `Document retrieved successfully from network!`, 'success');

    // Display document info
    const docEl = document.getElementById('peer2-doc');
    if (docEl) {
      docEl.innerHTML = `
        <strong>Key:</strong> ${doc._docID}<br>
        <strong>Name:</strong> ${doc.name}<br>
        <strong>Age:</strong> ${doc.age}<br>
        <strong>Source:</strong> Synced from Peer 1 via P2P
      `;
      docEl.style.display = 'block';
    }

  } catch (error) {
    updateStatus('peer2', `Error: ${error}`, 'error');
    console.error('Query error:', error);
  }
}

// Make functions available globally
(window as any).loadWasm = loadWasm;
(window as any).initPeer1 = initPeer1;
(window as any).initPeer2 = initPeer2;
(window as any).connectRelay1 = connectRelay1;
(window as any).connectRelay2 = connectRelay2;
(window as any).getPeerInfo1 = getPeerInfo1;
(window as any).getPeerInfo2 = getPeerInfo2;
(window as any).createSchema1 = createSchema1;
(window as any).createDocument1 = createDocument1;
(window as any).connectPeers = connectPeers;
(window as any).createSchema2 = createSchema2;
(window as any).queryDocument2 = queryDocument2;

// Auto-load WASM on page load
updateStatus('peer1', 'Ready - click "Load WASM" to start');
updateStatus('peer2', 'Ready - click "Load WASM" to start');
