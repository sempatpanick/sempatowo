// Command gen-config-schema regenerates internal/config/config.schema.json from
// the struct definitions and doc comments in package config.
//
// It is wired to the //go:generate line at the top of internal/config/config.go,
// so it runs from that directory:
//
//	go generate ./internal/config
package main

import (
	"fmt"
	"os"

	"github.com/semptpanick/sempatowo/internal/config/schemagen"
)

func main() {
	data, err := schemagen.Generate(".")
	if err != nil {
		fmt.Fprintf(os.Stderr, "gen-config-schema: %v\n", err)
		os.Exit(1)
	}
	if err := os.WriteFile("config.schema.json", data, 0o644); err != nil {
		fmt.Fprintf(os.Stderr, "gen-config-schema: %v\n", err)
		os.Exit(1)
	}
}
