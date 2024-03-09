// Copyright 2023 Democratized Data Foundation
//
// Use of this software is governed by the Business Source License
// included in the file licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with
// the Business Source License, use of this software will be governed
// by the Apache License, Version 2.0, included in the file
// licenses/APL.txt.

package migrations

import (
	"testing"

	"github.com/lens-vm/lens/host-go/config/model"
	"github.com/sourcenetwork/immutable"

	"github.com/sourcenetwork/defradb/client"
	testUtils "github.com/sourcenetwork/defradb/tests/integration"
	"github.com/sourcenetwork/defradb/tests/lenses"
)

// Migrations need to be able to be registered for unknown schema ids, so they
// may migrate to/from them if recieved by the P2P system.
func TestSchemaMigrationDoesNotErrorGivenUnknownSchemaRoots(t *testing.T) {
	test := testUtils.TestCase{
		Description: "Test schema migration, unknown schema ids",
		Actions: []any{
			testUtils.ConfigureMigration{
				LensConfig: client.LensConfig{
					SourceSchemaVersionID:      "does not exist",
					DestinationSchemaVersionID: "also does not exist",
					Lens: model.Lens{
						Lenses: []model.LensModule{
							{
								Path: lenses.SetDefaultModulePath,
								Arguments: map[string]any{
									"dst":   "verified",
									"value": false,
								},
							},
						},
					},
				},
			},
			testUtils.GetCollections{
				FilterOptions: client.CollectionFetchOptions{
					IncludeInactive: immutable.Some(true),
				},
				ExpectedResults: []client.CollectionDescription{
					{
						ID:              1,
						SchemaVersionID: "does not exist",
					},
					{
						ID:              2,
						SchemaVersionID: "also does not exist",
						Sources: []any{
							&client.CollectionSource{
								SourceCollectionID: 1,
								Transform: immutable.Some(
									model.Lens{
										Lenses: []model.LensModule{
											{
												Path: lenses.SetDefaultModulePath,
												Arguments: map[string]any{
													"dst":   "verified",
													"value": false,
												},
											},
										},
									},
								),
							},
						},
					},
				},
			},
		},
	}

	testUtils.ExecuteTestCase(t, test)
}

func TestSchemaMigrationGetMigrationsReturnsMultiple(t *testing.T) {
	test := testUtils.TestCase{
		Description: "Test schema migration, multiple migrations",
		Actions: []any{
			testUtils.ConfigureMigration{
				LensConfig: client.LensConfig{
					SourceSchemaVersionID:      "does not exist",
					DestinationSchemaVersionID: "also does not exist",
					Lens: model.Lens{
						Lenses: []model.LensModule{
							{
								Path: lenses.SetDefaultModulePath,
								Arguments: map[string]any{
									"dst":   "verified",
									"value": false,
								},
							},
						},
					},
				},
			},
			testUtils.ConfigureMigration{
				LensConfig: client.LensConfig{
					SourceSchemaVersionID:      "bafkreiebcgze3rs6j3g7gu65dwskdg5fn3qby5c6nqffhbdkcy2l5bbvp4",
					DestinationSchemaVersionID: "bafkreiexwzcpjuz3eaghcanr3fnmyc6el5w6i5ovhop5zfrqctucwlraba",
					Lens: model.Lens{
						Lenses: []model.LensModule{
							{
								Path: lenses.SetDefaultModulePath,
								Arguments: map[string]any{
									"dst":   "verified",
									"value": true,
								},
							},
						},
					},
				},
			},
			testUtils.GetCollections{
				FilterOptions: client.CollectionFetchOptions{
					IncludeInactive: immutable.Some(true),
				},
				ExpectedResults: []client.CollectionDescription{
					{
						ID:              1,
						SchemaVersionID: "does not exist",
					},
					{
						ID:              2,
						SchemaVersionID: "also does not exist",
						Sources: []any{
							&client.CollectionSource{
								SourceCollectionID: 1,
								Transform: immutable.Some(
									model.Lens{
										Lenses: []model.LensModule{
											{
												Path: lenses.SetDefaultModulePath,
												Arguments: map[string]any{
													"dst":   "verified",
													"value": false,
												},
											},
										},
									},
								),
							},
						},
					},
					{
						ID:              3,
						SchemaVersionID: "bafkreiebcgze3rs6j3g7gu65dwskdg5fn3qby5c6nqffhbdkcy2l5bbvp4",
					},
					{
						ID:              4,
						SchemaVersionID: "bafkreiexwzcpjuz3eaghcanr3fnmyc6el5w6i5ovhop5zfrqctucwlraba",
						Sources: []any{
							&client.CollectionSource{
								SourceCollectionID: 3,
								Transform: immutable.Some(
									model.Lens{
										Lenses: []model.LensModule{
											{
												Path: lenses.SetDefaultModulePath,
												Arguments: map[string]any{
													"dst":   "verified",
													"value": true,
												},
											},
										},
									},
								),
							},
						},
					},
				},
			},
		},
	}

	testUtils.ExecuteTestCase(t, test)
}

func TestSchemaMigrationReplacesExistingMigationBasedOnSourceID(t *testing.T) {
	test := testUtils.TestCase{
		Description: "Test schema migration, replace migration",
		Actions: []any{
			testUtils.ConfigureMigration{
				LensConfig: client.LensConfig{
					SourceSchemaVersionID:      "a",
					DestinationSchemaVersionID: "b",
					Lens: model.Lens{
						Lenses: []model.LensModule{
							{
								Path: lenses.SetDefaultModulePath,
								Arguments: map[string]any{
									"dst":   "verified",
									"value": false,
								},
							},
						},
					},
				},
			},
			testUtils.ConfigureMigration{
				// Replace the original migration with a new configuration
				LensConfig: client.LensConfig{
					SourceSchemaVersionID:      "a",
					DestinationSchemaVersionID: "c",
					Lens: model.Lens{
						Lenses: []model.LensModule{
							{
								Path: lenses.SetDefaultModulePath,
								Arguments: map[string]any{
									"dst":   "age",
									"value": 123,
								},
							},
						},
					},
				},
			},
			testUtils.GetCollections{
				FilterOptions: client.CollectionFetchOptions{
					IncludeInactive: immutable.Some(true),
				},
				ExpectedResults: []client.CollectionDescription{
					{
						ID:              1,
						SchemaVersionID: "a",
					},
					{
						ID:              2,
						SchemaVersionID: "b",
						Sources: []any{
							&client.CollectionSource{
								SourceCollectionID: 1,
								Transform: immutable.Some(
									model.Lens{
										Lenses: []model.LensModule{
											{
												Path: lenses.SetDefaultModulePath,
												Arguments: map[string]any{
													"dst":   "verified",
													"value": false,
												},
											},
										},
									},
								),
							},
						},
					},
					{
						ID:              3,
						SchemaVersionID: "c",
						Sources: []any{
							&client.CollectionSource{
								SourceCollectionID: 1,
								Transform: immutable.Some(
									model.Lens{
										Lenses: []model.LensModule{
											{
												Path: lenses.SetDefaultModulePath,
												Arguments: map[string]any{
													"dst":   "age",
													"value": float64(123),
												},
											},
										},
									},
								),
							},
						},
					},
				},
			},
		},
	}

	testUtils.ExecuteTestCase(t, test)
}
