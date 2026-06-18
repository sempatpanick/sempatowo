// Package main demonstrates using Remote Auth to transfer login to another device.
// This example shows how to use an existing token to approve QR login requests,
// effectively "cloning" your login to another device.
//
// Use Cases:
// - Transfer login to a new device without entering password
// - Automate multi-device login
// - Clone session to another browser/client
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

	// Create API client with existing token
	client, err := api.NewClient(api.ClientConfig{
		Token: token,
	})
	if err != nil {
		log.Fatal(err)
	}

	reader := bufio.NewReader(os.Stdin)

	fmt.Println("========================================")
	fmt.Println("  Remote Auth - Login Transfer Tool")
	fmt.Println("========================================")
	fmt.Println()
	fmt.Println("This tool allows you to transfer your Discord login")
	fmt.Println("to another device by approving its QR code scan.")
	fmt.Println()
	fmt.Println("Steps:")
	fmt.Println("1. Open Discord on the target device (desktop/web)")
	fmt.Println("2. Go to Login and select 'Scan QR Code'")
	fmt.Println("3. Copy the QR code URL or scan it")
	fmt.Println("4. Paste the URL below")
	fmt.Println()

	// Get current user info first
	ctx := context.Background()
	user, err := client.GetCurrentUser(ctx)
	if err != nil {
		log.Fatalf("Failed to get current user: %v", err)
	}

	fmt.Printf("Logged in as: %s#%s (%s)\n", user.Username, user.Discriminator, user.ID)
	fmt.Println()

	// Get QR code URL from user
	fmt.Print("Enter QR code URL: ")
	qrURL, _ := reader.ReadString('\n')
	qrURL = strings.TrimSpace(qrURL)

	if qrURL == "" {
		log.Fatal("QR code URL is required")
	}

	// Extract fingerprint from URL
	fingerprint, err := api.ExtractRemoteAuthFingerprint(qrURL)
	if err != nil {
		log.Fatalf("Invalid QR code URL: %v", err)
	}

	fmt.Printf("\nFingerprint extracted: %s...\n", truncate(fingerprint, 30))

	// Create timeout context
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Create remote auth session
	fmt.Println("\nCreating authentication session...")
	session, err := client.CreateRemoteAuthSession(ctx, fingerprint)
	if err != nil {
		log.Fatalf("Failed to create session: %v", err)
	}

	fmt.Println("Session created successfully!")
	fmt.Println()

	// Confirm with user
	fmt.Printf("You are about to transfer login for account:\n")
	fmt.Printf("  Username: %s\n", user.Username)
	fmt.Printf("  User ID:  %s\n", user.ID)
	fmt.Println()
	fmt.Print("Are you sure you want to approve this login? (yes/no): ")

	response, _ := reader.ReadString('\n')
	response = strings.TrimSpace(strings.ToLower(response))

	if response != "yes" && response != "y" {
		// Cancel the session
		fmt.Println("\nCancelling authentication...")
		err = client.CancelRemoteAuth(ctx, session.HandshakeToken)
		if err != nil {
			log.Printf("Warning: Failed to cancel session: %v", err)
		}
		fmt.Println("Login request cancelled.")
		return
	}

	// Approve the login
	fmt.Println("\nApproving login request...")
	err = client.FinishRemoteAuth(ctx, session.HandshakeToken, false)
	if err != nil {
		log.Fatalf("Failed to approve login: %v", err)
	}

	fmt.Println()
	fmt.Println("========================================")
	fmt.Println("  Login Approved Successfully!")
	fmt.Println("========================================")
	fmt.Println()
	fmt.Println("The target device should now be logged in.")
	fmt.Println("Check the device to confirm.")
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
