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

import "github.com/sourcenetwork/defradb/errors"

const (
	errMissingHostMethod    = "missing required host method"
	errInvalidHostObject    = "invalid host object"
	errInvalidTransactionId = "invalid transaction id"
	errGetAllNotSupported   = "getAll() is not supported by DefraDB blockstore"
)

var (
	ErrMissingHostMethod    = errors.New(errMissingHostMethod)
	ErrInvalidHostObject    = errors.New(errInvalidHostObject)
	ErrInvalidTransactionId = errors.New(errInvalidTransactionId)
	ErrGetAllNotSupported   = errors.New(errGetAllNotSupported)
)
