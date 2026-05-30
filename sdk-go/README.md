# zEnv Go SDK

[![Go](https://img.shields.io/badge/Go-1.24+-00ADD8?style=flat&logo=go)](https://golang.org)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](../LICENSE-MIT)

The official Go SDK for **zEnv** — the zero-knowledge secret manager.

This SDK allows you to fetch and decrypt secrets in your Go applications dynamically at runtime. Because zEnv uses a zero-knowledge architecture, the server only ever returns encrypted blobs. This SDK automatically handles key derivation and decryption entirely in-memory on your machine.

## Install

```bash
go get github.com/Judeadeniji/zenv-sh/sdk-go@latest
```

## How It Works

The SDK uses the [Amnesia crypto engine](../amnesia/) under the hood:

1. **Authentication:** Authenticates to the zEnv API using your `ZENV_TOKEN`.
2. **Key Derivation:** Uses your `ZENV_PROJECT_KEY` (Vault Key) and the server's Project Salt to derive your Project Data Encryption Key (DEK). This happens purely client-side.
3. **Fetching & Decrypting:** Fetches the encrypted secrets from the server and decrypts them locally using AES-256-GCM. The server never sees the plaintext.

## Quickstart

The easiest way to use the SDK is to rely on environment variables for configuration.

**Environment Variables:**
- `ZENV_TOKEN`: Your service token.
- `ZENV_PROJECT`: Your project UUID.
- `ZENV_PROJECT_KEY`: Your zero-knowledge vault key.
- `ZENV_ENV`: The environment to pull secrets from (e.g., `development`, `production`).
- `ZENV_API_URL`: (Optional) The API URL, defaults to `https://api.zenv.sh`.

```go
package main

import (
	"fmt"
	"log"

	"github.com/Judeadeniji/zenv-sh/sdk-go/zenv"
)

func main() {
	// Automatically loads from ZENV_TOKEN, ZENV_PROJECT, and ZENV_PROJECT_KEY
	client, err := zenv.NewClient()
	if err != nil {
		log.Fatalf("Failed to initialize zEnv client: %v", err)
	}

	// Fetch a specific secret
	dbPassword, err := client.FetchSecret("production", "DB_PASSWORD")
	if err != nil {
		log.Fatalf("Failed to fetch DB_PASSWORD: %v", err)
	}
	fmt.Printf("DB_PASSWORD: %s\n", dbPassword)

	// Fetch all secrets
	secrets, err := client.FetchAllSecrets("production")
	if err != nil {
		log.Fatalf("Failed to fetch secrets: %v", err)
	}
	fmt.Printf("Loaded %d secrets\n", len(secrets))
}
```

## Explicit Configuration

You can configure the client programmatically using functional options if you prefer not to use environment variables:

```go
client, err := zenv.NewClient(
	zenv.WithAPIURL("https://api.zenv.sh"),
	zenv.WithToken("ze_production_..."),
	zenv.WithProjectID("12345678-1234-1234-1234-123456789012"),
	zenv.WithVaultKey("your-vault-key"),
)
```

## Core API Client

If you need to interact with the raw zEnv HTTP API (e.g., to manage tokens, projects, or list organizations) without the higher-level crypto wrapper, you can use the base `client` package:

```go
package main

import (
	"fmt"
	"log"

	"github.com/Judeadeniji/zenv-sh/sdk-go/client"
)

func main() {
	apiClient := client.New("https://api.zenv.sh", "your-service-token")
	
	info, err := apiClient.Whoami()
	if err != nil {
		log.Fatal(err)
	}
	
	fmt.Printf("Logged in as: %s in environment: %s\n", info.Email, info.Environment)
}
```

## Security Guarantees

- **No Plaintext Transmitted:** Secrets are decrypted in memory, not on the wire.
- **No Credentials Leaked:** The Vault Key (`ZENV_PROJECT_KEY`) is never sent to the zEnv API. It is only used locally to derive the DEK.
- **Purity:** The crypto engine is strictly isolated and deterministic.

## License

MIT — see [LICENSE](../LICENSE-MIT).
