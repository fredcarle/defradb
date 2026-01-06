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

package datastore

import (
	"context"
	"fmt"
	"syscall/js"

	blocks "github.com/ipfs/go-block-format"
	"github.com/ipfs/go-cid"
	"github.com/sourcenetwork/corekv"
	"github.com/sourcenetwork/defradb/errors"
	"github.com/sourcenetwork/defradb/internal/datastore"
	defraJS "github.com/sourcenetwork/defradb/js/utils"
	"github.com/sourcenetwork/goji"
	"github.com/sourcenetwork/immutable"
)

type JSBlockstore struct {
	blockstore datastore.Blockstore
}

func NewJSBlockstore(rootstore corekv.ReaderWriter, chunkSize immutable.Option[int]) *JSBlockstore {
	return &JSBlockstore{
		// Use P2P blockstore by default
		blockstore: datastore.P2PBlockstoreFrom(rootstore, chunkSize),
	}
}

func (b *JSBlockstore) JSValue() js.Value {
	return js.ValueOf(map[string]any{
		"has":        goji.Async(b.has),
		"get":        b.getAsync(),
		"put":        goji.Async(b.put),
		"delete":     goji.Async(b.delete),
		"putMany":    goji.Async(b.putMany),
		"getMany":    goji.Async(b.getMany),
		"deleteMany": goji.Async(b.deleteMany),
		"getAll":     goji.Async(b.getAll),
	})
}

func (b *JSBlockstore) has(this js.Value, args []js.Value) (js.Value, error) {
	if len(args) == 0 || args[0].IsUndefined() || args[0].IsNull() {
		return js.Undefined(), errors.New("missing key argument")
	}
	key := args[0]
	cidStr := key.Call("toString").String()

	k, err := cid.Decode(cidStr)
	if err != nil {
		return js.Undefined(), err
	}

	has, err := b.blockstore.Has(context.Background(), k)
	if err != nil {
		return js.Undefined(), err
	}
	return goji.MarshalJS(has)
}

// getAsync returns an async generator function
// Uses JavaScript to create a proper async generator since Go can't set Symbol properties easily
func (b *JSBlockstore) getAsync() js.Func {
	return js.FuncOf(func(this js.Value, args []js.Value) any {
		fmt.Println("JSBlockstore.getAsync: creating async generator")

		// Get the block data immediately
		result, err := b.get(this, args)
		if err != nil {
			fmt.Println("JSBlockstore.getAsync: ERROR:", err)
			// Return an async generator that throws the error
			return js.Global().Call("eval", `
				(async function*() {
					throw new Error(`+fmt.Sprintf("%q", err.Error())+`);
				})
			`).Invoke()
		}

		fmt.Printf("JSBlockstore.getAsync: got Uint8Array, length: %d\n", result.Get("length").Int())

		// Create an async generator that yields this specific result
		// We use eval to create a true JavaScript async generator
		asyncGen := js.Global().Call("eval", `
			(async function*(data) {
				yield data;
			})
		`).Invoke(result)

		fmt.Println("JSBlockstore.getAsync: returning async generator")
		return asyncGen
	})
}

func (b *JSBlockstore) get(this js.Value, args []js.Value) (js.Value, error) {
	fmt.Println("JSBlockstore.get called with", len(args), "arguments")

	if len(args) == 0 || args[0].IsUndefined() || args[0].IsNull() {
		return js.Undefined(), errors.New("missing key argument")
	}

	key := args[0]
	cidStr := key.Call("toString").String()
	fmt.Println("JSBlockstore.get: CID =", cidStr)

	// Check if options are passed (second argument)
	ctx := context.Background()
	if len(args) > 1 && !args[1].IsUndefined() && !args[1].IsNull() {
		options := args[1]
		fmt.Println("JSBlockstore.get: options provided, type:", options.Type())
		// TODO: Could extract AbortSignal from options.signal if needed
	}

	k, err := cid.Decode(cidStr)
	if err != nil {
		fmt.Println("JSBlockstore.get: ERROR - invalid CID:", err)
		return js.Undefined(), err
	}

	block, err := b.blockstore.Get(ctx, k)
	if err != nil {
		fmt.Println("JSBlockstore.get: ERROR - block not found:", err)
		return js.Undefined(), err
	}

	blockData := block.RawData()
	fmt.Println("JSBlockstore.get: SUCCESS - found block, size:", len(blockData), "bytes")

	// Return Uint8Array
	uint8Array := js.Global().Get("Uint8Array")
	jsBytes := uint8Array.New(len(blockData))
	js.CopyBytesToJS(jsBytes, blockData)

	fmt.Println("JSBlockstore.get: returning Uint8Array, length:", jsBytes.Get("length").Int())
	return jsBytes, nil
}

func (b *JSBlockstore) put(this js.Value, args []js.Value) (js.Value, error) {
	fmt.Println("JSBlockstore.put called with", len(args), "arguments")

	if len(args) == 0 || args[0].IsUndefined() || args[0].IsNull() {
		return js.Undefined(), errors.New("missing key argument")
	}
	key := args[0]
	cidStr := key.Call("toString").String()
	fmt.Println("JSBlockstore.put: CID =", cidStr)

	data, err := defraJS.Uint8ArrayArg(args, 1, "data")
	if err != nil {
		fmt.Println("JSBlockstore.put: ERROR - invalid data:", err)
		return js.Undefined(), err
	}
	fmt.Println("JSBlockstore.put: data size =", len(data), "bytes")

	k, err := cid.Decode(cidStr)
	if err != nil {
		fmt.Println("JSBlockstore.put: ERROR - invalid CID:", err)
		return js.Undefined(), err
	}

	block, err := blocks.NewBlockWithCid(data, k)
	if err != nil {
		fmt.Println("JSBlockstore.put: ERROR - failed to create block:", err)
		return js.Undefined(), err
	}

	err = b.blockstore.Put(context.Background(), block)
	if err != nil {
		fmt.Println("JSBlockstore.put: ERROR - failed to store block:", err)
		return js.Undefined(), err
	}

	fmt.Println("JSBlockstore.put: SUCCESS - stored block, returning CID")
	// IMPORTANT: Blockstore interface requires put() to return the CID
	return key, nil
}

func (b *JSBlockstore) delete(this js.Value, args []js.Value) (js.Value, error) {
	if len(args) == 0 || args[0].IsUndefined() || args[0].IsNull() {
		return js.Undefined(), errors.New("missing key argument")
	}
	key := args[0]
	cidStr := key.Call("toString").String()

	k, err := cid.Decode(cidStr)
	if err != nil {
		return js.Undefined(), err
	}

	err = b.blockstore.DeleteBlock(context.Background(), k)
	return js.Undefined(), err
}

func (b *JSBlockstore) putMany(this js.Value, args []js.Value) (js.Value, error) {
	fmt.Println("putMany", args)

	// args[1] should be an async iterator of {cid, block} pairs
	// For now, we'll accept an array of {cid, block} objects
	source := args[1]
	if !source.Truthy() {
		return js.Undefined(), nil
	}

	// Create an async iterator that yields CIDs as they're put
	iteratorFunc := js.FuncOf(func(this js.Value, args []js.Value) any {
		promise := js.Global().Get("Promise")
		return promise.New(js.FuncOf(func(this js.Value, args []js.Value) any {
			resolve := args[0]

			// Process the source array/iterator
			processNext := js.FuncOf(func(this js.Value, iterArgs []js.Value) any {
				nextPromise := source.Call("next")

				return nextPromise.Call("then", js.FuncOf(func(this js.Value, promiseArgs []js.Value) any {
					result := promiseArgs[0]
					done := result.Get("done").Bool()

					if done {
						resolve.Invoke(js.ValueOf(map[string]any{
							"value": js.Undefined(),
							"done":  true,
						}))
						return nil
					}

					value := result.Get("value")
					cidStr := value.Get("cid").String()
					data := value.Get("block")

					k, err := cid.Decode(cidStr)
					if err != nil {
						return js.Global().Get("Promise").Call("reject", err.Error())
					}

					goData := make([]byte, data.Length())
					js.CopyBytesToGo(goData, data)

					block, err := blocks.NewBlockWithCid(goData, k)
					if err != nil {
						return js.Global().Get("Promise").Call("reject", err.Error())
					}

					err = b.blockstore.Put(context.Background(), block)
					if err != nil {
						return js.Global().Get("Promise").Call("reject", err.Error())
					}

					// Yield the CID
					resolve.Invoke(js.ValueOf(map[string]any{
						"value": cidStr,
						"done":  false,
					}))

					return nil
				}))
			})

			processNext.Invoke()
			return nil
		}))
	})

	// Return an async iterator
	asyncIteratorSymbol := js.Global().Get("Symbol").Get("asyncIterator")
	iterator := js.ValueOf(map[string]any{
		"next": iteratorFunc,
	})
	iterator.Set(asyncIteratorSymbol.String(), js.FuncOf(func(this js.Value, args []js.Value) any {
		return this
	}))
	return iterator, nil
}

func (b *JSBlockstore) getMany(this js.Value, args []js.Value) (js.Value, error) {
	fmt.Println("getMany", args)

	// args[1] should be an async iterator of CIDs
	source := args[1]
	if !source.Truthy() {
		return js.Undefined(), nil
	}

	// Create an async iterator that yields {cid, block} pairs
	// The next() function must be recreated for each call to properly handle promises
	iteratorFunc := js.FuncOf(func(this js.Value, args []js.Value) any {
		promise := js.Global().Get("Promise")
		return promise.New(js.FuncOf(func(this js.Value, promiseArgs []js.Value) any {
			resolve := promiseArgs[0]
			reject := promiseArgs[1]

			// Get next value from source iterator
			nextPromise := source.Call("next")

			// Handle the promise resolution
			thenCallback := js.FuncOf(func(this js.Value, thenArgs []js.Value) any {
				result := thenArgs[0]
				done := result.Get("done").Bool()

				if done {
					// Source iterator is exhausted
					resolve.Invoke(js.ValueOf(map[string]any{
						"value": js.Undefined(),
						"done":  true,
					}))
					return nil
				}

				// Get the CID from the source
				cidValue := result.Get("value")
				if cidValue.IsUndefined() || cidValue.IsNull() {
					reject.Invoke("CID value is undefined or null")
					return nil
				}

				cidStr := cidValue.String()
				fmt.Println("getMany: fetching block for CID:", cidStr)

				k, err := cid.Decode(cidStr)
				if err != nil {
					fmt.Println("getMany: invalid CID:", err)
					reject.Invoke(err.Error())
					return nil
				}

				// Get block from blockstore
				block, err := b.blockstore.Get(context.Background(), k)
				if err != nil {
					fmt.Println("getMany: block not found:", err)
					reject.Invoke(err.Error())
					return nil
				}

				fmt.Println("getMany: block found, size:", len(block.RawData()))

				// Convert block data to Uint8Array
				uint8Array := js.Global().Get("Uint8Array")
				jsBytes := uint8Array.New(len(block.RawData()))
				js.CopyBytesToJS(jsBytes, block.RawData())

				// Return {value: {cid, block}, done: false}
				resolve.Invoke(js.ValueOf(map[string]any{
					"value": map[string]any{
						"cid":   cidStr,
						"block": jsBytes,
					},
					"done": false,
				}))

				return nil
			})

			catchCallback := js.FuncOf(func(this js.Value, catchArgs []js.Value) any {
				fmt.Println("getMany: error in promise chain:", catchArgs[0])
				reject.Invoke(catchArgs[0])
				return nil
			})

			nextPromise.Call("then", thenCallback).Call("catch", catchCallback)

			return nil
		}))
	})

	// Return an async iterator
	asyncIteratorSymbol := js.Global().Get("Symbol").Get("asyncIterator")
	iterator := js.ValueOf(map[string]any{
		"next": iteratorFunc,
	})
	iterator.Set(asyncIteratorSymbol.String(), js.FuncOf(func(this js.Value, args []js.Value) any {
		return this
	}))
	return iterator, nil
}

func (b *JSBlockstore) deleteMany(this js.Value, args []js.Value) (js.Value, error) {
	fmt.Println("deleteMany", args)
	// args[1] should be an async iterator of CIDs
	source := args[1]
	if !source.Truthy() {
		return js.Undefined(), nil
	}

	// Create an async iterator that yields CIDs as they're deleted
	iteratorFunc := js.FuncOf(func(this js.Value, args []js.Value) any {
		promise := js.Global().Get("Promise")
		return promise.New(js.FuncOf(func(this js.Value, args []js.Value) any {
			resolve := args[0]

			nextPromise := source.Call("next")

			return nextPromise.Call("then", js.FuncOf(func(this js.Value, promiseArgs []js.Value) any {
				result := promiseArgs[0]
				done := result.Get("done").Bool()

				if done {
					resolve.Invoke(js.ValueOf(map[string]any{
						"value": js.Undefined(),
						"done":  true,
					}))
					return nil
				}

				cidStr := result.Get("value").String()
				k, err := cid.Decode(cidStr)
				if err != nil {
					return js.Global().Get("Promise").Call("reject", err.Error())
				}

				err = b.blockstore.DeleteBlock(context.Background(), k)
				if err != nil {
					return js.Global().Get("Promise").Call("reject", err.Error())
				}

				resolve.Invoke(js.ValueOf(map[string]any{
					"value": cidStr,
					"done":  false,
				}))

				return nil
			}))
		}))
	})

	// Return an async iterator
	asyncIteratorSymbol := js.Global().Get("Symbol").Get("asyncIterator")
	iterator := js.ValueOf(map[string]any{
		"next": iteratorFunc,
	})
	iterator.Set(asyncIteratorSymbol.String(), js.FuncOf(func(this js.Value, args []js.Value) any {
		return this
	}))
	return iterator, nil
}

func (b *JSBlockstore) getAll(this js.Value, args []js.Value) (js.Value, error) {
	fmt.Println("getAll", args)
	// DefraDB blockstore doesn't support enumeration
	// This would require iterating the entire database
	return js.Undefined(), defraJS.ErrGetAllNotSupported
}
