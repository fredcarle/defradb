// Copyright 2026 Democratized Data Foundation
//
// Use of this software is governed by the Business Source License
// included in the file licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with
// the Business Source License, use of this software will be governed
// by the Apache License, Version 2.0, included in the file
// licenses/APL.txt.

package p2p

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/fxamacker/cbor/v2"
	"github.com/ipld/go-ipld-prime/linking"
	cidlink "github.com/ipld/go-ipld-prime/linking/cid"
	"github.com/stretchr/testify/require"

	badgerds "github.com/dgraph-io/badger/v4"

	"github.com/sourcenetwork/corekv/badger"
	"github.com/sourcenetwork/immutable"

	"github.com/sourcenetwork/corekv/blockstore"
	"github.com/sourcenetwork/defradb/internal/core"
	coreblock "github.com/sourcenetwork/defradb/internal/core/block"
	"github.com/sourcenetwork/defradb/internal/core/crdt"
	"github.com/sourcenetwork/defradb/internal/datastore"
	"github.com/sourcenetwork/defradb/internal/db/lock"
	"github.com/sourcenetwork/defradb/internal/db/p2p/protocol"
	"github.com/sourcenetwork/defradb/internal/keys"
)

const testPeerID = "receiver-peer"

// stubDB satisfies the [DB] interface for the retry sweep, which only ever
// reaches for the multistore.
type stubDB struct {
	DB
	multistore *datastore.Multistore
}

func (s *stubDB) Multistore() *datastore.Multistore { return s.multistore }

// stubReplicatorProtocol answers every push with whatever the test's responder
// returns, standing in for the receiving node.
type stubReplicatorProtocol struct {
	mu        sync.Mutex
	calls     []string
	responder func(docID string, call int) error
}

func (s *stubReplicatorProtocol) SendRequest(
	_ context.Context,
	req protocol.PushLogRequest,
	_ string,
) (protocol.PushLogReply, error) {
	s.mu.Lock()
	s.calls = append(s.calls, req.DocID)
	n := len(s.calls)
	s.mu.Unlock()
	return protocol.PushLogReply{}, s.responder(req.DocID, n)
}

func (s *stubReplicatorProtocol) callsFor(docID string) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	count := 0
	for _, c := range s.calls {
		if c == docID {
			count++
		}
	}
	return count
}

// newRetryTestP2P builds a P2P whose stores hold a replicator with one pending
// retry record per docID, each with a single composite head so that retryDoc
// reaches the network call.
func newRetryTestP2P(t *testing.T, proto *stubReplicatorProtocol, docIDs ...string) *P2P {
	t.Helper()
	ctx := context.Background()
	rootstore, err := badger.NewDatastore("", badgerds.DefaultOptions("").WithInMemory(true))
	require.NoError(t, err)
	t.Cleanup(func() { _ = rootstore.Close() })
	multistore := datastore.NewMultistore(rootstore, lock.NewLockSet(), immutable.None[int]())

	lsys := cidlink.DefaultLinkSystem()
	lsys.SetWriteStorage(blockstore.NewIPLDStore(multistore.Blockstore()))

	for i, docID := range docIDs {
		docShortID := uint64(i + 1)

		require.NoError(t, multistore.Systemstore().Set(
			ctx,
			keys.NewDocIDToDocRefKey(docID).Bytes(),
			keys.EncodeDocRef(1, docShortID),
		))

		block := coreblock.Block{Delta: crdt.CRDT{DocCompositeDelta: &crdt.DocCompositeDelta{
			Priority:            1,
			CollectionVersionID: "collection-version",
			Status:              1,
		}}}
		link, err2 := lsys.Store(linking.LinkContext{Ctx: ctx}, coreblock.GetLinkPrototype(), block.GenerateNode())
		require.NoError(t, err2)

		headKey := keys.HeadstoreDocKey{DocShortID: docShortID, FieldID: core.COMPOSITE_NAMESPACE}.
			WithCid(link.(cidlink.Link).Cid) //nolint:forcetypeassert
		require.NoError(t, multistore.Headstore().Set(ctx, headKey.Bytes(), []byte{1}))

		require.NoError(t, multistore.Peerstore().Set(
			ctx,
			keys.NewReplicatorRetryDocIDKey(testPeerID, docID).Bytes(),
			[]byte{},
		))
	}

	require.NoError(t, multistore.Peerstore().Set(ctx, keys.NewReplicatorKey(testPeerID).Bytes(), []byte{}))

	return &P2P{
		ctx:                ctx,
		db:                 &stubDB{multistore: multistore},
		host:               &SimpleMockHost{},
		replicatorProtocol: proto,
		retryIntervals:     []time.Duration{30 * time.Second, 5 * time.Minute, 32 * time.Minute},
	}
}

func setRetryInfo(t *testing.T, p *P2P, rInfo retryInfo) {
	t.Helper()
	b, err := cbor.Marshal(rInfo)
	require.NoError(t, err)
	require.NoError(t, p.db.Multistore().Peerstore().Set(
		context.Background(),
		keys.NewReplicatorRetryIDKey(testPeerID).Bytes(),
		b,
	))
}

func getRetryInfo(t *testing.T, p *P2P) retryInfo {
	t.Helper()
	b, err := p.db.Multistore().Peerstore().Get(
		context.Background(),
		keys.NewReplicatorRetryIDKey(testPeerID).Bytes(),
	)
	require.NoError(t, err)
	rInfo := retryInfo{}
	require.NoError(t, cbor.Unmarshal(b, &rInfo))
	return rInfo
}

// backpressureWait classifies "rate limited" as transient, but the sweep's
// abort check tests only for "at capacity". A receiver that is uniformly rate
// limiting - the default 50/s limiter, the common case - therefore never
// aborts the sweep: every document burns the whole 150 x 100ms budget and the
// sweep walks on to the next one, so an N-document backlog takes N x 15s
// against a receiver that is rejecting all of them.
//
// The break decision should use the `transient` result from backpressureWait,
// not a second, narrower substring test.
func TestRetryReplicator_RateLimitedReceiverDoesNotAbortTheSweep(t *testing.T) {
	if testing.Short() {
		t.Skip("burns the full 15s per-document backpressure budget")
	}

	proto := &stubReplicatorProtocol{
		responder: func(docID string, _ int) error {
			if docID == "doc-2" {
				// Distinct, non-transient: proves the sweep reached this
				// document rather than aborting on doc-1.
				return errors.New("some other failure")
			}
			return errors.New("rate limited")
		},
	}
	p := newRetryTestP2P(t, proto, "doc-1", "doc-2")
	setRetryInfo(t, p, retryInfo{Retrying: true, NumRetries: 1})

	start := time.Now()
	p.retryReplicator(context.Background(), testPeerID)
	elapsed := time.Since(start)

	require.Equal(t, 151, proto.callsFor("doc-1"),
		"doc-1 should have exhausted its whole backpressure budget")
	require.Zero(t, proto.callsFor("doc-2"),
		fmt.Sprintf("sweep continued to doc-2 after %s of uniform rate limiting "+
			"instead of aborting; N documents cost N x 15s against a saturated receiver", elapsed))
}

// Leaving the sweep through ctx.Done() skips the switch at the end of
// retryReplicator, so the Retrying flag that addReplicatorAsRetrying persisted
// is never cleared - and nothing clears it at startup either. retryReplicators
// skips any record with Retrying == true, so this replicator's queued
// documents are never retried again for the life of the store.
//
// The window always existed at the top of the loop, but it was microseconds
// wide. The new backpressure wait sleeps for up to 150 x 2s per document, so a
// shutdown now lands inside it routinely.
func TestRetryReplicator_ShutdownDuringBackpressureWaitStrandsTheReplicator(t *testing.T) {
	proto := &stubReplicatorProtocol{
		responder: func(string, int) error { return errors.New("at capacity") },
	}
	p := newRetryTestP2P(t, proto, "doc-1")
	setRetryInfo(t, p, retryInfo{Retrying: true, NumRetries: 1, NextRetry: time.Now().Add(-time.Hour)})

	// Cancel while the sweep is parked in the 2s backpressure wait.
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(200 * time.Millisecond)
		cancel()
	}()
	p.retryReplicator(ctx, testPeerID)

	require.False(t, getRetryInfo(t, p).Retrying,
		"replicator left marked as retrying after the sweep returned")

	// And because it is still marked retrying, the sweeper will never pick it
	// up again: addReplicatorAsRetrying would have bumped NumRetries.
	p.retryReplicators(context.Background())
	time.Sleep(100 * time.Millisecond)
	require.Equal(t, 2, getRetryInfo(t, p).NumRetries,
		"retryReplicators skipped the stranded replicator")
}

// The "return to the first rung" branch reschedules at retryIntervals[0] but
// never resets NumRetries, which addReplicatorAsRetrying keeps incrementing.
// addReplicatorNextRetry picks the interval by NumRetries, so the de-escalation
// lasts exactly one wake-up: the next sweep that ends in the default branch
// jumps straight back to the maximum interval.
func TestAddReplicatorNextRetry_FirstRungDoesNotDeEscalate(t *testing.T) {
	ctx := context.Background()
	p := newRetryTestP2P(t, &stubReplicatorProtocol{
		responder: func(string, int) error { return nil },
	})
	// A replicator that escalated to the top rung while the receiver was saturated.
	setRetryInfo(t, p, retryInfo{NumRetries: len(p.retryIntervals)})

	// The sweep made progress: come back at the first interval.
	require.NoError(t, addReplicatorNextRetry(ctx, testPeerID, p.retryIntervals[:1], p.db.Multistore().Peerstore()))
	require.InDelta(t, float64(p.retryIntervals[0]), float64(time.Until(getRetryInfo(t, p).NextRetry)), float64(time.Second))

	// The next sweep that ends in the default branch should still be near the
	// first rung, since the receiver has been taking documents.
	require.NoError(t, addReplicatorNextRetry(ctx, testPeerID, p.retryIntervals, p.db.Multistore().Peerstore()))
	require.InDelta(t, float64(p.retryIntervals[0]), float64(time.Until(getRetryInfo(t, p).NextRetry)), float64(time.Second),
		"backoff jumped straight back to the escalated interval: NumRetries was never reset")
}

// Backpressure is classified by substring-matching an error string produced by
// a different implementation. Nothing in this repository emits "rate limited"
// or "at capacity", so there is no compile-time or test coupling: if the
// receiver rewords its nack, both the pacing and the abort silently revert to
// the pre-change burst behaviour with nothing failing.
func TestBackpressureWait_IsCoupledToUnownedErrorStrings(t *testing.T) {
	for _, nack := range []string{
		"push rejected: rate-limited, retry later",
		"push rejected: queue is full",
		"too many requests",
	} {
		_, transient := backpressureWait(errors.New(nack))
		require.True(t, transient, "reworded receiver nack %q is no longer recognised as backpressure", nack)
	}
}
