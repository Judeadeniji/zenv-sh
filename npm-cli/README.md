# @zenv-sh/cli

[![npm](https://img.shields.io/npm/v/@zenv-sh/cli)](https://www.npmjs.com/package/@zenv-sh/cli)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](../../LICENSE-MIT)

Platform-aware npm wrapper for the **zEnv CLI** binary. This package installs the correct native binary for your OS and architecture via optional dependencies.

## Install

```bash
npm install -g @zenv-sh/cli
```

## Supported Platforms

| OS | Architecture | Package |
|----|-------------|---------|
| Linux | x64 | `@zenv-sh/cli-linux-x64` |
| Linux | arm64 | `@zenv-sh/cli-linux-arm64` |
| macOS | x64 | `@zenv-sh/cli-darwin-x64` |
| macOS | arm64 | `@zenv-sh/cli-darwin-arm64` |

## How It Works

This package contains a thin Node.js wrapper (`bin/zenv.js`) that:

1. Detects the current platform and CPU architecture
2. Resolves the matching `@zenv-sh/cli-<os>-<cpu>` optional dependency
3. Spawns the native Go binary with the original arguments

The native binaries are installed as optional dependencies — npm automatically downloads only the one that matches your system.

## Usage

Once installed, the `zenv` command is available globally:

```bash
zenv --version
zenv login --api https://api.zenv.sh
zenv secrets list
zenv run -- node server.js
```

See the [CLI README](../cli/README.md) for full usage documentation.

## License

MIT — see [LICENSE](../LICENSE-MIT).
