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

package utils

import (
	"context"
	"encoding/hex"
	"fmt"
	"reflect"
	"sync"
	"syscall/js"

	"github.com/decred/dcrd/dcrec/secp256k1/v4"
	"github.com/sourcenetwork/goji"
	"github.com/sourcenetwork/immutable"

	acpIdentity "github.com/sourcenetwork/defradb/acp/identity"
	"github.com/sourcenetwork/defradb/client"
	"github.com/sourcenetwork/defradb/crypto"
	"github.com/sourcenetwork/defradb/internal/db"
	iIdentity "github.com/sourcenetwork/defradb/internal/identity"
)

func StringArg(args []js.Value, index int, name string) (string, error) {
	if len(args) < index {
		return "", fmt.Errorf("%s argument is required", name)
	}
	if args[index].Type() != js.TypeString {
		return "", fmt.Errorf("%s argument must be a string", name)
	}
	return args[index].String(), nil
}

func BoolArg(args []js.Value, index int, name string) (bool, error) {
	if len(args) < index {
		return false, fmt.Errorf("%s argument is required", name)
	}
	if args[index].Type() != js.TypeBoolean {
		return false, fmt.Errorf("%s argument must be a bool", name)
	}
	return args[index].Bool(), nil
}

func IntArg(args []js.Value, index int, name string) (int, error) {
	if len(args) < index {
		return 0, fmt.Errorf("%s argument is required", name)
	}
	if args[index].Type() != js.TypeBoolean {
		return 0, fmt.Errorf("%s argument must be an int", name)
	}
	return args[index].Int(), nil
}

func Uint8ArrayArg(args []js.Value, index int, name string) ([]byte, error) {
	if len(args) <= index {
		return nil, fmt.Errorf("%s argument is required", name)
	}
	jsVal := args[index]
	if !jsVal.InstanceOf(js.Global().Get("Uint8Array")) {
		return nil, fmt.Errorf("%s argument must be a Uint8Array", name)
	}
	data := make([]byte, jsVal.Length())
	js.CopyBytesToGo(data, jsVal)
	return data, nil
}

func StructArg(args []js.Value, index int, name string, out any) error {
	if len(args) < index {
		return fmt.Errorf("%s argument is required", name)
	}
	return goji.UnmarshalJS(args[index], out)
}

func ContextArg(args []js.Value, index int, txns *sync.Map) (context.Context, error) {
	ctx := context.Background()
	if index >= len(args) || args[index].IsUndefined() || args[index].IsNull() {
		return ctx, nil
	}
	identity, err := ContextIdentityArg(args[index])
	if err != nil {
		return ctx, err
	}
	txn, err := ContextTransactionArg(args[index], txns)
	if err != nil {
		return ctx, err
	}
	ctx = iIdentity.WithContext(ctx, identity)
	ctx = db.InitContext(ctx, txn)
	return ctx, nil
}

func ContextTransactionArg(value js.Value, txns *sync.Map) (client.Txn, error) {
	id := value.Get("transaction")
	if id.Type() != js.TypeNumber {
		return nil, nil
	}
	txn, ok := txns.Load(uint64(id.Int()))
	if !ok {
		return nil, ErrInvalidTransactionId
	}
	return txn.(client.Txn), nil //nolint:forcetypeassert
}

func ContextIdentityArg(value js.Value) (immutable.Option[acpIdentity.Identity], error) {
	full_ident := value.Get("full_identity")
	if full_ident.Type() == js.TypeString {
		data, err := hex.DecodeString(full_ident.String())
		if err != nil {
			return immutable.None[acpIdentity.Identity](), err
		}
		privKey := secp256k1.PrivKeyFromBytes(data)
		identity, err := acpIdentity.FromPrivateKey(crypto.NewPrivateKey(privKey))
		if err != nil {
			return immutable.None[acpIdentity.Identity](), err
		}
		return immutable.Some[acpIdentity.Identity](identity), nil
	}
	ident := value.Get("identity")
	if ident.Type() != js.TypeString {
		return immutable.None[acpIdentity.Identity](), nil
	}
	publicKey, err := crypto.PublicKeyFromString(crypto.KeyTypeSecp256r1, ident.String())
	if err != nil {
		return immutable.None[acpIdentity.Identity](), err
	}
	identity, err := acpIdentity.FromPublicKey(publicKey)
	if err != nil {
		return immutable.None[acpIdentity.Identity](), err
	}
	return immutable.Some(identity), nil
}

// setOptIdentity extracts identity from args at the given index and sets it on the option builder.
// We use reflection to call SetIdentity if the builder has it, ignoring the return value.
func setOptIdentity[B any](opt B, args []js.Value, argIndex int) {
	if len(args) > argIndex {
		if ident, err := contextIdentityArg(args[argIndex]); err == nil && ident.HasValue() {
			// Use reflect to call SetIdentity regardless of return type.
			v := reflect.ValueOf(opt)
			m := v.MethodByName("SetIdentity")
			if m.IsValid() {
				m.Call([]reflect.Value{reflect.ValueOf(ident.Value())})
			}
		}
	}
}

// initKeypairAndGetIdentity initializes the keypair and gets an identity.
func initKeypairAndGetIdentity() (acpIdentity.Identity, error) {
	createKeyPairFunc := js.Global().Get("initKeypair")
	if !createKeyPairFunc.Truthy() {
		return nil, fmt.Errorf("initKeypair function not found")
	}
	results, err := goji.Await(goji.PromiseValue(createKeyPairFunc.Invoke()))
	if err != nil {
		return nil, fmt.Errorf("failed to await initKeypair: %w", err)
	}
	if len(results) == 0 || results[0].String() == "" {
		return nil, fmt.Errorf("initKeypair returned no valid public key")
	}
	publicKey, err := crypto.PublicKeyFromString(crypto.KeyTypeSecp256r1, results[0].String())
	if err != nil {
		return nil, fmt.Errorf("failed to create public key from hex: %w", err)
	}
	ident, err := acpIdentity.FromPublicKey(publicKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create identity from public key: %w", err)
	}
	return ident, nil
}
