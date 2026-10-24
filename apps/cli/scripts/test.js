#!/usr/bin/env node

const { execSync } = require('child_process');
const path = require('path');

function runTests() {
  console.log('Testing issuegraph installation...\n');

  const issuegraphPath = path.join(__dirname, '..', 'bin', 'issuegraph.js');

  try {
    // Test 1: Version check
    console.log('Test 1: Checking issuegraph version...');
    const version = execSync(`node "${issuegraphPath}" version`, { encoding: 'utf8' });
    console.log(`✓ Version check passed: ${version.trim()}\n`);

    // Test 2: Help command
    console.log('Test 2: Checking issuegraph help...');
    execSync(`node "${issuegraphPath}" --help`, { stdio: 'pipe' });
    console.log('✓ Help command passed\n');

    console.log('✓ All tests passed!');
  } catch (err) {
    console.error('✗ Tests failed:', err.message);
    process.exit(1);
  }
}

runTests();
