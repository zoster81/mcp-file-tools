#!/usr/bin/env node

const fs = require('node:fs');
const path = require('node:path');

const semanticVersionPattern = /^\d+\.\d+\.\d+$/;
const releaseTagPattern = /^v(\d+\.\d+\.\d+)$/;

function readJSON(filePath) {
  let raw;
  try {
    raw = fs.readFileSync(filePath, 'utf8');
  } catch (error) {
    throw new Error(`could not read ${filePath}: ${error.message}`);
  }

  try {
    return JSON.parse(raw);
  } catch (error) {
    throw new Error(`invalid JSON in ${filePath}: ${error.message}`);
  }
}

function requireSemanticVersion(value, label) {
  if (typeof value !== 'string' || !semanticVersionPattern.test(value)) {
    throw new Error(`${label} must be a semantic version in major.minor.patch form`);
  }
  return value;
}

function verifyReleaseVersion(tag, root = path.resolve(__dirname, '..')) {
  const tagMatch = releaseTagPattern.exec(tag || '');
  if (!tagMatch) {
    throw new Error(`expected semantic release tag v<major.minor.patch>, got ${JSON.stringify(tag || '')}`);
  }
  const version = tagMatch[1];

  const pluginPath = path.join(root, 'plugin', '.claude-plugin', 'plugin.json');
  const marketplacePath = path.join(root, '.claude-plugin', 'marketplace.json');
  const plugin = readJSON(pluginPath);
  const marketplace = readJSON(marketplacePath);

  const pluginVersion = requireSemanticVersion(plugin.version, 'plugin version');
  const marketplaceEntry = Array.isArray(marketplace.plugins)
    ? marketplace.plugins.find((entry) => entry && entry.name === 'mcp-file-tools')
    : undefined;
  if (!marketplaceEntry) {
    throw new Error('marketplace metadata does not contain the mcp-file-tools plugin');
  }
  const marketplaceVersion = requireSemanticVersion(
    marketplaceEntry.version,
    'marketplace version',
  );

  if (marketplaceVersion !== pluginVersion) {
    throw new Error(
      `marketplace version ${marketplaceVersion} does not match plugin version ${pluginVersion}`,
    );
  }
  if (version !== pluginVersion) {
    throw new Error(`tag version ${version} does not match plugin version ${pluginVersion}`);
  }

  return { version, pluginVersion, marketplaceVersion };
}

if (require.main === module) {
  try {
    const result = verifyReleaseVersion(process.argv[2]);
    console.log(`release version verified: v${result.version}`);
  } catch (error) {
    console.error(`error: ${error.message}`);
    process.exitCode = 1;
  }
}

module.exports = { verifyReleaseVersion };
