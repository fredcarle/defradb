// Copyright 2022 Democratized Data Foundation
//
// Use of this software is governed by the Business Source License
// included in the file licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with
// the Business Source License, use of this software will be governed
// by the Apache License, Version 2.0, included in the file
// licenses/APL.txt.

/*
Package cli provides the command-line interface.
*/
package cli

import (
	"github.com/spf13/cobra"

	"github.com/sourcenetwork/defradb/logging"
)

var log = logging.MustNewLogger("cli")

// NewDefraCommand returns the root command instanciated with its tree of subcommands.
func NewDefraCommand() *cobra.Command {
	p2p_collection := MakeP2PCollectionCommand()
	p2p_collection.AddCommand(
		MakeP2PCollectionAddCommand(),
		MakeP2PCollectionRemoveCommand(),
		MakeP2PCollectionGetAllCommand(),
	)

	p2p_replicator := MakeP2PReplicatorCommand()
	p2p_replicator.AddCommand(
		MakeP2PReplicatorGetAllCommand(),
		MakeP2PReplicatorSetCommand(),
		MakeP2PReplicatorDeleteCommand(),
	)

	p2p := MakeP2PCommand()
	p2p.AddCommand(
		p2p_replicator,
		p2p_collection,
		MakeP2PInfoCommand(),
	)

	schema_migrate := MakeSchemaMigrationCommand()
	schema_migrate.AddCommand(
		MakeSchemaMigrationSetCommand(),
		MakeSchemaMigrationSetRegistryCommand(),
		MakeSchemaMigrationReloadCommand(),
		MakeSchemaMigrationUpCommand(),
		MakeSchemaMigrationDownCommand(),
	)

	schema := MakeSchemaCommand()
	schema.AddCommand(
		MakeSchemaAddCommand(),
		MakeSchemaPatchCommand(),
		MakeSchemaSetActiveCommand(),
		MakeSchemaDescribeCommand(),
		schema_migrate,
	)

	view := MakeViewCommand()
	view.AddCommand(
		MakeViewAddCommand(),
	)

	index := MakeIndexCommand()
	index.AddCommand(
		MakeIndexCreateCommand(),
		MakeIndexDropCommand(),
		MakeIndexListCommand(),
	)

	backup := MakeBackupCommand()
	backup.AddCommand(
		MakeBackupExportCommand(),
		MakeBackupImportCommand(),
	)

	tx := MakeTxCommand()
	tx.AddCommand(
		MakeTxCreateCommand(),
		MakeTxCommitCommand(),
		MakeTxDiscardCommand(),
	)

	collection := MakeCollectionCommand()
	collection.AddCommand(
		MakeCollectionGetCommand(),
		MakeCollectionListDocIDsCommand(),
		MakeCollectionDeleteCommand(),
		MakeCollectionUpdateCommand(),
		MakeCollectionCreateCommand(),
		MakeCollectionDescribeCommand(),
	)

	client := MakeClientCommand()
	client.AddCommand(
		MakeDumpCommand(),
		MakeRequestCommand(),
		schema,
		view,
		index,
		p2p,
		backup,
		tx,
		collection,
	)

	root := MakeRootCommand()
	root.AddCommand(
		client,
		MakeStartCommand(),
		MakeServerDumpCmd(),
		MakeVersionCommand(),
	)

	return root
}
