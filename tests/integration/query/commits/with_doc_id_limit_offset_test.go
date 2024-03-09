// Copyright 2022 Democratized Data Foundation
//
// Use of this software is governed by the Business Source License
// included in the file licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with
// the Business Source License, use of this software will be governed
// by the Apache License, Version 2.0, included in the file
// licenses/APL.txt.

package commits

import (
	"testing"

	testUtils "github.com/sourcenetwork/defradb/tests/integration"
)

func TestQueryCommitsWithDocIDAndLimitAndOffset(t *testing.T) {
	test := testUtils.TestCase{
		Description: "Simple all commits query with docID, limit and offset",
		Actions: []any{
			updateUserCollectionSchema(),
			testUtils.CreateDoc{
				CollectionID: 0,
				Doc: `{
						"name":	"John",
						"age":	21
					}`,
			},
			testUtils.UpdateDoc{
				CollectionID: 0,
				DocID:        0,
				Doc: `{
					"age":	22
				}`,
			},
			testUtils.UpdateDoc{
				CollectionID: 0,
				DocID:        0,
				Doc: `{
					"age":	23
				}`,
			},
			testUtils.UpdateDoc{
				CollectionID: 0,
				DocID:        0,
				Doc: `{
					"age":	24
				}`,
			},
			testUtils.Request{
				Request: ` {
						commits(docID: "bae-f54b9689-e06e-5e3a-89b3-f3aee8e64ca7", limit: 2, offset: 1) {
							cid
						}
					}`,
				Results: []map[string]any{
					{
						"cid": "bafybeic45wkhxtpn3vgd2dmmohq76vw56qz3cpu3oorha3hf2w6qu7bpoa",
					},
					{
						"cid": "bafybeicwg56ddi7smy3j2kkv5y4yghvdrj3twqqafzdwtinbkw5mlpxwz4",
					},
				},
			},
		},
	}

	testUtils.ExecuteTestCase(t, test)
}
