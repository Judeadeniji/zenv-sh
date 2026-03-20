# Security Policy

## Reporting a Vulnerability

If you discover a security vulnerability in zEnv, please report it responsibly.

**Do NOT open a public GitHub issue for security vulnerabilities.**

Instead, email: **security@zenv.sh**

Include:
- Description of the vulnerability
- Steps to reproduce
- Impact assessment
- Suggested fix (if any)

## Response Timeline

- **Acknowledgment**: within 48 hours
- **Initial assessment**: within 5 business days
- **Fix timeline**: depends on severity, typically within 30 days

## Scope

The following are in scope:
- Amnesia crypto engine (Go + TypeScript)
- API server
- Auth server
- CLI
- SDK
- Dashboard

## Zero-Knowledge Invariant

zEnv's core security property is that the server never has access to plaintext secrets. Any vulnerability that breaks this invariant is treated as critical severity.

The following should never leave the client:
- Vault Key
- KEK (Key Encryption Key)
- DEK (Data Encryption Key)
- Plaintext secrets

## Recognition

We will credit reporters in our security advisories (unless you prefer to remain anonymous).
