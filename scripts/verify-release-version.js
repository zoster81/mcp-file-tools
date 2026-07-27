#!/usr/bin/env node

const fs = require('node:fs');
const path = require('node:path');

const releaseTagPattern = /^v(\d+\.\d+\.\d+)$/;

function escapeRegExp(value) {
  return value.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
}

function readText(filePath) {
  try {
    return fs.readFileSync(filePath, 'utf8');
  } catch (error) {
    throw new Error(`could not read ${filePath}: ${error.message}`);
  }
}

function verifyReleaseVersion(tag, root = path.resolve(__dirname, '..')) {
  const tagMatch = releaseTagPattern.exec(tag || '');
  if (!tagMatch) {
    throw new Error(`expected semantic release tag v<major.minor.patch>, got ${JSON.stringify(tag || '')}`);
  }
  const version = tagMatch[1];
  const changelogPath = path.join(root, 'CHANGELOG.md');
  const changelog = readText(changelogPath);
  const releaseHeading = new RegExp(
    `^## ${escapeRegExp(version)} - \\d{4}-\\d{2}-\\d{2}\\s*$`,
    'm',
  );
  if (!releaseHeading.test(changelog)) {
    throw new Error(
      `CHANGELOG.md does not contain a dated release heading for ${version}`,
    );
  }
  return { version, changelogVersion: version };
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
