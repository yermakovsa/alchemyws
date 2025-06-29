package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/yermakovsa/alchemyws"
)

func main() {
	// Replace with your actual Alchemy API key
	apiKey := "your-alchemy-api-key"

	// Create a new Alchemy WebSocket client
	client, err := alchemyws.NewAlchemyClient(apiKey, nil)
	if err != nil {
		log.Fatalf("failed to create client: %v", err)
	}
	defer client.Close()

	// Subscribe to mined transactions
	minedTransactions, err := client.SubscribeMined(alchemyws.MinedTxOptions{
		Addresses: []alchemyws.AddressFilter{
			{From: "0x28C6c06298d514Db089934071355E5743bf21d60"}, // Binance ETH hot wallet
		},
		HashesOnly: false,
	})
	if err != nil {
		log.Fatalf("failed to subscribe to mined transactions: %v", err)
	}

	// Subscribe to pending transactions
	pendingTransactions, err := client.SubscribePending(alchemyws.PendingTxOptions{
		FromAddress: []string{"0x28C6c06298d514Db089934071355E5743bf21d60"}, // Binance ETH hot wallet
		HashesOnly:  false,
	})
	if err != nil {
		log.Fatalf("failed to subscribe to pending transactions: %v", err)
	}

	// Process mined transactions for 30 seconds
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	fmt.Println("Listening for mined and pending transactions...")

	for {
		select {
		case minedTransaction := <-minedTransactions:
			fmt.Printf("Mined tx: %+v\n", minedTransaction)
		case pendingTransaction := <-pendingTransactions:
			fmt.Printf("Pending tx: %+v\n", pendingTransaction)
		case <-ctx.Done():
			fmt.Println("Stopping listener after 30s.")
			return
		}
	}
}
