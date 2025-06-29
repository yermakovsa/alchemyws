# Alchemy WS Client (Go)

A minimal and idiomatic Go client for Alchemy WebSocket API.

This library provides real-time access to Ethereum transaction events using Alchemy WebSocket interface. It supports subscriptions for:

* Mined transactions (hash or full payload)

* Pending transactions (hash or full payload)

Designed for use in Go services where performance, clarity, and reliability matter.

⚠️ **Warning:** This library is currently in active development (v0.10.0). The API is evolving, and improvements are ongoing.

## Features

* Subscribe to mined transactions as either:
  * transaction hashes (lightweight mode)

  * full transaction objects (full mode)

* Subscribe to pending transactions as either:

  * transaction hashes (lightweight mode)

  * full transaction objects (full mode)
* Subscribe to new block headers (newHeads):
  * receive full block metadata for every new chain head
  * includes support for handling chain reorganizations (reorgs)
* Non-blocking, buffered channels for event streams

* Graceful handling of decode failures and malformed messages

* Idiomatic Go API with minimal dependencies

## Installation

To install the Alchemy WS Client, use `go get`:

```bash
go get github.com/yermakovsa/alchemyws
```

Then import it in your Go code:

```bash
import github.com/yermakovsa/alchemyws
```

This library requires Go 1.18+.

## Usage

Here is a basic example to get you started with subscribing to mined and pending transactions using the Alchemy WS Client:

```go
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/yermakovsa/alchemyws"
)

func main() {
	// Replace with your actual Alchemy API key
	apiKey := "your-alchemy-api-key"

	client, err := alchemyws.NewAlchemyClient(apiKey, nil)
	if err != nil {
		log.Fatalf("failed to create client: %v", err)
	}
	defer client.Close()

	minedCh, err := client.SubscribeMined(alchemyws.MinedTxOptions{
		Addresses: []alchemyws.AddressFilter{
			{From: "0x28C6c06298d514Db089934071355E5743bf21d60"}, // Binance hot wallet ETH
		},
		HashesOnly: false,
	})
	if err != nil {
		log.Fatalf("failed to subscribe to mined transactions: %v", err)
	}

	pendingCh, err := client.SubscribePending(alchemyws.PendingTxOptions{
		FromAddress: []string{"0x28C6c06298d514Db089934071355E5743bf21d60"}, // Binance hot wallet ETH
		HashesOnly:  false,
	})
	if err != nil {
		log.Fatalf("failed to subscribe to pending transactions: %v", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	fmt.Println("Listening for mined and pending transactions...")

	for {
		select {
		case minedTx := <-minedCh:
			fmt.Printf("Mined transaction: %+v\n", minedTx)
		case pendingTx := <-pendingCh:
			fmt.Printf("Pending transaction: %+v\n", pendingTx)
		case <-ctx.Done():
			fmt.Println("Shutting down...")
			return
		}
	}
}
```

### Key Points:
* Use NewAlchemyClient to create a new client instance with your API key.

* Call SubscribeMined and SubscribePending to receive channels streaming mined and pending transactions.

* Handle incoming events by reading from the returned channels.

* Gracefully handle termination signals to close the client properly.
  
⚙️ For more complete usage examples, including subscriptions to newHeads, see the [examples/](examples) directory.

## API Reference

### `func NewAlchemyClient(apiKey string, logger *log.Logger) (*AlchemyClient, error)`

Creates a new Alchemy WebSocket client.

#### Parameters:
* `apiKey` - Your Alchemy API key (required).
* `logger` - Optional logger for debugging
  
#### Returns:
* `*AlchemyClient` - Client instance.
* `error` - Initialization error, if any.
  
### `func (a *AlchemyClient) SubscribeMined(opts MinedTxOptions) (<-chan MinedTxEvent, error)`

Subscribes to mined transaction events with optional filtering.

#### Parameters
* `opts` - Subscription options:
  * `Addresses []AddressFilter` - Filter transactions by sender (`From`) and receiver (`To`) addresses.
  * `IncludeRemoved bool` - If true, includes events for removed transactions (e.g., chain reorganizations).
  * `HashesOnly bool` - If true, only transaction hashes are sent instead of full transaction data.

#### Returns
* `<-chan MinedTxEvent` - Channel delivering mined transaction events.
* `error` - Subscription error, if any.
*   
### `func (a *AlchemyClient) SubscribePending(opts PendingTxOptions) (<-chan PendingTxEvent, error)`

Subscribes to pending transaction events with optional filtering.

#### Parameters
* `opts` - Subscription options:
  * `FromAddress []string` - Filter pending transactions by sender addresses.
  * `ToAddress []string` - Filter pending transactions by receiver addresses.
  * `HashesOnly bool` - If true, only transaction hashes are sent instead of full transaction data.

#### Returns
* `<-chan PendingTxEvent` - Channel delivering pending transaction events.
* `error` - Subscription error, if any.

### `func (a *AlchemyClient) SubscribeNewHeads() (<-chan NewHeadEvent, error)`

Subscribes to new block headers (`newHeads`) as they are added to the blockchain.

This subscription emits a new event each time a block is appended to the chain, including during chain reorganizations. In case of reorgs, multiple headers with the same block number may be emitted. The most recent one should be considered correct.

#### Parameters
*None*

#### Returns
* `<-chan NewHeadEvent` - Channel delivering new block header events.
* `error` - Subscription error, if any.

### `func (a *AlchemyClient) Close() error`

Closes the WebSocket connection and cleans up resources.

#### Returns
* `error` - Error on close failure, if any.

## License

This project is licensed under the [MIT License](LICENSE).

You are free to use, modify, and distribute this software in both personal and commercial projects, as long as you include the original license and copyright











