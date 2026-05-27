package main

import (
	"fmt"
	"log"
	"os"

	"github.com/Judeadeniji/zenv-sh/sdk-go/zenv"
)

func main() {
	// For the sake of this example, we'll set the environment variables here.
	// Normally, you would export these in your shell or CI environment.
	// 
	// export ZENV_TOKEN="ze_..."
	// export ZENV_PROJECT="12345678-..."
	// export ZENV_PROJECT_KEY="correct-horse-battery-staple"
	//
	if os.Getenv("ZENV_TOKEN") == "" {
		log.Println("Note: This example requires ZENV_TOKEN, ZENV_PROJECT, and ZENV_PROJECT_KEY to be set.")
		return
	}

	// Initialize the client. This will authenticate with the API and derive
	// the cryptographic keys needed to decrypt secrets locally.
	client, err := zenv.NewClient()
	if err != nil {
		log.Fatalf("Failed to initialize zEnv client: %v", err)
	}

	envName := "development"
	secretName := "API_KEY"

	// Fetch a specific secret
	val, err := client.FetchSecret(envName, secretName)
	if err != nil {
		log.Printf("Warning: Failed to fetch %s: %v\n", secretName, err)
	} else {
		fmt.Printf("Successfully retrieved %s = %s\n", secretName, val)
	}

	// Fetch all secrets for an environment
	allSecrets, err := client.FetchAllSecrets(envName)
	if err != nil {
		log.Fatalf("Failed to fetch all secrets: %v", err)
	}

	fmt.Printf("\nFound %d secrets in '%s' environment:\n", len(allSecrets), envName)
	for k, v := range allSecrets {
		fmt.Printf(" - %s = %s\n", k, v)
	}
}
