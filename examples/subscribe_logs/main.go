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

	// Subscribe to logs
	logs, err := client.SubscribeLogs(alchemyws.LogsFilter{
		Address: "0x7a250d5630B4cF539739dF2C5dAcb4c659F2488D", // Uniswap V2 Router
		Topics: [][]string{
			{"0xd78ad95fa46c994b6551d0da85fc275fe613ce37657fb8d5e6d3aa74fec0f52"}, // Swap(address,uint256,uint256,uint256,uint256,address)
		},
	})
	if err != nil {
		log.Fatalf("failed to subscribe to logs: %v", err)
	}

	// Process headers for 30 seconds
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	fmt.Println("Listening for logs...")

	for {
		select {
		case log := <-logs:
			fmt.Printf("New Log: %+v\n", log)
		case <-ctx.Done():
			fmt.Println("Stopping listener after 30s.")
			return
		}
	}
}
