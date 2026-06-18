package main

import (
	"log"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/hytams/discordgo-self"
	"github.com/hytams/discordgo-self/types"
)

func main() {
	token := os.Getenv("DISCORD_TOKEN")
	if token == "" {
		log.Fatal("DISCORD_TOKEN environment variable is required")
	}

	guildIDStr := os.Getenv("GUILD_ID")
	channelIDStr := os.Getenv("CHANNEL_ID")

	if guildIDStr == "" || channelIDStr == "" {
		log.Fatal("GUILD_ID and CHANNEL_ID environment variables are required")
	}

	// Parse IDs
	gID, _ := strconv.ParseUint(guildIDStr, 10, 64)
	cID, _ := strconv.ParseUint(channelIDStr, 10, 64)
	guildID := types.Snowflake(gID)
	channelID := types.Snowflake(cID)

	// Configure client
	client, err := discordgo.New(token,
		discordgo.WithDebug(true),
		discordgo.WithConfig(&discordgo.Config{
			StateEnabled: true,
		}),
	)
	if err != nil {
		log.Fatal(err)
	}

	// Add READY handler
	client.OnReady(func() {
		log.Printf("Logged in as %s#%s", client.User.Username, client.User.Discriminator)

		// Wait a bit to ensure everything is ready
		time.Sleep(2 * time.Second)

		// Join Voice Channel
		log.Printf("Joining Voice Channel: %s (Guild: %s)", channelID, guildID)
		vc, err := client.JoinVoiceChannel(guildID, channelID, false, false)
		if err != nil {
			log.Printf("Failed to join voice channel: %v", err)
			return
		}

		log.Println("Successfully joined voice channel!")

		// Start speaking loop (Silence frames to keep connection alive)
		go func() {
			// Opus silence frame (approx 20ms)
			silence := []byte{0xF8, 0xFF, 0xFE}

			// Set speaking status
			if err := vc.Speaking(true); err != nil {
				log.Printf("Failed to set speaking: %v", err)
			}

			ticker := time.NewTicker(20 * time.Millisecond)
			defer ticker.Stop()

			for {
				select {
				case <-ticker.C:
					if err := vc.SendOpus(silence); err != nil {
						log.Printf("Error sending opus: %v", err)
						return
					}
				}
			}
		}()
	})

	// Connect
	if err := client.Open(); err != nil {
		log.Fatal(err)
	}

	// Keep running until interrupt
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	log.Println("Shutting down...")
	client.Close()
}
