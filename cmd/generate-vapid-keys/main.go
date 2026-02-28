// Command generate-vapid-keys generates a VAPID key pair for Web Push
// authentication. The output can be used to set the VAPID_PUBLIC_KEY,
// VAPID_PRIVATE_KEY, and VAPID_SUBJECT environment variables for production.
//
// Usage:
//
//	go run ./cmd/generate-vapid-keys                 # .env format (default)
//	go run ./cmd/generate-vapid-keys --format=fly    # fly secrets set command
//	go run ./cmd/generate-vapid-keys --format=heroku # heroku config:set command
//	go run ./cmd/generate-vapid-keys --format=json   # JSON for tooling
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"

	wplib "github.com/SherClockHolmes/webpush-go"
)

func main() {
	format := flag.String("format", "env", "output format: env, fly, heroku, json")
	flag.Parse()

	privateKey, publicKey, err := wplib.GenerateVAPIDKeys()
	if err != nil {
		log.Fatalf("Failed to generate VAPID keys: %v", err)
	}

	switch *format {
	case "env":
		fmt.Println("# VAPID key pair for Web Push authentication")
		fmt.Println("# Add these to your .env or environment variables:")
		fmt.Println()
		fmt.Printf("VAPID_PUBLIC_KEY=%s\n", publicKey)
		fmt.Printf("VAPID_PRIVATE_KEY=%s\n", privateKey)
		fmt.Println("VAPID_SUBJECT=mailto:admin@mycompany.com")

	case "fly":
		fmt.Println("# Run this command to set VAPID secrets on Fly.io:")
		fmt.Printf("fly secrets set VAPID_PUBLIC_KEY=%s VAPID_PRIVATE_KEY=%s VAPID_SUBJECT=mailto:admin@mycompany.com\n", publicKey, privateKey)

	case "heroku":
		fmt.Println("# Run this command to set VAPID config on Heroku:")
		fmt.Printf("heroku config:set VAPID_PUBLIC_KEY=%s VAPID_PRIVATE_KEY=%s VAPID_SUBJECT=mailto:admin@mycompany.com\n", publicKey, privateKey)

	case "json":
		out := map[string]string{
			"VAPID_PUBLIC_KEY":  publicKey,
			"VAPID_PRIVATE_KEY": privateKey,
			"VAPID_SUBJECT":     "mailto:admin@mycompany.com",
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(out); err != nil {
			log.Fatalf("Failed to encode JSON: %v", err)
		}

	default:
		fmt.Fprintf(os.Stderr, "unknown format: %s (expected: env, fly, heroku, json)\n", *format)
		os.Exit(1)
	}
}
