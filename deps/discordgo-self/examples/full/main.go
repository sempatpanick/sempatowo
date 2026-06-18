package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/skip2/go-qrcode"
	discordgo "github.com/hytams/discordgo-self"
	"github.com/hytams/discordgo-self/manager"
)

const tokenFile = "token.json"

type TokenData struct {
	Token    string `json:"token"`
	UserID   string `json:"user_id,omitempty"`
	Username string `json:"username,omitempty"`
	SavedAt  string `json:"saved_at"`
}

func main() {
	fmt.Println("==============================================")
	fmt.Println("  Discord Selfbot - Full Example")
	fmt.Println("==============================================")

	token := loadToken()

	if token == "" {
		fmt.Println("\n[!] No saved token found. Starting QR Login...")
		token = doQRLogin()
		if token == "" {
			log.Fatal("Failed to get token from QR login")
		}
	} else {
		fmt.Println("[+] Loaded saved token from", tokenFile)
	}

	startBot(token)
}

func loadToken() string {
	data, err := os.ReadFile(tokenFile)
	if err != nil {
		return ""
	}

	var tokenData TokenData
	if err := json.Unmarshal(data, &tokenData); err != nil {
		return ""
	}

	return tokenData.Token
}

func saveToken(token, userID, username string) error {
	tokenData := TokenData{
		Token:    token,
		UserID:   userID,
		Username: username,
		SavedAt:  time.Now().Format(time.RFC3339),
	}

	data, err := json.MarshalIndent(tokenData, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(tokenFile, data, 0600)
}

func doQRLogin() string {
	qrClient, err := manager.NewRemoteAuthClientWithPlatform(manager.IOSProps())
	if err != nil {
		log.Printf("Failed to create QR client: %v", err)
		return ""
	}

	tokenChan := make(chan string, 1)

	qrClient.OnFingerprint = func(url string) {
		fmt.Println("\n[QR Code] Scan this with Discord Mobile:")
		fmt.Println("")

		qr, err := qrcode.New(url, qrcode.Medium)
		if err != nil {
			fmt.Println("URL:", url)
		} else {
			fmt.Println(qr.ToString(false))
		}
		fmt.Println("Waiting for scan...")
	}

	qrClient.OnToken = func(token string) {
		fmt.Println("\n[+] Login successful! Token received.")
		tokenChan <- token
	}

	qrClient.OnCaptcha = func(captcha *manager.CaptchaInfo) string {
		fmt.Println("\n[!] Captcha required!")
		fmt.Printf("[!] Service: %s, Sitekey: %s\n", captcha.Service, captcha.Sitekey)
		fmt.Println("[!] Implement your own captcha solving logic here")
		return ""
	}

	if err := qrClient.Start(); err != nil {
		log.Printf("Failed to start QR login: %v", err)
		return ""
	}

	select {
	case token := <-tokenChan:
		return token
	case <-time.After(5 * time.Minute):
		log.Println("QR login timeout")
		qrClient.Close()
		return ""
	}
}

func startBot(token string) {
	fmt.Println("\n[*] Connecting to Discord...")

	client, err := discordgo.New(token, discordgo.WithDebug(false))
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}

	client.OnReady(func() {
		fmt.Println("==============================================")
		fmt.Printf("  Logged in as: %s#%s\n", client.User.Username, client.User.Discriminator)
		fmt.Printf("  User ID: %s\n", client.User.ID)
		fmt.Printf("  Latency: %v\n", client.Latency())
		fmt.Println("==============================================")

		saveToken(token, client.User.ID.String(), client.User.Username)
		fmt.Println("[+] Token saved to", tokenFile)
	})

	client.OnMessageCreate(func(m *discordgo.Message) {
		if client.User != nil && m.Author.ID == client.User.ID {
			return
		}

		fmt.Printf("[#%s] %s: %s\n", m.ChannelID, m.Author.Username, m.Content)

		switch {
		case strings.HasPrefix(m.Content, "!ping"):
			client.SendMessage(m.ChannelID, "Pong! 🏓")

		case strings.HasPrefix(m.Content, "!info"):
			msg := fmt.Sprintf("```\nSelfbot Status:\nUser: %s\nLatency: %v\n```",
				client.User.Username, client.Latency())
			client.SendMessage(m.ChannelID, msg)

		case strings.HasPrefix(m.Content, "!echo "):
			content := strings.TrimPrefix(m.Content, "!echo ")
			client.SendMessage(m.ChannelID, content)
		}
	})

	if err := client.Open(); err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer client.Close()

	if err := client.WaitUntilReady(30 * time.Second); err != nil {
		log.Fatalf("Failed to wait for ready: %v", err)
	}

	fmt.Println("\n[Bot Running] Press Ctrl+C to exit.")
	fmt.Println("Commands: !ping, !info, !echo <text>")

	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc

	fmt.Println("\n[*] Shutting down...")
}
