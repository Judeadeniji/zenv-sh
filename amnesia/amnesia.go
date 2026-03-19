// Package amnesia is a pure cryptographic primitive library for zEnv.
//
// Amnesia knows about keys, nonces, plaintexts, and ciphertexts.
// It knows nothing about API calls, databases, users, tokens, projects, or environments.
//
// All encryption and decryption in zEnv flows through this package.
// The server never participates in decryption — Amnesia runs exclusively on the client.
package amnesia
