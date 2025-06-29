package alchemyws

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	fakeAPIKey      = "fake-api-key"
	minedSubID      = "sub-id-123"
	pendingHashSub  = "sub-id-456"
	pendingFullSub  = "sub-id-789"
	timeoutDuration = 100 * time.Millisecond
)

type mockConn struct {
	readQueue  chan []byte
	writeQueue chan []byte
	closed     bool
	writeMu    sync.Mutex
}

func newMockConn() *mockConn {
	return &mockConn{
		readQueue:  make(chan []byte, 10),
		writeQueue: make(chan []byte, 10),
	}
}

func (m *mockConn) Read(ctx context.Context) (websocket.MessageType, []byte, error) {
	select {
	case <-ctx.Done():
		return websocket.MessageText, nil, ctx.Err()
	case msg := <-m.readQueue:
		return websocket.MessageText, msg, nil
	}
}

func (m *mockConn) Write(ctx context.Context, typ websocket.MessageType, p []byte) error {
	m.writeMu.Lock()
	defer m.writeMu.Unlock()
	if m.closed {
		return errors.New("connection closed")
	}
	m.writeQueue <- p
	return nil
}

func (m *mockConn) Close(code websocket.StatusCode, reason string) error {
	m.writeMu.Lock()
	defer m.writeMu.Unlock()
	m.closed = true
	return nil
}

// --- Helper Functions ---

func marshalRaw(v any) json.RawMessage {
	b, _ := json.Marshal(v)
	return json.RawMessage(b)
}

func waitFor[T any](t *testing.T, ch <-chan T) T {
	select {
	case val := <-ch:
		return val
	case <-time.After(timeoutDuration):
		t.Fatal("timeout waiting for channel value")
		return *new(T)
	}
}

func sendToConn(t *testing.T, conn *mockConn, msg any) {
	b, err := json.Marshal(msg)
	require.NoError(t, err)
	conn.readQueue <- b
}

func subscriptionEnvelope(subID string, result any) rPCEnvelope {
	msg := subscriptionMessage{
		Subscription: subID,
		Result:       marshalRaw(result),
	}
	return rPCEnvelope{
		Method: "eth_subscription",
		Params: marshalRaw(msg),
	}
}

// --- Test Cases ---

func TestSubscribeMined_Success(t *testing.T) {
	conn := newMockConn()
	client, _ := newAlchemyClientWithConn(fakeAPIKey, nil, conn)

	event := MinedTxEvent{Transaction: Transaction{Hash: "0x123"}}

	go func() {
		sendToConn(t, conn, rPCEnvelope{ID: 1, Result: minedSubID})
		time.Sleep(5 * time.Millisecond)
		sendToConn(t, conn, subscriptionEnvelope(minedSubID, event))
	}()

	ch, err := client.SubscribeMined(MinedTxOptions{})
	require.NoError(t, err)

	result := waitFor(t, ch)
	assert.Equal(t, "0x123", result.Transaction.Hash)
}

func TestSubscribeMined_DecodeError(t *testing.T) {
	conn := newMockConn()
	client, _ := newAlchemyClientWithConn(fakeAPIKey, nil, conn)

	go func() {
		sendToConn(t, conn, rPCEnvelope{ID: 1, Result: minedSubID})
		time.Sleep(5 * time.Millisecond)
		msg := subscriptionMessage{Subscription: minedSubID, Result: json.RawMessage(`{invalid}`)}
		sendToConn(t, conn, rPCEnvelope{Method: "eth_subscription", Params: marshalRaw(msg)})
	}()

	ch, err := client.SubscribeMined(MinedTxOptions{})
	require.NoError(t, err)

	select {
	case <-ch:
		t.Fatal("unexpected message received on decode error")
	case <-time.After(timeoutDuration):
	}
}

func TestSubscribePending_HashOnly(t *testing.T) {
	conn := newMockConn()
	client, _ := newAlchemyClientWithConn(fakeAPIKey, nil, conn)

	hash := "0xabc"

	go func() {
		sendToConn(t, conn, rPCEnvelope{ID: 1, Result: pendingHashSub})
		time.Sleep(5 * time.Millisecond)
		sendToConn(t, conn, subscriptionEnvelope(pendingHashSub, hash))
	}()

	ch, err := client.SubscribePending(PendingTxOptions{})
	require.NoError(t, err)

	result := waitFor(t, ch)
	assert.Equal(t, hash, result.Hash)
}

func TestSubscribePending_FullTransaction(t *testing.T) {
	conn := newMockConn()
	client, _ := newAlchemyClientWithConn(fakeAPIKey, nil, conn)

	tx := Transaction{Hash: "0xdeadbeef"}

	go func() {
		sendToConn(t, conn, rPCEnvelope{ID: 1, Result: pendingFullSub})
		time.Sleep(5 * time.Millisecond)
		sendToConn(t, conn, subscriptionEnvelope(pendingFullSub, tx))
	}()

	ch, err := client.SubscribePending(PendingTxOptions{})
	require.NoError(t, err)

	result := waitFor(t, ch)
	assert.Equal(t, tx.Hash, result.Transaction.Hash)
}

func TestSubscribePending_DecodeError(t *testing.T) {
	conn := newMockConn()
	client, _ := newAlchemyClientWithConn(fakeAPIKey, nil, conn)

	go func() {
		sendToConn(t, conn, rPCEnvelope{ID: 1, Result: "sub-pending-err"})
		time.Sleep(5 * time.Millisecond)
		msg := subscriptionMessage{Subscription: "sub-pending-err", Result: json.RawMessage(`{malformed`)}
		sendToConn(t, conn, rPCEnvelope{Method: "eth_subscription", Params: marshalRaw(msg)})
	}()

	ch, err := client.SubscribePending(PendingTxOptions{})
	require.NoError(t, err)

	select {
	case <-ch:
		t.Fatal("unexpected value in case of decode failure")
	case <-time.After(timeoutDuration):
	}
}

func TestSubscribeMinedAndPending_Concurrent(t *testing.T) {
	const testTimeout = time.Second

	conn := newMockConn()
	client, _ := newAlchemyClientWithConn(fakeAPIKey, nil, conn)

	minedEvents := []MinedTxEvent{
		{Transaction: Transaction{Hash: "0x1"}},
		{Transaction: Transaction{Hash: "0x2"}},
		{Transaction: Transaction{Hash: "0x3"}},
	}
	pendingEvents := []PendingTxEvent{
		{Hash: "0xa"},
		{Hash: "0xb"},
		{Hash: "0xc"},
		{Hash: "0xd"},
	}

	send := func(env rPCEnvelope) {
		b, err := json.Marshal(env)
		require.NoError(t, err)
		conn.readQueue <- b
	}

	// Simulate server responses
	go func() {
		send(rPCEnvelope{ID: 1, Result: "sub-id-mined"})
		time.Sleep(10 * time.Millisecond)
		send(rPCEnvelope{ID: 2, Result: "sub-id-pending"})

		time.Sleep(10 * time.Millisecond) // Simulate slight delay

		// Interleave events
		for i := 0; i < max(len(minedEvents), len(pendingEvents)); i++ {
			if i < len(minedEvents) {
				send(subscriptionEnvelope("sub-id-mined", minedEvents[i]))
			}
			if i < len(pendingEvents) {
				send(subscriptionEnvelope("sub-id-pending", pendingEvents[i].Hash))
			}
		}
	}()

	minedCh, errMined := client.SubscribeMined(MinedTxOptions{})
	pendingCh, errPending := client.SubscribePending(PendingTxOptions{})
	require.NoError(t, errMined)
	require.NoError(t, errPending)

	var (
		gotMined   []MinedTxEvent
		gotPending []PendingTxEvent
	)

	timeout := time.After(testTimeout)
LOOP:
	for {
		select {
		case m := <-minedCh:
			gotMined = append(gotMined, m)
		case p := <-pendingCh:
			gotPending = append(gotPending, p)
		case <-timeout:
			break LOOP
		}
		if len(gotMined) == len(minedEvents) && len(gotPending) == len(pendingEvents) {
			break
		}
	}

	t.Run("validate mined events", func(t *testing.T) {
		assert.Equal(t, len(minedEvents), len(gotMined))
		for i, want := range minedEvents {
			assert.Equal(t, want.Transaction.Hash, gotMined[i].Transaction.Hash)
		}
	})

	t.Run("validate pending events", func(t *testing.T) {
		assert.Equal(t, len(pendingEvents), len(gotPending))
		for i, want := range pendingEvents {
			assert.Equal(t, want.Hash, gotPending[i].Hash)
		}
	})
}

func TestSubscribeNewHeads_ReceivesBlockHeader(t *testing.T) {
	conn := newMockConn()
	client, _ := newAlchemyClientWithConn(fakeAPIKey, nil, conn)

	head := NewHeadEvent{
		Number:     "0x2b",
		ParentHash: "0xbase",
	}

	go func() {
		sendToConn(t, conn, rPCEnvelope{ID: 1, Result: "0xsub-min"})
		time.Sleep(5 * time.Millisecond)
		sendToConn(t, conn, subscriptionEnvelope("0xsub-min", head))
	}()

	ch, err := client.SubscribeNewHeads()
	require.NoError(t, err)

	result := waitFor(t, ch)
	assert.Equal(t, head.Number, result.Number)
	assert.Equal(t, head.ParentHash, result.ParentHash)
}
