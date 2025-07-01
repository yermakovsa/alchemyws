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

	// Subscribe to new pending transactions
	newPendingTransactions, err := client.SubscribeNewPendingTransactions()
	if err != nil {
		log.Fatalf("failed to subscribe to new pending transactions: %v", err)
	}

	// Process new pending transactions for 30 seconds
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	fmt.Println("Listening for new pending transactions...")

	for {
		select {
		case newPendingTransaction := <-newPendingTransactions:
			fmt.Printf("New Pending Transaction Hash: %+v\n", newPendingTransaction)
		case <-ctx.Done():
			fmt.Println("Stopping listener after 30s.")
			return
		}
	}
}
