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
	"github.com/spf13/cobra"

	"github.com/sourcenetwork/defradb/datastore"
)

func MakeIndexDropCommand() *cobra.Command {
	var collectionArg string
	var nameArg string
	var cmd = &cobra.Command{
		Use:   "drop -c --collection <collection> -n --name <name>",
		Short: "Drop a collection's secondary index",
		Long: `Drop a collection's secondary index.
		
Example: drop the index 'UsersByName' for 'Users' collection:
  defradb client index create --collection Users --name UsersByName`,
		ValidArgs: []string{"collection", "name"},
		RunE: func(cmd *cobra.Command, args []string) error {
			store := mustGetContextStore(cmd)

			col, err := store.GetCollectionByName(cmd.Context(), collectionArg)
			if err != nil {
				return err
			}
			if tx, ok := cmd.Context().Value(txContextKey).(datastore.Txn); ok {
				col = col.WithTxn(tx)
			}
			return col.DropIndex(cmd.Context(), nameArg)
		},
	}
	cmd.Flags().StringVarP(&collectionArg, "collection", "c", "", "Collection name")
	cmd.Flags().StringVarP(&nameArg, "name", "n", "", "Index name")

	return cmd
}
