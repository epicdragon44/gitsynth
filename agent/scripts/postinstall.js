#!/usr/bin/env node

const fs = require('fs');
const path = require('path');
const { promisify } = require('util');
const { exec } = require('child_process');

const execAsync = promisify(exec);
const copyFileAsync = promisify(fs.copyFile);
const chmodAsync = promisify(fs.chmod);

const binDir = path.join(__dirname, '..', 'bin');
const mainBinaryPath = path.join(binDir, 'gitsynth');

async function main() {
  try {
    // Determine the platform-specific binary
    let sourceBinary;
    if (process.platform === 'linux') {
      sourceBinary = path.join(binDir, 'gitsynth-linux');
    } else if (process.platform === 'darwin') {
      sourceBinary = path.join(binDir, 'gitsynth-darwin');
    } else if (process.platform === 'win32') {
      sourceBinary = path.join(binDir, 'gitsynth-win.exe');
    } else {
      console.warn(`Unsupported platform: ${process.platform}. Using default binary.`);
      sourceBinary = mainBinaryPath;
    }

    // Only copy if platform-specific binary exists and is different from main binary
    if (sourceBinary !== mainBinaryPath) {
      try {
        if (fs.existsSync(sourceBinary)) {
          await copyFileAsync(sourceBinary, mainBinaryPath);
          console.log(`Using platform-specific binary for ${process.platform}`);
        }
      } catch (err) {
        console.warn(`Could not copy platform-specific binary: ${err.message}`);
        console.warn('Using default binary instead.');
      }
    }

    // Make binary executable on non-Windows platforms
    if (process.platform !== 'win32') {
      console.log('Setting executable permissions...');
      await chmodAsync(mainBinaryPath, 0o755);
    }

    console.log('GitSynth installed successfully!');
    console.log('Run "gitsynth" to get started');
  } catch (error) {
    console.error('Error during installation:', error);
    process.exit(1);
  }
}

main();