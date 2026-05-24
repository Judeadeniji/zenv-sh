# zEnv CLI

Go CLI for [zEnv](../README.md). Single static binary, no Node runtime. Uses [Amnesia](../amnesia/) for client-side encryption — the API only ever sees ciphertext.

## Install

```bash
make build-cli          # from repo root → bin/zenv
./bin/zenv --help

go run ./cli/cmd/zenv -- secrets list   # without installing
```

Set install-wide API URLs once (optional):

```bash
zenv config set --global api_url http://localhost:8080
zenv config set --global auth_url http://localhost:3000
```

Defaults: `http://localhost:8080` (API), `http://localhost:3000` (dashboard).

---

## Two credentials (read this first)

The CLI needs **two separate secrets**. They are not interchangeable.

| Credential | Purpose | Typical storage |
|------------|---------|-----------------|
| **Service token** (`ze_…`) | Authenticates HTTP calls to the API | Per-project credentials file, or `ZENV_TOKEN` in CI |
| **Project vault key** | Derives the project DEK; encrypt/decrypt secret values | Per-project credentials file, `ZENV_PROJECT_KEY` in CI, or `zenv unlock` on a laptop |

The server is a ciphertext warehouse. Neither credential sends plaintext secrets to the API. The **project vault key never leaves your machines** in plaintext except where you put it (env vars, credentials files).

### Service tokens are for machines

Service tokens exist for **unattended** use: CI, bots, cron, `zenv run` in production.

- `zenv login` saves **only the token** — no Vault Key prompt.
- Pipelines should use `ZENV_TOKEN` + `ZENV_PROJECT_KEY` from the platform secret store, not `zenv unlock`.

### `zenv unlock` is for developers

`zenv unlock` prompts for your **Vault Key** (passphrase/PIN), unwraps the project vault key from your key grant, and saves it **for that project only**. Use it once on a trusted laptop, not in CI.

---

## Service token scope

Every service token is bound at creation time to:

- **One project** (`project_id`)
- **One environment** (`development`, `staging`, or `production`)
- **Permission** (`read` or `read_write`)

Token format: `ze_{environment}_{random_hex}` (the prefix hints at the env; the authoritative scope is in the database).

Implications:

- You need **different tokens** for dev vs prod or for different projects.
- `zenv whoami` shows the token’s scope.
- The CLI **refuses** to run secret commands if `.zenv` / flags point at a different project or env than the token (`project mismatch` / `environment mismatch`).

---

## Configuration layout

Three layers work together like git + ssh config:

```
~/.config/zenv/
  config                              # api_url, auth_url (install-wide)
  projects/
    <project-uuid>/
      credentials                     # token + project_key for THAT project (mode 0600)
  credentials                         # legacy global fallback (discouraged)

./.zenv                               # in repo (safe to commit — no secrets)
  project=<uuid>
  env=development
```

### What goes where

| Key | Where | Committable? |
|-----|--------|--------------|
| `api_url`, `auth_url` | `~/.config/zenv/config` | N/A (machine-local) |
| `token`, `project_key` | `~/.config/zenv/projects/<id>/credentials` | **Never** |
| `project`, `env` | `.zenv` (walked up from cwd) | Usually yes |

**Secrets are never read from `.zenv`.** Putting `token=` or `project_key=` in `.zenv` is ignored for loading (and `zenv config set` rejects writing secrets there).

### Resolution order

On every command, config is loaded in this order (highest wins):

**Project / environment**

1. `--project` / `--env` flags  
2. Nearest `.zenv` (walks up directories like git)  
3. `ZENV_PROJECT` / `ZENV_ENV`

**API URLs**

1. `.zenv` → global `config` → `ZENV_API_URL` / `ZENV_AUTH_URL` → defaults  

**Token / project key** (only after `project` is known)

1. `~/.config/zenv/projects/<project-id>/credentials`  
2. Legacy `~/.config/zenv/credentials`  
3. `ZENV_TOKEN` / `ZENV_PROJECT_KEY`

Environment variables always override file values when set.

### Multi-project / multi-org workflow

```bash
# Repo A
cd ~/work/backend-a
zenv projects init 31a4884b-ec44-437d-a7c7-e17752137cfa
zenv login                    # token → projects/31a4884b-.../credentials
zenv unlock                   # project_key → same file

# Repo B
cd ~/work/backend-b
zenv projects init <other-uuid>
zenv login
zenv unlock

zenv config profiles          # list all projects with token/key status
```

Each project has its own credentials directory. Switching repos switches context via `.zenv`; the CLI loads the matching `projects/<id>/credentials` automatically.

### Legacy global credentials

If you still have `~/.config/zenv/credentials`, it is used as a **fallback** when the per-project file has no value. On first use with a resolved project, values may be copied into the per-project file. New setups should skip the global file entirely.

To store one key globally (shared across all projects — **not recommended**):

```bash
zenv config set --global project_key '<key>'   # prints a warning
```

---

## Quick start

### Developer laptop

```bash
cd your-repo
zenv projects init <project-id>    # writes .zenv
zenv login                         # paste token from dashboard
zenv unlock                        # enter Vault Key once
zenv whoami
zenv secrets list
```

### CI / automation

Do **not** run `zenv unlock` in CI. Inject secrets from your platform:

```bash
export ZENV_TOKEN=ze_development_...
export ZENV_PROJECT_KEY=<from dashboard — store as CI secret>
export ZENV_PROJECT=<project-uuid>     # if .zenv is not in the job workspace
export ZENV_ENV=development

zenv check DATABASE_URL API_KEY --env development
zenv run -- npm test
```

---

## Commands

### Auth

| Command | Needs project key? | Description |
|---------|------------------|-------------|
| `zenv login` | No | Save service token for the token’s project |
| `zenv unlock` | No (creates it) | Derive project key from Vault Key; save per project |
| `zenv whoami` | No | Token name, user, org, project, env, permission, key status |

### Secrets

Requires full config: token, project key, `project`, `env`.

```bash
zenv secrets set KEY VALUE
zenv secrets get KEY
zenv secrets list
zenv secrets delete KEY
```

### Run

```bash
zenv run -- node server.js
zenv run -- python app.py
```

Fetches all secrets for the current project+env, decrypts locally, appends `NAME=value` to the environment, then **`exec`s** the command (replaces the `zenv` process). Use `--` before the command.

### Check (CI)

```bash
zenv check DATABASE_URL STRIPE_KEY --env production
```

Verifies secrets **exist** (by name hash). Does not decrypt or print values. Exits `1` if any name is missing.

### Environment files

```bash
zenv env pull                      # writes .env.local (replaces entire file)
zenv env pull -o .env.production
zenv env pull -o -                 # stdout
zenv env diff development staging  # compare envs
```

### Projects

```bash
zenv projects init <project-id>    # pin cwd to project (.zenv)
zenv projects list --org <org-id>
zenv projects get <project-id>
zenv projects create --name … --public-key …   # advanced; needs vault material
```

### Service tokens (API)

```bash
zenv tokens create --name ci-bot --permission read_write
zenv tokens list
zenv tokens revoke <token-id>
```

`tokens create` uses the **current** `project` and `env` from config. Requires an existing token with sufficient access.

### Config

```bash
zenv config set project <uuid>           # .zenv
zenv config set env production           # .zenv
zenv config set token ze_…               # current project’s credentials
zenv config set --global api_url http://…

zenv config get project
zenv config list                         # global + .zenv + active project creds
zenv config profiles                     # all projects with saved credentials
zenv config path                         # ~/.config/zenv
```

Use `--project <uuid>` with `set`/`get`/`unset` to target another project’s credentials without changing `.zenv`.

### Organizations

```bash
zenv orgs list
zenv orgs get <org-id>
zenv orgs members <org-id>
# …
```

Org commands need a token only (no project key).

---

## Quirks and behavior

### `login` always targets the token’s project

`zenv login` calls the API to learn the token’s `project_id`, saves the token under `projects/<id>/credentials`, and updates **`.zenv` in the current directory** with that project (and env from the token). If you wanted a different project in `.zenv`, run `zenv projects init` afterward.

### `unlock` does not use the global credentials file

Project keys are saved to `~/.config/zenv/projects/<id>/credentials`, not `~/.config/zenv/credentials`. You’ll see the exact path printed after unlock.

### Mismatch errors are intentional

If `.zenv` says project A but your token is for project B, secret commands fail with a clear error. Fix with `zenv projects init <token-project-id>` or use the correct token for this repo.

### Commands that need what

| Needs token only | Needs token + project key + project + env |
|------------------|-------------------------------------------|
| `whoami`, `login`, `orgs …`, `projects list/get` | `secrets`, `run`, `env pull`, `check`, `tokens create` |

### `zenv run` injection count

The stderr line `injected N secrets` reflects bulk-fetched items; decrypt failures are warned but skipped (command still runs with partial env).

### Crypto is always client-side

```
project_key + project_salt (from API)
  → Argon2id → Project KEK
  → unwrap wrapped_project_dek → Project DEK
  → encrypt/decrypt secrets + HMAC secret names
```

Wrong `project_key` fails at unwrap with a message mentioning `ZENV_PROJECT_KEY`.

### `.zenv` discovery

The CLI walks from **current working directory** upward until it finds `.zenv`. Running `zenv` from a subdirectory of a repo still picks up the repo root `.zenv`.

### Flags override everything

```bash
zenv secrets list --project <uuid> --env staging
```

Useful for one-off ops without editing `.zenv`.

### Debug logging

```bash
zenv -v secrets list
```

---

## Troubleshooting

| Symptom | Likely cause | Fix |
|---------|--------------|-----|
| `not authenticated` | No token for this project | `zenv login` or `ZENV_TOKEN` |
| `project key not set` | No key for resolved project | `zenv unlock` or `ZENV_PROJECT_KEY` |
| `no project specified` | No `.zenv` / flag / env | `zenv projects init <id>` |
| `project mismatch` | `.zenv` project ≠ token’s project | `zenv projects init <id>` or re-login with correct token |
| `environment mismatch` | `.zenv` env ≠ token’s env | `zenv config set env <token-env>` |
| `unwrap project DEK (wrong ZENV_PROJECT_KEY?)` | Wrong vault key for this project | Correct key from dashboard; re-run `unlock` |
| `vault not set up` / vault material 404 | User has no identity row | Complete vault setup in dashboard |
| Token works in `whoami` but not secrets | Missing project key | `unlock` or set `ZENV_PROJECT_KEY` |

---

## Security notes

- Credential files are written with mode `0600`.
- Do not commit `~/.config/zenv/projects/` or legacy `credentials`.
- `.zenv` should only contain `project` and `env` (and optionally `api_url` / `auth_url` overrides).
- Service tokens are hashed server-side; plaintext is shown once at creation.
- Prefer per-project credential files over `--global` for secrets.

---

## License

MIT — see [LICENSE](LICENSE).
