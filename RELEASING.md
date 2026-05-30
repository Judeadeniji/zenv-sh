# Release Guide

zEnv is composed of several moving parts: the Go CLI/API, the core Go cryptographic modules, and the TypeScript packages. Releases are highly automated via GitHub Actions.

> **⚠️ Pre-Alpha Notice:** zEnv is currently in pre-alpha. All releases **must** use pre-release versioning until the project is declared stable.

---

## Versioning Scheme

We follow [Semantic Versioning 2.0](https://semver.org/) with **pre-release identifiers**.

### Pre-Alpha Format

```
v0.0.<patch>-alpha.<build>
```

| Segment | Meaning | Example |
|---------|---------|---------|
| `0.0.x` | Pre-alpha baseline — no stability guarantees | `0.0.1` |
| `-alpha.y` | Sequential build within a patch | `-alpha.1`, `-alpha.2` |

**Examples:**
- `v0.0.1-alpha.1` → First pre-alpha build
- `v0.0.1-alpha.2` → Second build (bug fix or iteration)
- `v0.0.2-alpha.1` → Next patch cycle begins

### Rules

1. **Never** use bare versions like `v1.0.0` or `v0.1.0` during pre-alpha.
2. Increment the **build number** (`-alpha.y`) for small changes and fixes.
3. Increment the **patch** (`0.0.x`) when starting a new batch of changes.
4. npm will **not** install pre-release versions by default — users must explicitly opt in:
   ```bash
   npm install @zenv-sh/cli@0.0.1-alpha.1
   ```

### Graduating from Pre-Alpha

When the project is ready:
- **Alpha → Beta:** Switch to `v0.1.0-beta.1`
- **Beta → RC:** Switch to `v0.1.0-rc.1`
- **Stable:** Release `v0.1.0`

---

## 1. Releasing the CLI and API Binaries

Changes to `cli/` or `api/` are released by pushing a version tag. The `.github/workflows/release.yml` workflow will:

1. Cross-compile binaries for `linux` and `darwin` (amd64 + arm64)
2. Create a GitHub Release with the binaries attached
3. Publish platform-specific npm packages (`@zenv-sh/cli-<os>-<cpu>`)
4. Publish the npm wrapper package (`@zenv-sh/cli`)

### Steps

```bash
# 1. Ensure you are on main with all changes pushed
git checkout main
git pull origin main

# 2. Create a pre-alpha tag
git tag v0.0.1-alpha.1

# 3. Push the tag to trigger the release
git push origin v0.0.1-alpha.1
```

That's it. GitHub Actions handles the rest.

### NPM Authentication

The release workflow authenticates with npm using the `NPM_TOKEN` repository secret. This token must be configured in:

**Settings → Secrets and variables → Actions → `NPM_TOKEN`**

Generate a token at [npmjs.com → Access Tokens](https://www.npmjs.com/settings/~/tokens) with publish access to the `@zenv-sh` scope.

### If a Release Fails Mid-Way

If the workflow fails after creating the GitHub Release (e.g., npm publish errors), delete the release and re-push the tag:

```bash
# Delete the failed release and remote tag
gh release delete v0.0.1-alpha.1 --yes --cleanup-tag

# Delete local tag, recreate, and push
git tag -d v0.0.1-alpha.1
git tag v0.0.1-alpha.1
git push origin v0.0.1-alpha.1
```

---

## 2. Releasing the Core Go Modules (`amnesia` or `sdk-go`)

Go modules in subdirectories require their own tags for the Go module proxy.

```bash
# 1. Tag the subdirectories
git tag amnesia/v0.0.1-alpha.1
git tag sdk-go/v0.0.1-alpha.1
git push origin amnesia/v0.0.1-alpha.1 sdk-go/v0.0.1-alpha.1

# 2. Update CLI and API dependencies
cd cli && go mod tidy
cd ../api && go mod tidy
cd ..

# 3. Commit and push the dependency bump
git commit -am "chore: bump core dependencies to v0.0.1-alpha.1"
git push origin main

# 4. Tag the root repo to build new binaries
git tag v0.0.1-alpha.1
git push origin v0.0.1-alpha.1
```

---

## 3. Releasing the TypeScript Packages (`@zenv-sh/sdk` or `amnesia`)

TypeScript packages use [Changesets](https://github.com/changesets/changesets) and are deployed by `.github/workflows/publish.yml`.

1. **Create a changeset during development:**
   ```bash
   pnpm changeset
   ```
   Follow the prompts to select packages and bump type. Commit the generated file.

2. **Merge to main:**
   A GitHub Action bot will open a "Version Packages" Pull Request.

3. **Publish:**
   Merge the "Version Packages" PR. The workflow will build and publish to npm automatically.

---

## Quick Reference

| What changed | Tag format | Workflow |
|---|---|---|
| CLI / API | `v0.0.x-alpha.y` | `release.yml` |
| `amnesia/` (Go) | `amnesia/v0.0.x-alpha.y` | Go module proxy |
| `sdk-go/` | `sdk-go/v0.0.x-alpha.y` | Go module proxy |
| `@zenv-sh/*` (TS) | Managed by Changesets | `publish.yml` |
