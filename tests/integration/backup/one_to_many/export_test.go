// Copyright 2023 Democratized Data Foundation
//
// Use of this software is governed by the Business Source License
// included in the file licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with
// the Business Source License, use of this software will be governed
// by the Apache License, Version 2.0, included in the file
// licenses/APL.txt.

package backup

import (
	"testing"

	"github.com/sourcenetwork/defradb/client"
	testUtils "github.com/sourcenetwork/defradb/tests/integration"
)

func TestBackupExport_JustUserCollection_NoError(t *testing.T) {
	test := testUtils.TestCase{
		Actions: []any{
			testUtils.CreateDoc{
				CollectionID: 0,
				Doc:          `{"name": "John", "age": 30}`,
			},
			testUtils.BackupExport{
				Config: client.BackupConfig{
					Collections: []string{"User"},
				},
				ExpectedContent: `{"User":[{"_docID":"bae-e933420a-988a-56f8-8952-6c245aebd519","_docIDNew":"bae-e933420a-988a-56f8-8952-6c245aebd519","age":30,"name":"John"}]}`,
			},
		},
	}

	executeTestCase(t, test)
}

func TestBackupExport_AllCollectionsMultipleDocsAndDocUpdate_NoError(t *testing.T) {
	test := testUtils.TestCase{
		Actions: []any{
			testUtils.CreateDoc{
				CollectionID: 0,
				Doc:          `{"name": "John", "age": 30}`,
			},
			testUtils.CreateDoc{
				CollectionID: 0,
				Doc:          `{"name": "Bob", "age": 31}`,
			},
			testUtils.CreateDoc{
				CollectionID: 1,
				Doc:          `{"name": "John and the sourcerers' stone", "author": "bae-e933420a-988a-56f8-8952-6c245aebd519"}`,
			},
			testUtils.UpdateDoc{
				CollectionID: 0,
				DocID:        0,
				Doc:          `{"age": 31}`,
			},
			testUtils.BackupExport{
				ExpectedContent: `{"Book":[{"_docID":"bae-5cf2fec3-d8ed-50d5-8286-39109853d2da","_docIDNew":"bae-edeade01-2d21-5d6d-aadf-efc5a5279de5","author_id":"bae-807ea028-6c13-5f86-a72b-46e8b715a162","name":"John and the sourcerers' stone"}],"User":[{"_docID":"bae-0648f44e-74e8-593b-a662-3310ec278927","_docIDNew":"bae-0648f44e-74e8-593b-a662-3310ec278927","age":31,"name":"Bob"},{"_docID":"bae-e933420a-988a-56f8-8952-6c245aebd519","_docIDNew":"bae-807ea028-6c13-5f86-a72b-46e8b715a162","age":31,"name":"John"}]}`,
			},
		},
	}

	executeTestCase(t, test)
}

func TestBackupExport_AllCollectionsMultipleDocsAndMultipleDocUpdate_NoError(t *testing.T) {
	test := testUtils.TestCase{
		Actions: []any{
			testUtils.CreateDoc{
				CollectionID: 0,
				Doc:          `{"name": "John", "age": 30}`,
			},
			testUtils.CreateDoc{
				CollectionID: 0,
				Doc:          `{"name": "Bob", "age": 31}`,
			},
			testUtils.CreateDoc{
				CollectionID: 1,
				Doc:          `{"name": "John and the sourcerers' stone", "author": "bae-e933420a-988a-56f8-8952-6c245aebd519"}`,
			},
			testUtils.CreateDoc{
				CollectionID: 1,
				Doc:          `{"name": "Game of chains", "author": "bae-e933420a-988a-56f8-8952-6c245aebd519"}`,
			},
			testUtils.UpdateDoc{
				CollectionID: 0,
				DocID:        0,
				Doc:          `{"age": 31}`,
			},
			testUtils.BackupExport{
				ExpectedContent: `{"Book":[{"_docID":"bae-4399f189-138d-5d49-9e25-82e78463677b","_docIDNew":"bae-78a40f28-a4b8-5dca-be44-392b0f96d0ff","author_id":"bae-807ea028-6c13-5f86-a72b-46e8b715a162","name":"Game of chains"},{"_docID":"bae-5cf2fec3-d8ed-50d5-8286-39109853d2da","_docIDNew":"bae-edeade01-2d21-5d6d-aadf-efc5a5279de5","author_id":"bae-807ea028-6c13-5f86-a72b-46e8b715a162","name":"John and the sourcerers' stone"}],"User":[{"_docID":"bae-0648f44e-74e8-593b-a662-3310ec278927","_docIDNew":"bae-0648f44e-74e8-593b-a662-3310ec278927","age":31,"name":"Bob"},{"_docID":"bae-e933420a-988a-56f8-8952-6c245aebd519","_docIDNew":"bae-807ea028-6c13-5f86-a72b-46e8b715a162","age":31,"name":"John"}]}`,
			},
		},
	}

	executeTestCase(t, test)
}
