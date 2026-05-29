#!/usr/bin/env node

const { spawnSync } = require('child_process');
const os = require('os');
const path = require('path');

const platform = os.platform();
const arch = os.arch();

const knownPlatforms = {
  darwin: 'darwin',
  linux: 'linux',
};

const knownArchs = {
  x64: 'x64',
  arm64: 'arm64'
};

const p = knownPlatforms[platform];
const a = knownArchs[arch];

if (!p || !a) {
  console.error(`Unsupported platform/architecture: ${platform}-${arch}`);
  process.exit(1);
}

const packageName = `@zenv-sh/cli-${p}-${a}`;
const binName = 'zenv';

let binPath;
try {
  // Find the exact path to the binary provided by the optional dependency
  binPath = path.join(require.resolve(`${packageName}/package.json`), '..', 'bin', binName);
} catch (e) {
  console.error(`Failed to find optional dependency ${packageName}.`);
  console.error('Make sure you are not using --no-optional or --ignore-scripts if your package manager ignores optional deps.');
  process.exit(1);
}

const result = spawnSync(binPath, process.argv.slice(2), { stdio: 'inherit' });
if (result.error) {
  console.error(result.error);
  process.exit(1);
}
process.exit(result.status ?? 0);
