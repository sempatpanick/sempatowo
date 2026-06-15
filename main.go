package main

import (
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/joho/godotenv"
	"github.com/semptpanick/sempatowo/internal/farm"
)

func main() {
	_ = godotenv.Load()

	tokenEnv := strings.TrimSpace(os.Getenv("TOKEN"))
	if tokenEnv == "" {
		log.Fatal("No TOKEN found in .env")
	}

	tokens := strings.Split(tokenEnv, ",")
	for _, t := range tokens {
		t = strings.TrimSpace(t)
		if t == "" {
			continue
		}
		go func(token string) {
			b := farm.New(token)
			if err := b.Run(); err != nil {
				log.Printf("Bot error: %v", err)
			}
		}(t)
	}

	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM)
	<-sc
}
