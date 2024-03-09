// Copyright 2023 Democratized Data Foundation
//
// Use of this software is governed by the Business Source License
// included in the file licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with
// the Business Source License, use of this software will be governed
// by the Apache License, Version 2.0, included in the file
// licenses/APL.txt.

package http

import (
	"encoding/json"

	"github.com/sourcenetwork/defradb/errors"
)

const (
	errFailedToLoadKeys       string = "failed to load given keys"
	errMethodIsNotImplemented string = "the method is not implemented"
)

// Errors returnable from this package.
//
// This list is incomplete. Undefined errors may also be returned.
// Errors returned from this package may be tested against these errors with errors.Is.
var (
	ErrNoListener             = errors.New("cannot serve with no listener")
	ErrNoEmail                = errors.New("email address must be specified for tls with autocert")
	ErrInvalidRequestBody     = errors.New("invalid request body")
	ErrStreamingNotSupported  = errors.New("streaming not supported")
	ErrMigrationNotFound      = errors.New("migration not found")
	ErrMissingRequest         = errors.New("missing request")
	ErrInvalidTransactionId   = errors.New("invalid transaction id")
	ErrP2PDisabled            = errors.New("p2p network is disabled")
	ErrMethodIsNotImplemented = errors.New(errMethodIsNotImplemented)
)

type errorResponse struct {
	Error error `json:"error"`
}

func (e errorResponse) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]any{"error": e.Error.Error()})
}

func (e *errorResponse) UnmarshalJSON(data []byte) error {
	var out map[string]any
	if err := json.Unmarshal(data, &out); err != nil {
		return err
	}
	e.Error = parseError(out["error"])
	return nil
}

func NewErrFailedToLoadKeys(inner error, publicKeyPath, privateKeyPath string) error {
	return errors.Wrap(
		errFailedToLoadKeys,
		inner,
		errors.NewKV("PublicKeyPath", publicKeyPath),
		errors.NewKV("PrivateKeyPath", privateKeyPath),
	)
}
