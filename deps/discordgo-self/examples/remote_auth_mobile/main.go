// Package main demonstrates using Remote Auth Mobile feature.
// This allows a logged-in mobile device to approve/deny QR login requests.
package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/hytams/discordgo-self/api"
)

func main() {
	token := os.Getenv("DISCORD_TOKEN")
	if token == "" {
		log.Fatal("DISCORD_TOKEN environment variable is required")
	}

	// Create API client
	client, err := api.NewClient(api.ClientConfig{
		Token: token,
	})
	if err != nil {
		log.Fatal(err)
	}

	reader := bufio.NewReader(os.Stdin)

	fmt.Println("Remote Auth Mobile Example")
	fmt.Println("==========================")
	fmt.Println()
	fmt.Println("This example demonstrates how to approve/deny QR login requests")
	fmt.Println("from other devices using your already logged-in account.")
	fmt.Println()

	// Get QR code URL from user
	fmt.Print("Enter the QR code URL (https://discord.com/ra/...): ")
	qrURL, _ := reader.ReadString('\n')
	qrURL = strings.TrimSpace(qrURL)

	// Extract fingerprint from URL
	fingerprint, err := api.ExtractRemoteAuthFingerprint(qrURL)
	if err != nil {
		log.Fatalf("Failed to extract fingerprint: %v", err)
	}

	fmt.Printf("Extracted fingerprint: %s\n", fingerprint[:20]+"...")
	fmt.Println()

	// Create remote auth session
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	fmt.Println("Creating remote auth session...")
	session, err := client.CreateRemoteAuthSession(ctx, fingerprint)
	if err != nil {
		log.Fatalf("Failed to create session: %v", err)
	}

	fmt.Println("Session created successfully!")
	fmt.Printf("Handshake token: %s...\n", session.HandshakeToken[:50])
	fmt.Println()

	// Prompt user to approve or deny
	fmt.Print("Approve this login request? (y/n): ")
	response, _ := reader.ReadString('\n')
	response = strings.TrimSpace(strings.ToLower(response))

	if response == "y" || response == "yes" {
		// Approve the login
		fmt.Println("Approving login request...")
		err = client.FinishRemoteAuth(ctx, session.HandshakeToken, false)
		if err != nil {
			log.Fatalf("Failed to finish auth: %v", err)
		}
		fmt.Println("Login approved! The other device should now be logged in.")
	} else {
		// Deny the login
		fmt.Println("Denying login request...")
		err = client.CancelRemoteAuth(ctx, session.HandshakeToken)
		if err != nil {
			log.Fatalf("Failed to cancel auth: %v", err)
		}
		fmt.Println("Login denied. The other device will not be logged in.")
	}
}
