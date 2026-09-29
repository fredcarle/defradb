// Copyright 2025 Democratized Data Foundation
//
// Use of this software is governed by the Business Source License
// included in the file licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with
// the Business Source License, use of this software will be governed
// by the Apache License, Version 2.0, included in the file
// licenses/APL.txt.

package message

import (
	"bytes"
	"context"
	"crypto/rand"
	"testing"
	"time"

	"github.com/fxamacker/cbor/v2"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/peer"

	"github.com/sourcenetwork/defradb/client"

	"github.com/stretchr/testify/require"
)

func TestReceive_StreamLargerThanMax_ReturnsErrMessageTooLarge(t *testing.T) {
	stream := bytes.NewReader(make([]byte, maxMessageSize+1))
	err := Receive(stream, "some peer ID", nil, &MetaData{})
	require.ErrorIs(t, err, ErrMessageTooLarge)
}

type closeRecordingStream struct {
	*bytes.Reader
	closed bool
}

func (s *closeRecordingStream) Close() error {
	s.closed = true
	return nil
}

// go-libp2p releases a stream's resource-manager reservation only when the
// local side closes it. Every inbound stream handler hands its stream to
// Receive and never touches it again, so Receive must close it — otherwise each
// accepted stream holds one of the per-(protocol, peer) inbound slots forever
// and the peer's 97th stream is refused.
func TestReceive_ClosesTheStreamWhenItCan(t *testing.T) {
	stream := &closeRecordingStream{Reader: bytes.NewReader([]byte("not cbor"))}
	_ = Receive(stream, "some peer ID", nil, &MetaData{})
	require.True(t, stream.closed, "inbound stream left open after Receive")
}

type fakeHost struct{ client.Host }

func (fakeHost) ID() string                                         { return "sender" }
func (fakeHost) Pubkey() ([]byte, error)                            { return []byte{1}, nil }
func (fakeHost) Sign([]byte) ([]byte, error)                        { return []byte{2}, nil }
func (fakeHost) Send(context.Context, []byte, string, string) error { return nil }

// nackingProto answers every request with a reply that carries an error
// string, the way a saturated receiver nacks a PushLog.
type nackingProto struct {
	host client.Host
	nack string
}

func (p *nackingProto) Host() client.Host { return p.host }
func (p *nackingProto) SetResponseChan(messageID string, ch chan Message) {
	reply := &MetaData{}
	reply.SetMessageID(messageID)
	reply.SetErrMessage(p.nack)
	ch <- reply
}
func (p *nackingProto) DeleteResponseChan(string)                   {}
func (p *nackingProto) GetResponseChan(string) (chan Message, bool) { return nil, false }

// A receiver's nack must surface as Send's error; reading the request's own
// (empty) error string instead reports the rejected push as delivered, and the
// replicator then never writes a retry record for it.
func TestSend_ReturnsTheReplysErrMessage(t *testing.T) {
	proto := &nackingProto{host: fakeHost{}, nack: "at capacity: receiver is saturated, back off"}
	_, err := Send[*MetaData](context.Background(), proto, &MetaData{}, "receiver", "/defradb/rep_req/0.0.1")
	require.EqualError(t, err, proto.nack)
}

// capturingProto keeps the response channel Send registered, standing in for an
// inbound handler goroutine that has already read the channel out of the map
// (message.go:127) and is about to deliver its reply into it.
type capturingProto struct {
	host client.Host
	ch   chan Message
}

func (p *capturingProto) Host() client.Host                           { return p.host }
func (p *capturingProto) SetResponseChan(_ string, ch chan Message)   { p.ch = ch }
func (p *capturingProto) DeleteResponseChan(string)                   {}
func (p *capturingProto) GetResponseChan(string) (chan Message, bool) { return nil, false }

// A reply that arrives after Send has given up must not land on a closed
// channel: Receive sends into a channel it read from the map before the
// timeout removed the entry, so closing on the timeout path is a send on a
// closed channel, which is a data race under -race and a panic without it.
func TestSend_LateReplyAfterTimeoutMustNotPanic(t *testing.T) {
	proto := &capturingProto{host: fakeHost{}}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	_, err := Send[*MetaData](ctx, proto, &MetaData{}, "receiver", "/defradb/rep_req/0.0.1")
	require.ErrorIs(t, err, ErrResponseTimeout)

	require.NotPanics(t, func() { proto.ch <- &MetaData{} })
}

// signingHost is a host backed by a real key pair, so messages it signs pass
// verifyMessage on the Receive side.
type signingHost struct {
	client.Host
	priv crypto.PrivKey
	id   peer.ID
}

func newSigningHost(t *testing.T) *signingHost {
	priv, pub, err := crypto.GenerateEd25519Key(rand.Reader)
	require.NoError(t, err)
	id, err := peer.IDFromPublicKey(pub)
	require.NoError(t, err)
	return &signingHost{priv: priv, id: id}
}

func (h *signingHost) ID() string { return h.id.String() }
func (h *signingHost) Pubkey() ([]byte, error) {
	return crypto.MarshalPublicKey(h.priv.GetPublic())
}
func (h *signingHost) Sign(b []byte) ([]byte, error) { return h.priv.Sign(b) }

// occupiedProto stands in for the window between an inbound handler reading the
// response channel out of the map (message.go:127) and the owner of that
// channel calling DeleteResponseChan: the channel is still registered and
// already holds the one reply it is buffered for.
type occupiedProto struct {
	host client.Host
	ch   chan Message
}

func (p *occupiedProto) Host() client.Host                           { return p.host }
func (p *occupiedProto) SetResponseChan(string, chan Message)        {}
func (p *occupiedProto) DeleteResponseChan(string)                   {}
func (p *occupiedProto) GetResponseChan(string) (chan Message, bool) { return p.ch, true }

// A second reply carrying an already-seen message ID must not wedge the handler
// goroutine. The response channel is buffered at one, so the unconditional
// `messageChan <- m` blocks forever on the second delivery, and because
// `defer closer.Close()` only runs when Receive returns, the stream is never
// closed either - reinstating the per-(protocol, peer) reservation leak this
// change exists to fix.
func TestReceive_DuplicateReplyMustNotBlockTheHandler(t *testing.T) {
	host := newSigningHost(t)

	m := &MetaData{}
	require.NoError(t, signAndSetMetaData(host, m))
	b, err := cbor.Marshal(m)
	require.NoError(t, err)

	// The channel Send registered, already holding the first reply.
	ch := make(chan Message, 1)
	ch <- &MetaData{}

	stream := &closeRecordingStream{Reader: bytes.NewReader(b)}
	done := make(chan error, 1)
	go func() {
		done <- Receive(stream, host.ID(), &occupiedProto{host: host, ch: ch}, &MetaData{})
	}()

	select {
	case err := <-done:
		require.NoError(t, err)
	case <-time.After(time.Second):
		require.Fail(t, "Receive blocked on a full response channel",
			"the handler goroutine is wedged and the stream is still open (closed=%v)", stream.closed)
	}
}
