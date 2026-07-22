package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/joho/godotenv"
	"github.com/semptpanick/sempatowo/internal/config"
	"github.com/semptpanick/sempatowo/internal/farm"
	"github.com/semptpanick/sempatowo/internal/util"
)

func logPanic(msg string) { log.Print(msg) }

func main() {
	simulateCaptcha := flag.Bool("simulate-captcha", false, "connect and inject a fake OwO captcha to test pause, browser, and notifications")
	checkConfig := flag.Bool("check-config", false, "validate the environment and every config file, then exit")
	flag.Parse()

	_ = godotenv.Load()

	// The whole environment is read and validated here, before anything
	// connects. A bad CAPTCHA_SERVICE used to surface the first time a captcha
	// appeared, which is exactly when nobody is watching.
	env, err := config.LoadEnv()
	if err != nil {
		log.Fatalf("Environment error:\n%v", err)
	}
	if err := env.EnsureDirs(); err != nil {
		log.Fatalf("%v", err)
	}

	if *checkConfig {
		os.Exit(runCheckConfig(env))
	}

	if *simulateCaptcha {
		if len(env.Tokens) > 1 {
			log.Println("simulate-captcha uses only the first token")
		}
		util.Go(logPanic, "simulate-captcha", func() {
			b := farm.New(env.Tokens[0], env)
			if err := b.RunSimulateCaptcha(); err != nil {
				log.Printf("Bot error: %v", err)
			}
		})
	} else {
		// Each account gets its own recovered goroutine so a panic in one
		// does not take down the others sharing this process.
		for _, token := range env.Tokens {
			util.Go(logPanic, "bot", func() {
				b := farm.New(token, env)
				if err := b.Run(); err != nil {
					log.Printf("Bot error: %v", err)
				}
			})
		}
	}

	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM)
	<-sc
}
