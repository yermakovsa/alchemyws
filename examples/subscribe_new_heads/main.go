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

	// Subscribe to new block headers
	headers, err := client.SubscribeNewHeads()
	if err != nil {
		log.Fatalf("failed to subscribe to mined transactions: %v", err)
	}

	// Process headers for 30 seconds
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	fmt.Println("Listening for new block headers...")

	for {
		select {
		case head := <-headers:
			fmt.Printf("New Block Head: %+v\n", head)
		case <-ctx.Done():
			fmt.Println("Stopping listener after 30s.")
			return
		}
	}
}
