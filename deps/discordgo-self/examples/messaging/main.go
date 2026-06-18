package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	discordgo "github.com/hytams/discordgo-self"
)

func main() {
	token := os.Getenv("DISCORD_TOKEN")
	if token == "" {
		log.Fatal("DISCORD_TOKEN environment variable is required")
	}

	client, err := discordgo.New(token)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}

	client.OnReady(func() {
		fmt.Printf("Logged in as %s\n", client.User.Username)
	})

	if err := client.Open(); err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer client.Close()

	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc
}

var _ = demonstrateMessaging

func demonstrateMessaging(client *discordgo.Client, channelID discordgo.Snowflake) {
	msg, err := client.SendMessage(channelID, "Hello! This is a simple message.")
	if err != nil {
		log.Printf("Failed to send simple message: %v", err)
		return
	}
	fmt.Printf("Sent message: %s\n", msg.ID)

	time.Sleep(2 * time.Second)

	editedMsg, err := client.EditMessage(channelID, msg.ID, "Hello! This message was edited.")
	if err != nil {
		log.Printf("Failed to edit message: %v", err)
	} else {
		fmt.Printf("Edited message: %s\n", editedMsg.ID)
	}

	time.Sleep(2 * time.Second)

	embed := &discordgo.Embed{
		Title:       "📊 Status Report",
		Description: "Everything is working great!",
		Color:       0x00FF00,
		Fields: []*discordgo.EmbedField{
			{
				Name:   "Latency",
				Value:  fmt.Sprintf("%dms", client.Latency().Milliseconds()),
				Inline: true,
			},
			{
				Name:   "Status",
				Value:  "✅ Online",
				Inline: true,
			},
		},
	}

	embedMsg, err := client.SendEmbed(channelID, embed)
	if err != nil {
		log.Printf("Failed to send embed: %v", err)
	} else {
		fmt.Printf("Sent embed: %s\n", embedMsg.ID)
	}

	time.Sleep(2 * time.Second)

	replyMsg, err := client.SendReply(channelID, msg.ID, "This is a reply to the first message!")
	if err != nil {
		log.Printf("Failed to send reply: %v", err)
	} else {
		fmt.Printf("Sent reply: %s\n", replyMsg.ID)
	}

	time.Sleep(2 * time.Second)

	emojis := []string{"👍", "❤️", "🔥"}
	for _, emoji := range emojis {
		if err := client.AddReaction(channelID, msg.ID, emoji); err != nil {
			log.Printf("Failed to add reaction %s: %v", emoji, err)
		} else {
			fmt.Printf("Added reaction: %s\n", emoji)
		}
		time.Sleep(500 * time.Millisecond)
	}

	time.Sleep(2 * time.Second)

	if err := client.TriggerTyping(channelID); err != nil {
		log.Printf("Failed to trigger typing: %v", err)
	} else {
		fmt.Println("Triggered typing indicator")
	}

	messages, err := client.GetMessages(channelID, 10)
	if err != nil {
		log.Printf("Failed to get messages: %v", err)
	} else {
		fmt.Printf("Got %d messages from channel\n", len(messages))
		for _, m := range messages {
			fmt.Printf("  - %s: %s\n", m.Author.Username, truncate(m.Content, 50))
		}
	}

	if err := client.DeleteMessage(channelID, msg.ID); err != nil {
		log.Printf("Failed to delete message: %v", err)
	} else {
		fmt.Println("Deleted first message")
	}

	textContent := []byte("Hello, this is a text file content!")
	fileMsg, err := client.SendFileFromBytes(channelID, "hello.txt", textContent, "Here's a text file!")
	if err != nil {
		log.Printf("Failed to send file: %v", err)
	} else {
		fmt.Printf("Sent file: %s\n", fileMsg.ID)
	}
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
