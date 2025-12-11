#!/usr/bin/env node

/**
 * Updates package.json version from CI environment variable.
 * Usage: VERSION=1.0.0 node scripts/set-version.js
 */

const fs = require('fs');
const path = require('path');

const version = process.env.VERSION;

if (!version) {
  console.log('No VERSION env var set, skipping version update');
  process.exit(0);
}

// Validate semver format (loose check)
if (!/^\d+\.\d+\.\d+/.test(version)) {
  console.error(`Invalid version format: ${version}`);
  console.error('Expected semver format like 1.0.0 or 1.0.0-beta.1');
  process.exit(1);
}

const packageJsonPath = path.join(__dirname, '..', 'package.json');
const packageJson = JSON.parse(fs.readFileSync(packageJsonPath, 'utf8'));

const oldVersion = packageJson.version;
packageJson.version = version;

fs.writeFileSync(packageJsonPath, JSON.stringify(packageJson, null, 2) + '\n');

console.log(`Updated version: ${oldVersion} -> ${version}`);
