package alchemyws

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"sync/atomic"

	"github.com/coder/websocket"
)

const (
	AlchemyWSURL   = "wss://eth-mainnet.g.alchemy.com/v2/"
	JSONRPCVersion = "2.0"
	MethodMined    = "alchemy_minedTransactions"
	MethodPending  = "alchemy_pendingTransactions"
)

type WSConn interface {
	Read(ctx context.Context) (websocket.MessageType, []byte, error)
	Write(ctx context.Context, typ websocket.MessageType, p []byte) error
	Close(code websocket.StatusCode, reason string) error
}

type AlchemyClient struct {
	apiKey         string
	conn           WSConn
	ctx            context.Context
	cancel         context.CancelFunc
	logger         *log.Logger
	writeMu        sync.Mutex
	subscribers    map[string]chan json.RawMessage
	subsMu         sync.RWMutex
	pendingSubs    map[int]chan string
	pendingMu      sync.Mutex
	requestCounter int64
}

// NewAlchemyClient connects to Alchemy's WebSocket API and starts a single read loop
func NewAlchemyClient(apiKey string, logger *log.Logger) (*AlchemyClient, error) {
	if logger == nil {
		logger = log.Default()
	}
	ctx, cancel := context.WithCancel(context.Background())
	conn, _, err := websocket.Dial(ctx, AlchemyWSURL+apiKey, nil)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to connect: %w", err)
	}

	client := &AlchemyClient{
		apiKey:      apiKey,
		conn:        conn,
		ctx:         ctx,
		cancel:      cancel,
		logger:      logger,
		subscribers: make(map[string]chan json.RawMessage),
		pendingSubs: make(map[int]chan string),
	}

	go client.readLoop()

	return client, nil
}

// newAlchemyClientWithConn creates a client using a custom WebSocket connection.
// Intended for internal testing only
func newAlchemyClientWithConn(apiKey string, logger *log.Logger, conn WSConn) (*AlchemyClient, error) {
	if logger == nil {
		logger = log.Default()
	}
	ctx, cancel := context.WithCancel(context.Background())

	client := &AlchemyClient{
		apiKey:      apiKey,
		conn:        conn,
		ctx:         ctx,
		cancel:      cancel,
		logger:      logger,
		subscribers: make(map[string]chan json.RawMessage),
		pendingSubs: make(map[int]chan string),
	}

	go client.readLoop()

	return client, nil
}

// Close shuts down the WebSocket connection
func (a *AlchemyClient) Close() error {
	a.cancel()
	return a.conn.Close(websocket.StatusNormalClosure, "client closed")
}

// SubscribeMined subscribes to mined transactions
func (a *AlchemyClient) SubscribeMined(opts MinedTxOptions) (<-chan MinedTxEvent, error) {
	return subscribeGeneric[MinedTxEvent](a, MethodMined, opts, func(data json.RawMessage) (any, error) {
		var msg MinedTxEvent
		return msg, json.Unmarshal(data, &msg)
	})
}

// SubscribePending subscribes to pending transactions
func (a *AlchemyClient) SubscribePending(opts PendingTxOptions) (<-chan PendingTxEvent, error) {
	return subscribeGeneric[PendingTxEvent](a, MethodPending, opts, func(data json.RawMessage) (any, error) {
		var hash string
		if err := json.Unmarshal(data, &hash); err == nil {
			return PendingTxEvent{Hash: hash}, nil
		}
		var tx Transaction
		if err := json.Unmarshal(data, &tx); err != nil {
			return nil, err
		}
		return PendingTxEvent{Transaction: tx}, nil
	})
}

// Core generic subscription handler
func subscribeGeneric[T any](a *AlchemyClient, method string, opts any, decode func(json.RawMessage) (any, error)) (<-chan T, error) {
	id := int(atomic.AddInt64(&a.requestCounter, 1))
	req := RPCRequest{
		JSONRPC: JSONRPCVersion,
		ID:      id,
		Method:  "eth_subscribe",
		Params:  []any{method, opts},
	}

	payload, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	respCh := make(chan string, 1)
	a.setPendingSub(id, respCh)

	a.writeMu.Lock()
	err = a.conn.Write(a.ctx, websocket.MessageText, payload)
	a.writeMu.Unlock()
	if err != nil {
		return nil, err
	}

	subID := <-respCh
	out := make(chan T, 100)
	raw := make(chan json.RawMessage, 100)
	a.setSubscriber(subID, raw)

	go decodeLoop(a, out, raw, decode)

	return out, nil
}

func decodeLoop[T any](a *AlchemyClient, out chan T, raw chan json.RawMessage, decode func(json.RawMessage) (any, error)) {
	defer close(out)
	for {
		select {
		case <-a.ctx.Done():
			return
		case msg, ok := <-raw:
			if !ok {
				return
			}
			parsed, err := decode(msg)
			if err != nil {
				a.logger.Printf("[alchemyws] decode error: %v", err)
				continue
			}
			out <- parsed.(T)
		}
	}
}

// Internal read loop to dispatch all WebSocket messages
func (a *AlchemyClient) readLoop() {
	defer func() {
		if r := recover(); r != nil {
			a.logger.Printf("panic recovered: %v", r)
		}
	}()

	for {
		select {
		case <-a.ctx.Done():
			a.logger.Printf("[alchemyws] context canceled, stopping read loop")
			return
		default:
			_, data, err := a.conn.Read(a.ctx)
			if err != nil {
				if a.ctx.Err() != nil {
					a.logger.Printf("[alchemyws] context closed: %v", a.ctx.Err())
					return
				}
				a.logger.Printf("[alchemyws] read error: %v", err)
				return
			}

			var envelope RPCEnvelope
			if err := json.Unmarshal(data, &envelope); err != nil {
				a.logger.Printf("[alchemyws] unmarshal error: %v", err)
				continue
			}

			switch {
			case envelope.ID != 0 && envelope.Result != "":
				a.handleSubscriptionResponse(envelope.ID, envelope.Result)
			case envelope.Method == "eth_subscription":
				a.handleEventMessage(envelope.Params)
			default:
				a.logger.Printf("[alchemyws] unknown message: %s", data)
			}
		}
	}
}

// Handles initial subscription ID response
func (a *AlchemyClient) handleSubscriptionResponse(id int, subID string) {
	if ch, ok := a.getAndDeletePendingSub(id); ok {
		ch <- subID
	}
}

func (a *AlchemyClient) handleEventMessage(params json.RawMessage) {
	var msg SubscriptionMessage
	if err := json.Unmarshal(params, &msg); err != nil {
		a.logger.Printf("[alchemyws] event unmarshal error: %v", err)
		return
	}
	if ch, ok := a.getSubscriber(msg.Subscription); ok {
		ch <- msg.Result
	}
}

func (a *AlchemyClient) setSubscriber(subID string, ch chan json.RawMessage) {
	a.subsMu.Lock()
	defer a.subsMu.Unlock()
	a.subscribers[subID] = ch
}

func (a *AlchemyClient) getSubscriber(subID string) (chan json.RawMessage, bool) {
	a.subsMu.RLock()
	defer a.subsMu.RUnlock()
	ch, ok := a.subscribers[subID]
	return ch, ok
}

func (a *AlchemyClient) setPendingSub(id int, ch chan string) {
	a.pendingMu.Lock()
	defer a.pendingMu.Unlock()
	a.pendingSubs[id] = ch
}

func (a *AlchemyClient) getAndDeletePendingSub(id int) (chan string, bool) {
	a.pendingMu.Lock()
	defer a.pendingMu.Unlock()
	ch, ok := a.pendingSubs[id]
	if ok {
		delete(a.pendingSubs, id)
	}
	return ch, ok
}
