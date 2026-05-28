# Release Guide

zEnv is composed of several moving parts: the Go CLI/API, the core Go cryptographic modules, and the TypeScript packages. Because of the GitHub Actions workflows configured in the repository, releasing new versions is highly automated.

Depending on what you have changed, follow the instructions below to trigger a release.

---

## 1. Releasing the CLI and API Binaries

If you only made changes to the `cli/` or `api/` folders, releasing is as simple as pushing a root tag. The `.github/workflows/release.yml` GitHub Action will automatically intercept it, compile the binaries across all operating systems, and attach them to a new GitHub Release page.

1. **Ensure you are on the `main` branch with all changes pushed:**
   ```bash
   git checkout main
   git pull origin main
   ```

2. **Create a new version tag (must start with `v`):**
   ```bash
   git tag v0.1.1
   ```

3. **Push the tag to GitHub:**
   ```bash
   git push origin v0.1.1
   ```
*(That's it! GitHub will build and attach `linux` and `darwin` binaries to the release automatically).*

---

## 2. Releasing the Core Go Modules (`amnesia` or `sdk-go`)

Because Go workspaces manage modules independently, if you change the cryptographic engine (`amnesia/`) or the Go SDK (`sdk-go/`), you must explicitly tag those subdirectories so the global Go proxy can cache the new versions. Then, you bump the CLI/API to depend on them.

1. **Tag the specific subdirectories:**
   ```bash
   git tag amnesia/v0.1.2
   git tag sdk-go/v0.1.2
   git push origin amnesia/v0.1.2 sdk-go/v0.1.2
   ```

2. **Update the CLI and API module dependencies:**
   Open `cli/go.mod` and `api/go.mod` and update the `require` blocks to point to the new `v0.1.2` versions.
   ```bash
   # Run go mod tidy to ensure everything is correct
   cd cli && go mod tidy
   cd ../api && go mod tidy
   cd ..
   ```

3. **Commit the dependency bump:**
   ```bash
   git commit -am "chore: bump core dependencies to v0.1.2"
   git push origin main
   ```

4. **Tag the root repo to build the new binaries:**
   ```bash
   git tag v0.1.2
   git push origin v0.1.2
   ```

---

## 3. Releasing the TypeScript Packages (`@zenv/sdk` or `amnesia`)

Your TypeScript packages use [Changesets](https://github.com/changesets/changesets). The deployment is handled entirely by `.github/workflows/publish.yml`.

1. **Create a changeset during development:**
   When working on a feature in a branch, run:
   ```bash
   pnpm changeset
   ```
   Follow the CLI prompts to select which packages changed and whether it's a major, minor, or patch update. Commit the generated markdown file.

2. **Merge to Main:**
   When you merge your feature branch into `main`, a GitHub Action bot will read the changeset file and automatically open a new "Version Packages" Pull Request.

3. **Publishing:**
   When you are ready to officially release the packages, simply merge the "Version Packages" Pull Request. The GitHub Action will automatically build the packages and publish them to the GitHub NPM Registry.
