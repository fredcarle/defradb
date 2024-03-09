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

func TestQueryCommitsWithUnknownDocID(t *testing.T) {
	test := testUtils.TestCase{
		Description: "Simple all commits query with unknown document ID",
		Actions: []any{
			updateUserCollectionSchema(),
			testUtils.CreateDoc{
				CollectionID: 0,
				Doc: `{
						"name":	"John",
						"age":	21
					}`,
			},
			testUtils.Request{
				Request: `query {
						commits(docID: "unknown document ID") {
							cid
						}
					}`,
				Results: []map[string]any{},
			},
		},
	}

	testUtils.ExecuteTestCase(t, test)
}

func TestQueryCommitsWithDocID(t *testing.T) {
	test := testUtils.TestCase{
		Description: "Simple all commits query with docID",
		Actions: []any{
			updateUserCollectionSchema(),
			testUtils.CreateDoc{
				CollectionID: 0,
				Doc: `{
						"name":	"John",
						"age":	21
					}`,
			},
			testUtils.Request{
				Request: `query {
						commits(docID: "bae-f54b9689-e06e-5e3a-89b3-f3aee8e64ca7") {
							cid
						}
					}`,
				Results: []map[string]any{
					{
						"cid": "bafybeicvpgfinf2m2jufbbcy5mhv6jca6in5k4fzx5op7xvvcmbp7sceaa",
					},
					{
						"cid": "bafybeib2espk2hq366wjnmazg45uvoswqbvf4plx7fgzayagxdn737onci",
					},
					{
						"cid": "bafybeigvpf62j7j2wbpid5iavzxielbhbsbbirmgzqkw3wpptdvysuztwi",
					},
				},
			},
		},
	}

	testUtils.ExecuteTestCase(t, test)
}

func TestQueryCommitsWithDocIDAndLinks(t *testing.T) {
	test := testUtils.TestCase{
		Description: "Simple all commits query with docID, with links",
		Actions: []any{
			updateUserCollectionSchema(),
			testUtils.CreateDoc{
				CollectionID: 0,
				Doc: `{
						"name":	"John",
						"age":	21
					}`,
			},
			testUtils.Request{
				Request: `query {
						commits(docID: "bae-f54b9689-e06e-5e3a-89b3-f3aee8e64ca7") {
							cid
							links {
								cid
								name
							}
						}
					}`,
				Results: []map[string]any{
					{
						"cid":   "bafybeicvpgfinf2m2jufbbcy5mhv6jca6in5k4fzx5op7xvvcmbp7sceaa",
						"links": []map[string]any{},
					},
					{
						"cid":   "bafybeib2espk2hq366wjnmazg45uvoswqbvf4plx7fgzayagxdn737onci",
						"links": []map[string]any{},
					},
					{
						"cid": "bafybeigvpf62j7j2wbpid5iavzxielbhbsbbirmgzqkw3wpptdvysuztwi",
						"links": []map[string]any{
							{
								"cid":  "bafybeicvpgfinf2m2jufbbcy5mhv6jca6in5k4fzx5op7xvvcmbp7sceaa",
								"name": "age",
							},
							{
								"cid":  "bafybeib2espk2hq366wjnmazg45uvoswqbvf4plx7fgzayagxdn737onci",
								"name": "name",
							},
						},
					},
				},
			},
		},
	}

	testUtils.ExecuteTestCase(t, test)
}

func TestQueryCommitsWithDocIDAndUpdate(t *testing.T) {
	test := testUtils.TestCase{
		Description: "Simple all commits query with docID, multiple results",
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
			testUtils.Request{
				Request: `query {
						commits(docID: "bae-f54b9689-e06e-5e3a-89b3-f3aee8e64ca7") {
							cid
							height
						}
					}`,
				Results: []map[string]any{
					{
						"cid":    "bafybeicwg56ddi7smy3j2kkv5y4yghvdrj3twqqafzdwtinbkw5mlpxwz4",
						"height": int64(2),
					},
					{
						"cid":    "bafybeicvpgfinf2m2jufbbcy5mhv6jca6in5k4fzx5op7xvvcmbp7sceaa",
						"height": int64(1),
					},
					{
						"cid":    "bafybeib2espk2hq366wjnmazg45uvoswqbvf4plx7fgzayagxdn737onci",
						"height": int64(1),
					},
					{
						"cid":    "bafybeidxnkwhuzmkdw5wuippru3tp74vcmz5jvcziambpjadxeathdh26a",
						"height": int64(2),
					},
					{
						"cid":    "bafybeigvpf62j7j2wbpid5iavzxielbhbsbbirmgzqkw3wpptdvysuztwi",
						"height": int64(1),
					},
				},
			},
		},
	}

	testUtils.ExecuteTestCase(t, test)
}

// This test is for documentation reasons only. This is not
// desired behaviour (first results includes link._head, second
// includes link._Name).
func TestQueryCommitsWithDocIDAndUpdateAndLinks(t *testing.T) {
	test := testUtils.TestCase{
		Description: "Simple all commits query with docID, multiple results and links",
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
			testUtils.Request{
				Request: `query {
						commits(docID: "bae-f54b9689-e06e-5e3a-89b3-f3aee8e64ca7") {
							cid
							links {
								cid
								name
							}
						}
					}`,
				Results: []map[string]any{
					{
						"cid": "bafybeicwg56ddi7smy3j2kkv5y4yghvdrj3twqqafzdwtinbkw5mlpxwz4",
						"links": []map[string]any{
							{
								"cid":  "bafybeicvpgfinf2m2jufbbcy5mhv6jca6in5k4fzx5op7xvvcmbp7sceaa",
								"name": "_head",
							},
						},
					},
					{
						"cid":   "bafybeicvpgfinf2m2jufbbcy5mhv6jca6in5k4fzx5op7xvvcmbp7sceaa",
						"links": []map[string]any{},
					},
					{
						"cid":   "bafybeib2espk2hq366wjnmazg45uvoswqbvf4plx7fgzayagxdn737onci",
						"links": []map[string]any{},
					},
					{
						"cid": "bafybeidxnkwhuzmkdw5wuippru3tp74vcmz5jvcziambpjadxeathdh26a",
						"links": []map[string]any{
							{
								"cid":  "bafybeigvpf62j7j2wbpid5iavzxielbhbsbbirmgzqkw3wpptdvysuztwi",
								"name": "_head",
							},
							{
								"cid":  "bafybeicwg56ddi7smy3j2kkv5y4yghvdrj3twqqafzdwtinbkw5mlpxwz4",
								"name": "age",
							},
						},
					},
					{
						"cid": "bafybeigvpf62j7j2wbpid5iavzxielbhbsbbirmgzqkw3wpptdvysuztwi",
						"links": []map[string]any{
							{
								"cid":  "bafybeicvpgfinf2m2jufbbcy5mhv6jca6in5k4fzx5op7xvvcmbp7sceaa",
								"name": "age",
							},
							{
								"cid":  "bafybeib2espk2hq366wjnmazg45uvoswqbvf4plx7fgzayagxdn737onci",
								"name": "name",
							},
						},
					},
				},
			},
		},
	}

	testUtils.ExecuteTestCase(t, test)
}
