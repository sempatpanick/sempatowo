package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/joho/godotenv"
	"github.com/semptpanick/sempatowo/internal/farm"
)

func main() {
	simulateCaptcha := flag.Bool("simulate-captcha", false, "connect and inject a fake OwO captcha to test pause, browser, and notifications")
	flag.Parse()

	_ = godotenv.Load()

	tokenEnv := strings.TrimSpace(os.Getenv("TOKEN"))
	if tokenEnv == "" {
		log.Fatal("No TOKEN found in .env")
	}

	tokens := parseTokens(tokenEnv)
	if len(tokens) == 0 {
		log.Fatal("No valid TOKEN found in .env")
	}

	if *simulateCaptcha {
		if len(tokens) > 1 {
			log.Println("simulate-captcha uses only the first token")
		}
		go func(token string) {
			b := farm.New(token)
			if err := b.RunSimulateCaptcha(); err != nil {
				log.Printf("Bot error: %v", err)
			}
		}(tokens[0])
	} else {
		for _, token := range tokens {
			go func(token string) {
				b := farm.New(token)
				if err := b.Run(); err != nil {
					log.Printf("Bot error: %v", err)
				}
			}(token)
		}
	}

	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM)
	<-sc
}

func parseTokens(tokenEnv string) []string {
	parts := strings.Split(tokenEnv, ",")
	var tokens []string
	for _, t := range parts {
		t = strings.TrimSpace(t)
		if t != "" {
			tokens = append(tokens, t)
		}
	}
	return tokens
}
