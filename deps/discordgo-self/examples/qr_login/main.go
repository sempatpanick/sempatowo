package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"

	"github.com/skip2/go-qrcode"
	"github.com/hytams/discordgo-self/manager"
)

func main() {
	qrClient, err := manager.NewRemoteAuthClientWithPlatform(manager.IOSProps())

	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}

	qrClient.OnFingerprint = func(url string) {
		fmt.Println("=====================================================")
		fmt.Println("QR Code Login")
		fmt.Println("Scan this with your Discord Mobile App:")
		fmt.Println("")

		qr, err := qrcode.New(url, qrcode.Medium)
		if err != nil {
			fmt.Printf("Could not generate QR code: %v\n", err)
			fmt.Println("URL:", url)
		} else {
			fmt.Println(qr.ToString(false))
		}

		fmt.Println("=====================================================")
	}

	qrClient.OnUserData = func(user *manager.RemoteUser) {
		fmt.Printf("\n[+] Scanned by: %s (%s)\n", user.Username, user.Discriminator)
		fmt.Println("Please confirm the login on your mobile device...")
	}

	qrClient.OnToken = func(token string) {
		fmt.Println("\n[SUCCESS] Login Complete!")
		fmt.Printf("Token: %s\n", token)
		os.Exit(0)
	}

	if err := qrClient.Start(); err != nil {
		log.Fatalf("Error starting: %v", err)
	}

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt)
	<-stop
}
