//go:build tools
// +build tools

// ethstats-client is a manual testing tool for the ethstats server.
// It reuses the test client from pkg/ethstats/testutil to ensure
// consistency between manual and automated testing.
package main

import (
	"context"
	"crypto/rand"
	"flag"
	"fmt"
	"log"
	"math/big"
	"os"
	"os/signal"
	"strings"
	"time"

	"github.com/ethpandaops/xatu/pkg/ethstats/protocol"
	"github.com/ethpandaops/xatu/pkg/ethstats/testutil"
)

func main() {
	var ethstatsURL string

	flag.StringVar(&ethstatsURL, "ethstats", "", "Ethstats URL in format: username:password@host:port")
	flag.Parse()

	if ethstatsURL == "" {
		log.Fatal("--ethstats flag is required")
	}

	// Parse the ethstats URL
	parts := strings.SplitN(ethstatsURL, "@", 2)
	if len(parts) != 2 {
		log.Fatal("Invalid ethstats URL format. Expected: username:password@host:port")
	}

	credentials := parts[0]
	address := parts[1]

	// Extract username from credentials (for node ID)
	credParts := strings.SplitN(credentials, ":", 2)
	if len(credParts) != 2 {
		log.Fatal("Invalid credentials format. Expected: username:password")
	}

	nodeID := credParts[0]

	// Build server URL
	serverURL := "http://" + address

	log.Printf("Connecting to ethstats server at %s as node '%s'", serverURL, nodeID)

	// Create test client
	client := testutil.NewTestClient(serverURL, nodeID, credentials)

	// Setup signal handling
	ctx, cancel := context.WithCancel(context.Background())
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt)

	// Connect to server
	if err := client.Connect(ctx); err != nil {
		cancel()
		log.Fatalf("Failed to connect: %v", err)
	}

	defer func() {
		client.Close()
		cancel()
	}()

	log.Println("Connected to server")

	// Send hello message
	if err := client.SendHello(); err != nil {
		log.Printf("Failed to send hello: %v", err)

		return
	}

	log.Println("Hello message sent, waiting for authentication...")

	// Wait for ready message
	if _, err := client.WaitForMessage("ready", 5*time.Second); err != nil {
		log.Printf("Failed to authenticate: %v", err)

		return
	}

	log.Println("Successfully authenticated")

	// Start sending events
	go sendRandomEvents(ctx, client)

	// Handle ping messages in the background
	go handlePings(ctx, client)

	// Wait for interrupt signal
	<-sigChan
	log.Println("Shutting down...")
}

func randInt(maxValue int) int {
	n, err := rand.Int(rand.Reader, big.NewInt(int64(maxValue)))
	if err != nil {
		return 0
	}

	return int(n.Int64())
}

func randUint64(maxValue uint64) uint64 {
	maxBig := new(big.Int).SetUint64(maxValue)

	n, err := rand.Int(rand.Reader, maxBig)
	if err != nil {
		return 0
	}

	return n.Uint64()
}

func sendRandomEvents(ctx context.Context, client *testutil.TestClient) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	const baseBlock = uint64(18000000)

	blockNumber := baseBlock + randUint64(100000)
	pendingCount := 100

	// Send initial stats
	stats := testutil.CreateTestStats(true, false, false, 25+randInt(10))
	if err := client.SendStats(stats); err != nil {
		log.Printf("Failed to send stats: %v", err)
	}

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// Randomly choose which event to send
			eventType := randInt(4)

			switch eventType {
			case 0:
				// Send block report
				blockNumber++
				block := testutil.CreateTestBlock(blockNumber, fmt.Sprintf("0x%064x", blockNumber-1))

				if err := client.SendBlock(&block); err != nil {
					log.Printf("Failed to send block: %v", err)
				} else {
					log.Printf("Sent block #%d", blockNumber)
				}

			case 1:
				// Send stats update
				peers := 20 + randInt(15)
				syncing := randInt(10) < 1 // 10% chance of syncing
				stats := testutil.CreateTestStats(true, syncing, false, peers)

				if err := client.SendStats(stats); err != nil {
					log.Printf("Failed to send stats: %v", err)
				} else {
					log.Printf("Sent stats update (peers: %d, syncing: %v)", peers, syncing)
				}

			case 2:
				// Send pending transactions update
				delta := randInt(100) - 50
				pendingCount += delta

				if pendingCount < 0 {
					pendingCount = 0
				}

				if err := client.SendPending(pendingCount); err != nil {
					log.Printf("Failed to send pending: %v", err)
				} else {
					log.Printf("Sent pending count: %d", pendingCount)
				}

			case 3:
				// Send latency report
				latency := 10 + randInt(100)

				if err := client.SendLatency(latency); err != nil {
					log.Printf("Failed to send latency: %v", err)
				} else {
					log.Printf("Sent latency: %dms", latency)
				}
			}

			// Occasionally send history
			shouldSendHistory := randInt(10) < 1 // 10% chance

			if shouldSendHistory {
				blocks := []protocol.Block{}

				for i := uint64(0); i < 5; i++ {
					offset := i + 1

					if blockNumber <= offset {
						break
					}

					blockNum := blockNumber - offset
					parentHash := fmt.Sprintf("0x%064x", blockNum-1)
					b := testutil.CreateTestBlock(blockNum, parentHash)
					blocks = append(blocks, b)
				}

				if err := client.SendHistory(blocks); err != nil {
					log.Printf("Failed to send history: %v", err)
				} else {
					log.Printf("Sent history with %d blocks", len(blocks))
				}
			}
		}
	}
}

func handlePings(ctx context.Context, client *testutil.TestClient) {
	// Send periodic pings
	pingTicker := time.NewTicker(30 * time.Second)
	defer pingTicker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-pingTicker.C:
			if err := client.SendNodePing(); err != nil {
				log.Printf("Failed to send ping: %v", err)

				continue
			}

			// Wait for pong
			if _, err := client.WaitForMessage("node-pong", 5*time.Second); err != nil {
				log.Printf("Failed to receive pong: %v", err)
			} else {
				log.Println("Ping-pong successful")
			}
		}
	}
}
