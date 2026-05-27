# zEnv Go SDK

The official Go SDK for [zEnv](https://github.com/Judeadeniji/zenv-sh), the end-to-end encrypted secrets manager.

Use this SDK to securely fetch and decrypt secrets in your Go applications dynamically at runtime. Because zEnv is zero-knowledge, the SDK handles the decryption of your project's Data Encryption Key (DEK) and decrypts the secrets locally in memory.

## Installation

```bash
go get github.com/Judeadeniji/zenv-sh/sdk-go
```

## Quickstart

By default, the SDK automatically loads credentials from your environment variables:
- `ZENV_TOKEN`: Your service token.
- `ZENV_PROJECT`: Your project UUID.
- `ZENV_PROJECT_KEY`: The zero-knowledge vault key used to decrypt secrets.
- `ZENV_API_URL`: (Optional) The API URL if self-hosting.

```go
package main

import (
	"fmt"
	"log"

	"github.com/Judeadeniji/zenv-sh/sdk-go/zenv"
)

func main() {
	// Automatically uses ZENV_TOKEN, ZENV_PROJECT, and ZENV_PROJECT_KEY
	client, err := zenv.NewClient()
	if err != nil {
		log.Fatalf("Failed to initialize zEnv client: %v", err)
	}

	// Fetch a specific secret from the "production" environment
	dbPassword, err := client.FetchSecret("production", "DB_PASSWORD")
	if err != nil {
		log.Fatalf("Failed to fetch DB_PASSWORD: %v", err)
	}

	fmt.Printf("Successfully retrieved DB_PASSWORD (length: %d)\n", len(dbPassword))
}
```

## Advanced Configuration

You can also provide configuration explicitly using Functional Options instead of environment variables:

```go
client, err := zenv.NewClient(
    zenv.WithToken("ze_1234567890"),
    zenv.WithProjectID("12345678-1234-1234-1234-123456789012"),
    zenv.WithVaultKey("your-vault-key"),
)
```

## API Reference

### `FetchSecret(env, name string) (string, error)`
Fetches and decrypts a single secret by its name. Returns an error if the secret does not exist.

### `FetchAllSecrets(env string) (map[string]string, error)`
Fetches and decrypts all secrets for the given environment in a single API call.
