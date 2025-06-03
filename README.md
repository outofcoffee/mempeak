# mempeak

A Unix command-line tool to monitor peak memory usage of processes, similar to `time` but for memory.

## Example

[![Build Status](https://github.com/outofcoffee/mempeak/workflows/ci/badge.svg)](https://github.com/outofcoffee/mempeak/actions)
[![Go Report Card](https://goreportcard.com/badge/github.com/outofcoffee/mempeak)](https://goreportcard.com/report/github.com/outofcoffee/mempeak)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

```
$ mempeak some-command
... (normal command output)

total peak memory usage: 387.2 MB
```

## Overview

`mempeak` is a Go port of the original `memusg` tool that monitors the peak memory usage of a command and all its subprocesses during execution. It provides a simple, `time`-like interface for tracking memory consumption across entire process trees.

## Features

- **Cross-platform**: Works on Linux, macOS, and other Unix-like systems
- **Simple interface**: Just prefix your command with `mempeak`
- **Process tree tracking**: Monitors the entire process tree, including all subprocesses
- **Per-process breakdown**: Shows memory usage for each process with PID and name
- **Peak memory tracking**: Monitors memory usage throughout execution
- **Human-readable output**: Displays memory in appropriate units (B, KB, MB, GB, etc.)
- **Exit code preservation**: Maintains the exit code of the monitored command
- **Lightweight**: Minimal overhead with 100ms polling interval

## Installation

### Homebrew (macOS/Linux)

```bash
brew tap outofcoffee/tap
brew install mempeak
```

Or install directly:
```bash
brew install outofcoffee/tap/mempeak
```

### Pre-built binaries

Download the latest release from the [releases page](https://github.com/outofcoffee/mempeak/releases).

### Using Go

```bash
go install github.com/outofcoffee/mempeak@latest
```

### From source

```bash
git clone https://github.com/outofcoffee/mempeak.git
cd mempeak
make install
```

## Usage

```bash
mempeak <command> [args...]
```

### Examples

Monitor memory usage of a simple command:
```bash
mempeak ls -la
```

Monitor a long-running process:
```bash
mempeak ./my-application --config config.json
```

Monitor a build process:
```bash
mempeak make build
```

### Output

`mempeak` outputs detailed memory usage information to stderr after the command completes, showing both per-process breakdown and total usage:

```
$ mempeak sh -c 'echo "parent"; (sleep 1; echo "child1") & (sleep 1; echo "child2") & wait'
parent
child1
child2
mempeak: process tree memory usage:
  PID 12345 (sh): 1.8 MB
  PID 12346 (sh): 1.3 MB
  PID 12347 (sh): 1.4 MB
  PID 12348 (sleep): 1.1 MB
  PID 12349 (sleep): 1.1 MB
mempeak: total peak memory usage: 6.8 MB
```

For commands that don't spawn subprocesses, the output shows just the main process:

```
$ mempeak node my-script.js
... (normal command output)
mempeak: process tree memory usage:
  PID 12350 (node): 387.2 MB
mempeak: total peak memory usage: 387.2 MB
```

## How it works

1. Starts the target command as a child process
2. Continuously discovers all processes in the process tree (parent and children)
3. Monitors memory usage for each process in the background using:
   - `/proc/pid/status` and `/proc/pid/stat` on Linux (more efficient)
   - `ps` command on macOS/Unix systems (fallback)
4. Tracks peak memory usage for each process throughout execution (100ms polling interval)
5. Reports per-process breakdown and total peak memory when the command completes
6. Exits with the same code as the monitored command

## Building

Requirements:
- Go 1.19 or later

```bash
# Build for current platform
make build

# Build for all platforms
make build-all

# Install locally
make install

# Run tests
make test

# Clean build artifacts
make clean
```

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Acknowledgments

- Inspired by the original [memusg](https://github.com/jhclark/memusg) by Jonathan Clark
- Built as a portable Go alternative with cross-platform support
