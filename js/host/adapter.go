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

package host

import (
	"context"
	"fmt"
	"io"
	"syscall/js"

	"github.com/ipfs/go-cid"
	"github.com/sourcenetwork/corekv/blockstore"
	"github.com/sourcenetwork/goji"

	"github.com/sourcenetwork/defradb/client"
)

// JSHostAdapter wraps a JavaScript Host implementation to satisfy the client.Host interface.
// This allows browser-based TypeScript implementations (using js-libp2p) to be injected
// into the Go WASM DefraDB core.
type JSHostAdapter struct {
	jsHost js.Value
}

// NewJSHostAdapter creates a new adapter that wraps a JavaScript Host object.
func NewJSHostAdapter(jsHost js.Value) *JSHostAdapter {
	return &JSHostAdapter{
		jsHost: jsHost,
	}
}

// ID returns the peer ID of the host.
func (h *JSHostAdapter) ID() string {
	idFunc := h.jsHost.Get("id")
	if !idFunc.Truthy() {
		return ""
	}
	result := h.jsHost.Call("id")
	return result.String()
}

// ActivePeers returns the addresses of peers that are currently connected to.
func (h *JSHostAdapter) ActivePeers() ([]string, error) {
	addrsFunc := h.jsHost.Get("activePeers")
	if !addrsFunc.Truthy() {
		fmt.Println("activePeers", "fail")
		return nil, ErrMissingHostMethod
	}

	result := h.jsHost.Call("activePeers")

	var addresses []string
	if err := goji.UnmarshalJS(result, &addresses); err != nil {
		return nil, err
	}
	return addresses, nil
}

// Addresses returns the host's list of addresses.
func (h *JSHostAdapter) Addresses() ([]string, error) {
	addrsFunc := h.jsHost.Get("addresses")
	if !addrsFunc.Truthy() {
		fmt.Println("addresses", "fail")
		return nil, ErrMissingHostMethod
	}

	result := h.jsHost.Call("addresses")

	var addresses []string
	if err := goji.UnmarshalJS(result, &addresses); err != nil {
		return nil, err
	}
	return addresses, nil
}

// Pubkey returns the byte slice representation of the host's public key.
func (h *JSHostAdapter) Pubkey() ([]byte, error) {
	pubkeyFunc := h.jsHost.Get("pubkey")
	if !pubkeyFunc.Truthy() {
		fmt.Println("pubkey", "fail")
		return nil, ErrMissingHostMethod
	}

	promise := h.jsHost.Call("pubkey")
	result, err := goji.Await(goji.PromiseValue(promise))
	if err != nil {
		return nil, err
	}

	// Handle Uint8Array from JavaScript
	uint8Array := result[0]
	length := uint8Array.Get("length").Int()
	data := make([]byte, length)
	js.CopyBytesToGo(data, uint8Array)
	return data, nil
}

// Connect tries to connect to the peer with the given addresses.
func (h *JSHostAdapter) Connect(ctx context.Context, addresses []string) error {
	connectFunc := h.jsHost.Get("connect")
	if !connectFunc.Truthy() {
		fmt.Println("connect", "fail")
		return ErrMissingHostMethod
	}

	jsAddrs, err := goji.MarshalJS(addresses)
	if err != nil {
		return err
	}

	promise := h.jsHost.Call("connect", jsAddrs)
	_, err = goji.Await(goji.PromiseValue(promise))
	return err
}

// Disconnect will try to disconnect from the peer with the given ID.
func (h *JSHostAdapter) Disconnect(ctx context.Context, peerID string) error {
	disconnectFunc := h.jsHost.Get("disconnect")
	if !disconnectFunc.Truthy() {
		fmt.Println("disconnect", "fail")
		return ErrMissingHostMethod
	}

	jsCtx, err := goji.MarshalJS(contextToJS(ctx))
	if err != nil {
		return err
	}

	promise := h.jsHost.Call("disconnect", jsCtx, peerID)
	_, err = goji.Await(goji.PromiseValue(promise))
	return err
}

// Send will try to send the given data to a peer.
func (h *JSHostAdapter) Send(ctx context.Context, data []byte, peerID string, protocolID string) error {
	sendFunc := h.jsHost.Get("send")
	if !sendFunc.Truthy() {
		fmt.Println("send", "fail")
		return ErrMissingHostMethod
	}

	jsCtx, err := goji.MarshalJS(contextToJS(ctx))
	if err != nil {
		return err
	}

	// Convert []byte to Uint8Array
	uint8Array := js.Global().Get("Uint8Array").New(len(data))
	js.CopyBytesToJS(uint8Array, data)

	promise := h.jsHost.Call("send", jsCtx, uint8Array, peerID, protocolID)
	_, err = goji.Await(goji.PromiseValue(promise))
	return err
}

// Sign will return a hash of the provided data signed with the private key of the host.
func (h *JSHostAdapter) Sign(data []byte) ([]byte, error) {
	signFunc := h.jsHost.Get("sign")
	if !signFunc.Truthy() {
		fmt.Println("sign", "fail")
		return nil, ErrMissingHostMethod
	}

	// Convert []byte to Uint8Array
	uint8Array := js.Global().Get("Uint8Array").New(len(data))
	js.CopyBytesToJS(uint8Array, data)

	promise := h.jsHost.Call("sign", uint8Array)
	result, err := goji.Await(goji.PromiseValue(promise))
	if err != nil {
		return nil, err
	}

	// Handle Uint8Array from JavaScript
	signedData := result[0]
	length := signedData.Get("length").Int()
	signature := make([]byte, length)
	js.CopyBytesToGo(signature, signedData)
	return signature, nil
}

// SetStreamHandler tells the host to listen for messages of the provided protocol ID and
// handle them with the given handler.
func (h *JSHostAdapter) SetStreamHandler(protocolID string, handler client.StreamHandler) {
	setHandlerFunc := h.jsHost.Get("setStreamHandler")
	if !setHandlerFunc.Truthy() {
		return
	}

	// Wrap the Go handler to be callable from JavaScript
	jsHandler := js.FuncOf(func(this js.Value, args []js.Value) any {
		if len(args) < 2 {
			return nil
		}

		// Convert JavaScript ReadableStream to Go io.Reader
		jsStream := args[0]
		peerID := args[1].String()

		// Create a reader that reads from the JavaScript stream
		reader := newJSStreamReader(jsStream)
		handler(reader, peerID)

		return nil
	})

	fmt.Println("got here", setHandlerFunc.Type())
	h.jsHost.Call("setStreamHandler", protocolID, jsHandler)
	fmt.Println("got here2")

}

// AddPubSubTopic adds a pubsub topic to the host.
func (h *JSHostAdapter) AddPubSubTopic(
	topicName string,
	subscribe bool,
	handler client.PubsubMessageHandler,
	eventHandler client.PeerEventHandler,
) error {
	addTopicFunc := h.jsHost.Get("addPubSubTopic")
	if !addTopicFunc.Truthy() {
		fmt.Println("addPubSubTopic", "fail")
		return ErrMissingHostMethod
	}

	// Wrap the Go handler to be callable from JavaScript
	jsHandler := js.FuncOf(func(this js.Value, args []js.Value) any {
		if len(args) < 3 {
			return nil
		}

		from := args[0].String()
		topic := args[1].String()

		// Convert Uint8Array to []byte
		msgArray := args[2]
		length := msgArray.Get("length").Int()
		msg := make([]byte, length)
		js.CopyBytesToGo(msg, msgArray)

		response, err := handler(from, topic, msg)
		if err != nil {
			return js.ValueOf(map[string]any{
				"error": err.Error(),
			})
		}

		// Convert response []byte to Uint8Array
		uint8Array := js.Global().Get("Uint8Array").New(len(response))
		js.CopyBytesToJS(uint8Array, response)

		return uint8Array
	})

	// Wrap the Go handler to be callable from JavaScript
	jsEventHandler := js.FuncOf(func(this js.Value, args []js.Value) any {
		if len(args) < 3 {
			return nil
		}

		from := args[0].String()
		topic := args[1].String()

		// Convert Uint8Array to []byte
		msgArray := args[2]
		length := msgArray.Get("length").Int()
		msg := make([]byte, length)
		js.CopyBytesToGo(msg, msgArray)

		response, err := handler(from, topic, msg)
		if err != nil {
			return js.ValueOf(map[string]any{
				"error": err.Error(),
			})
		}

		// Convert response []byte to Uint8Array
		uint8Array := js.Global().Get("Uint8Array").New(len(response))
		js.CopyBytesToJS(uint8Array, response)

		return uint8Array
	})

	promise := h.jsHost.Call("addPubSubTopic", topicName, subscribe, jsHandler, jsEventHandler)
	_, err := goji.Await(goji.PromiseValue(promise))
	return err
}

// RemovePubSubTopic removes the given topic from the host.
func (h *JSHostAdapter) RemovePubSubTopic(topic string) error {
	removeTopicFunc := h.jsHost.Get("removePubSubTopic")
	if !removeTopicFunc.Truthy() {
		fmt.Println("removePubSubTopic", "fail")
		return ErrMissingHostMethod
	}

	promise := h.jsHost.Call("removePubSubTopic", topic)
	_, err := goji.Await(goji.PromiseValue(promise))
	return err
}

// PublishToTopicAsync sends a new message on the given topic without waiting for a response.
func (h *JSHostAdapter) PublishToTopicAsync(ctx context.Context, topic string, data []byte) error {
	fmt.Println("push to topic asyc in host adapter")
	publishAsyncFunc := h.jsHost.Get("publishToTopicAsync")
	if !publishAsyncFunc.Truthy() {
		fmt.Println("publishToTopicAsync", "fail")
		return ErrMissingHostMethod
	}

	// Convert []byte to Uint8Array
	uint8Array := js.Global().Get("Uint8Array").New(len(data))
	js.CopyBytesToJS(uint8Array, data)

	promise := h.jsHost.Call("publishToTopicAsync", topic, uint8Array)
	_, err := goji.Await(goji.PromiseValue(promise))
	return err
}

// PublishToTopic sends a new message on the given topic, returning a response channel.
func (h *JSHostAdapter) PublishToTopic(
	ctx context.Context,
	topic string,
	data []byte,
	withMultiResponse bool,
) (<-chan client.PubsubResponse, error) {
	publishFunc := h.jsHost.Get("publishToTopic")
	if !publishFunc.Truthy() {
		fmt.Println("publishToTopic", "fail")
		return nil, ErrMissingHostMethod
	}

	// Convert []byte to Uint8Array
	uint8Array := js.Global().Get("Uint8Array").New(len(data))
	js.CopyBytesToJS(uint8Array, data)

	promise := h.jsHost.Call("publishToTopic", topic, uint8Array, withMultiResponse)
	result, err := goji.Await(goji.PromiseValue(promise))
	if err != nil {
		return nil, err
	}

	// Create a channel to receive responses
	responseChan := make(chan client.PubsubResponse)

	// The JavaScript side should return an AsyncIterator
	jsIterator := result[0]

	// Start a goroutine to consume the async iterator
	go func() {
		defer close(responseChan)
		for {
			nextPromise := jsIterator.Call("next")
			nextResult, err := goji.Await(goji.PromiseValue(nextPromise))
			if err != nil {
				return
			}

			done := nextResult[0].Get("done").Bool()
			if done {
				return
			}

			value := nextResult[0].Get("value")

			// Parse the PubsubResponse
			var resp client.PubsubResponse
			resp.ID = value.Get("id").String()
			resp.From = value.Get("from").String()

			dataArray := value.Get("data")
			length := dataArray.Get("length").Int()
			resp.Data = make([]byte, length)
			js.CopyBytesToGo(resp.Data, dataArray)

			errMsg := value.Get("error")
			if errMsg.Truthy() {
				resp.Err = js.Error{Value: errMsg}
			}

			select {
			case responseChan <- resp:
			case <-ctx.Done():
				return
			}
		}
	}()

	return responseChan, nil
}

// IPLDStore returns the host's IPLD store implementation.
// This calls the JavaScript peer's ipldStore() method which returns
// a bitswap-enabled blockstore that can fetch blocks from the network.
func (h *JSHostAdapter) IPLDStore() blockstore.IPLDStore {
	ipldStoreFunc := h.jsHost.Get("ipldStore")
	if !ipldStoreFunc.Truthy() {
		return nil
	}

	// Call ipldStore() to get the bitswap-enabled blockstore
	jsBlockstore := h.jsHost.Call("ipldStore")

	// Wrap it in an adapter that implements blockstore.IPLDStore
	return &jsIPLDStoreAdapter{
		jsBlockstore: jsBlockstore,
	}
}

// ContextWithSession returns a new context with a session for the underlying block service.
func (h *JSHostAdapter) ContextWithSession(ctx context.Context) context.Context {
	sessionFunc := h.jsHost.Get("contextWithSession")
	if !sessionFunc.Truthy() {
		// If not implemented in JS, just return the original context
		return ctx
	}

	jsCtx, err := goji.MarshalJS(contextToJS(ctx))
	if err != nil {
		return ctx
	}

	result := h.jsHost.Call("contextWithSession", jsCtx)
	// The TypeScript side can maintain session state internally
	// For now, we return the original context with a session marker
	return context.WithValue(ctx, "p2pSession", result)
}

// SetBlockAccessFunc sets the function to use to determine if a peer has access to
// the requested blocks on the block service.
func (h *JSHostAdapter) SetBlockAccessFunc(accessFunc client.BlockAccessFunc) {
	setAccessFunc := h.jsHost.Get("setBlockAccessFunc")
	if !setAccessFunc.Truthy() {
		return
	}

	// Wrap the Go function to be callable from JavaScript
	jsAccessFunc := js.FuncOf(func(this js.Value, args []js.Value) any {
		if len(args) < 3 {
			return false
		}

		jsCtx := args[0]
		peerID := args[1].String()
		cidStr := args[2].String()

		// Parse the CID
		c, err := cid.Decode(cidStr)
		if err != nil {
			return false
		}

		// Convert JS context back to Go context
		ctx := jsToContext(jsCtx)

		return accessFunc(ctx, peerID, c)
	})

	h.jsHost.Call("setBlockAccessFunc", jsAccessFunc)
}

// jsStreamReader implements io.Reader for JavaScript ReadableStream
type jsStreamReader struct {
	reader js.Value
	buffer []byte
}

func newJSStreamReader(stream js.Value) *jsStreamReader {
	reader := stream.Call("getReader")
	return &jsStreamReader{
		reader: reader,
		buffer: make([]byte, 0),
	}
}

func (r *jsStreamReader) Read(p []byte) (n int, err error) {
	if len(r.buffer) > 0 {
		n = copy(p, r.buffer)
		r.buffer = r.buffer[n:]
		return n, nil
	}

	// Read from the JavaScript stream
	readPromise := r.reader.Call("read")
	result, err := goji.Await(goji.PromiseValue(readPromise))
	if err != nil {
		return 0, err
	}

	done := result[0].Get("done").Bool()
	if done {
		return 0, io.EOF
	}

	value := result[0].Get("value")
	length := value.Get("length").Int()
	chunk := make([]byte, length)
	js.CopyBytesToGo(chunk, value)

	n = copy(p, chunk)
	if n < len(chunk) {
		r.buffer = chunk[n:]
	}

	return n, nil
}

// contextToJS converts a Go context to a JavaScript object
func contextToJS(ctx context.Context) map[string]any {
	jsCtx := make(map[string]any)

	// Extract deadline if present
	if deadline, ok := ctx.Deadline(); ok {
		jsCtx["deadline"] = deadline.UnixMilli()
	}

	// Note: Context values would need to be extracted individually
	// depending on what the TypeScript implementation needs

	return jsCtx
}

// jsToContext converts a JavaScript context object back to Go context
func jsToContext(jsCtx js.Value) context.Context {
	ctx := context.Background()

	// Check for deadline
	deadline := jsCtx.Get("deadline")
	if deadline.Truthy() {
		// Would need to reconstruct deadline from timestamp
		// This is a simplified implementation
	}

	return ctx
}

// Close closes the JavaScript Host.
// This is a no-op as the JavaScript Host is managed by the browser.
// The Close method is provided to satisfy the Peer interface in the node package.
func (h *JSHostAdapter) Close() {
	// External hosts are managed by JavaScript, so we don't close them here.
	// The JavaScript code is responsible for closing the host when appropriate.
	// If the JavaScript Host has a close method, it can be called here:
	closeFunc := h.jsHost.Get("close")
	if closeFunc.Truthy() {
		h.jsHost.Call("close")
	}
}

// jsIPLDStoreAdapter wraps a JavaScript blockstore to implement blockstore.IPLDStore.
// This allows the bitswap-enabled JavaScript blockstore to be used by Go code.
type jsIPLDStoreAdapter struct {
	jsBlockstore js.Value
}

// Get retrieves a block from the JavaScript blockstore.
// The JavaScript blockstore is backed by Helia's bitswap, so if the block
// is not found locally, it will be fetched from remote peers.
func (s *jsIPLDStoreAdapter) Get(ctx context.Context, key string) ([]byte, error) {
	getFunc := s.jsBlockstore.Get("get")
	if !getFunc.Truthy() {
		fmt.Println("get", "fail")
		return nil, ErrMissingHostMethod
	}

	// Parse the CID string and convert to a JavaScript CID object
	// Helia's blockstore expects a CID object from multiformats/cid, not a string
	cidParse := js.Global().Get("CID").Get("parse")
	if !cidParse.Truthy() {
		fmt.Println("CID parse")
		// Fallback: try to call get with string (for backwards compatibility)
		fmt.Println("Warning: CID.parse not available, trying with string")
		promise := s.jsBlockstore.Call("get", key)
		result, err := goji.Await(goji.PromiseValue(promise))
		if err != nil {
			return nil, err
		}
		uint8Array := result[0]
		length := uint8Array.Get("length").Int()
		data := make([]byte, length)
		js.CopyBytesToGo(data, uint8Array)
		return data, nil
	}
	fmt.Println("no CID parse")

	// Parse the CID string to a CID object
	cidObj := cidParse.Invoke(key)

	// Call the JavaScript get method with the CID object
	promise := s.jsBlockstore.Call("get", cidObj)
	result, err := goji.Await(goji.PromiseValue(promise))
	if err != nil {
		return nil, err
	}

	// The result should be a Uint8Array
	uint8Array := result[0]
	length := uint8Array.Get("length").Int()
	data := make([]byte, length)
	js.CopyBytesToGo(data, uint8Array)

	return data, nil
}

// Put stores a block in the JavaScript blockstore.
func (s *jsIPLDStoreAdapter) Put(ctx context.Context, key string, data []byte) error {
	putFunc := s.jsBlockstore.Get("put")
	if !putFunc.Truthy() {
		fmt.Println("put", "fail")
		return ErrMissingHostMethod
	}

	// Parse the CID string and convert to a JavaScript CID object
	cidParse := js.Global().Get("CID").Get("parse")
	if !cidParse.Truthy() {
		// Fallback: try with string
		uint8Array := js.Global().Get("Uint8Array").New(len(data))
		js.CopyBytesToJS(uint8Array, data)
		promise := s.jsBlockstore.Call("put", key, uint8Array)
		_, err := goji.Await(goji.PromiseValue(promise))
		return err
	}

	cidObj := cidParse.Invoke(key)

	// Convert []byte to Uint8Array
	uint8Array := js.Global().Get("Uint8Array").New(len(data))
	js.CopyBytesToJS(uint8Array, data)

	// Call the JavaScript put method with CID object
	promise := s.jsBlockstore.Call("put", cidObj, uint8Array)
	_, err := goji.Await(goji.PromiseValue(promise))
	return err
}

// Has checks if a block exists in the JavaScript blockstore.
func (s *jsIPLDStoreAdapter) Has(ctx context.Context, key string) (bool, error) {
	hasFunc := s.jsBlockstore.Get("has")
	if !hasFunc.Truthy() {
		fmt.Println("has", "fail")
		return false, ErrMissingHostMethod
	}

	// Parse the CID string and convert to a JavaScript CID object
	cidParse := js.Global().Get("CID").Get("parse")
	if !cidParse.Truthy() {
		// Fallback: try with string
		promise := s.jsBlockstore.Call("has", key)
		result, err := goji.Await(goji.PromiseValue(promise))
		if err != nil {
			return false, err
		}
		return result[0].Bool(), nil
	}

	cidObj := cidParse.Invoke(key)

	// Call the JavaScript has method with CID object
	promise := s.jsBlockstore.Call("has", cidObj)
	result, err := goji.Await(goji.PromiseValue(promise))
	if err != nil {
		return false, err
	}

	// The result should be a boolean
	return result[0].Bool(), nil
}
