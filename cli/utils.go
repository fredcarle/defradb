// Copyright 2023 Democratized Data Foundation
//
// Use of this software is governed by the Business Source License
// included in the file licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with
// the Business Source License, use of this software will be governed
// by the Apache License, Version 2.0, included in the file
// licenses/APL.txt.

package cli

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/sourcenetwork/defradb/client"
	"github.com/sourcenetwork/defradb/datastore"
	"github.com/sourcenetwork/defradb/http"
)

type contextKey string

var (
	// cfgContextKey is the context key for the config.
	cfgContextKey = contextKey("cfg")
	// rootDirContextKey is the context key for the root directory.
	rootDirContextKey = contextKey("rootDir")
	// txContextKey is the context key for the datastore.Txn
	//
	// This will only be set if a transaction id is specified.
	txContextKey = contextKey("tx")
	// dbContextKey is the context key for the client.DB
	dbContextKey = contextKey("db")
	// storeContextKey is the context key for the client.Store
	//
	// If a transaction exists, all operations will be executed
	// in the current transaction context.
	storeContextKey = contextKey("store")
	// colContextKey is the context key for the client.Collection
	//
	// If a transaction exists, all operations will be executed
	// in the current transaction context.
	colContextKey = contextKey("col")
)

// mustGetContextStore returns the store for the current command context.
//
// If a store is not set in the current context this function panics.
func mustGetContextStore(cmd *cobra.Command) client.Store {
	return cmd.Context().Value(storeContextKey).(client.Store)
}

// mustGetContextP2P returns the p2p implementation for the current command context.
//
// If a p2p implementation is not set in the current context this function panics.
func mustGetContextP2P(cmd *cobra.Command) client.P2P {
	return cmd.Context().Value(dbContextKey).(client.P2P)
}

// mustGetContextConfig returns the config for the current command context.
//
// If a config is not set in the current context this function panics.
func mustGetContextConfig(cmd *cobra.Command) *viper.Viper {
	return cmd.Context().Value(cfgContextKey).(*viper.Viper)
}

// mustGetContextRootDir returns the rootdir for the current command context.
//
// If a rootdir is not set in the current context this function panics.
func mustGetContextRootDir(cmd *cobra.Command) string {
	return cmd.Context().Value(rootDirContextKey).(string)
}

// tryGetContextCollection returns the collection for the current command context
// and a boolean indicating if the collection was set.
func tryGetContextCollection(cmd *cobra.Command) (client.Collection, bool) {
	col, ok := cmd.Context().Value(colContextKey).(client.Collection)
	return col, ok
}

// setContextConfig sets teh config for the current command context.
func setContextConfig(cmd *cobra.Command) error {
	rootdir := mustGetContextRootDir(cmd)
	flags := cmd.Root().PersistentFlags()
	cfg, err := loadConfig(rootdir, flags)
	if err != nil {
		return err
	}
	ctx := context.WithValue(cmd.Context(), cfgContextKey, cfg)
	cmd.SetContext(ctx)
	return nil
}

// setContextTransaction sets the transaction for the current command context.
func setContextTransaction(cmd *cobra.Command, txId uint64) error {
	if txId == 0 {
		return nil
	}
	cfg := mustGetContextConfig(cmd)
	tx, err := http.NewTransaction(cfg.GetString("api.address"), txId)
	if err != nil {
		return err
	}
	ctx := context.WithValue(cmd.Context(), txContextKey, tx)
	cmd.SetContext(ctx)
	return nil
}

// setContextStore sets the store for the current command context.
func setContextStore(cmd *cobra.Command) error {
	cfg := mustGetContextConfig(cmd)
	db, err := http.NewClient(cfg.GetString("api.address"))
	if err != nil {
		return err
	}
	ctx := context.WithValue(cmd.Context(), dbContextKey, db)
	if tx, ok := ctx.Value(txContextKey).(datastore.Txn); ok {
		ctx = context.WithValue(ctx, storeContextKey, db.WithTxn(tx))
	} else {
		ctx = context.WithValue(ctx, storeContextKey, db)
	}
	cmd.SetContext(ctx)
	return nil
}

// setContextRootDir sets the rootdir for the current command context.
func setContextRootDir(cmd *cobra.Command) error {
	rootdir, err := cmd.Root().PersistentFlags().GetString("rootdir")
	if err != nil {
		return err
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	if rootdir == "" {
		rootdir = filepath.Join(home, ".defradb")
	}
	ctx := context.WithValue(cmd.Context(), rootDirContextKey, rootdir)
	cmd.SetContext(ctx)
	return nil
}

// loadOrGeneratePrivateKey loads the private key from the given path
// or generates a new key and writes it to a file at the given path.
func loadOrGeneratePrivateKey(path string) (crypto.PrivKey, error) {
	key, err := loadPrivateKey(path)
	if err == nil {
		return key, nil
	}
	if os.IsNotExist(err) {
		return generatePrivateKey(path)
	}
	return nil, err
}

// generatePrivateKey generates a new private key and writes it
// to a file at the given path.
func generatePrivateKey(path string) (crypto.PrivKey, error) {
	key, _, err := crypto.GenerateKeyPair(crypto.Ed25519, 0)
	if err != nil {
		return nil, err
	}
	data, err := crypto.MarshalPrivateKey(key)
	if err != nil {
		return nil, err
	}
	err = os.MkdirAll(filepath.Dir(path), 0755)
	if err != nil {
		return nil, err
	}
	return key, os.WriteFile(path, data, 0644)
}

// loadPrivateKey reads the private key from the file at the given path.
func loadPrivateKey(path string) (crypto.PrivKey, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return crypto.UnmarshalPrivateKey(data)
}

func writeJSON(cmd *cobra.Command, out any) error {
	enc := json.NewEncoder(cmd.OutOrStdout())
	enc.SetIndent("", "  ")
	return enc.Encode(out)
}
