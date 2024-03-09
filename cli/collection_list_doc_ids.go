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

	"github.com/sourcenetwork/defradb/http"
)

func MakeCollectionListDocIDsCommand() *cobra.Command {
	var cmd = &cobra.Command{
		Use:   "docIDs",
		Short: "List all document IDs (docIDs).",
		Long: `List all document IDs (docIDs).
		
Example:
  defradb client collection docIDs --name User
		`,
		RunE: func(cmd *cobra.Command, args []string) error {
			col, ok := tryGetContextCollection(cmd)
			if !ok {
				return cmd.Usage()
			}

			docCh, err := col.GetAllDocIDs(cmd.Context())
			if err != nil {
				return err
			}
			for docIDResult := range docCh {
				results := &http.DocIDResult{
					DocID: docIDResult.ID.String(),
				}
				if docIDResult.Err != nil {
					results.Error = docIDResult.Err.Error()
				}
				if err := writeJSON(cmd, results); err != nil {
					return err
				}
			}
			return nil
		},
	}
	return cmd
}
