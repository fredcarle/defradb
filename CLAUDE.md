# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

DefraDB is a user-centric database built on MerkleCRDTs and IPLD, featuring a multi-write-master architecture. It uses DQL (DefraDB Query Language), which is compatible with GraphQL but with additional features. The project emphasizes data ownership, privacy, and security with peer-to-peer networking capabilities.

## Build and Development Commands

### Building and Running

```bash
# Install DefraDB binary
make install

# Build binary to build/defradb
make build

# Build to custom path
make build path="path/to/defradb-binary"

# Run DefraDB from source
make start

# Run with development mode
make dev:start
```

### Testing

```bash
# Run all tests (unit and integration)
make test

# Run tests quickly without race detector
make test:quick

# Run specific test directory
gotestsum --format pkgname -- ./path/to/tests $(TEST_FLAGS)

# Run tests with coverage
make test:coverage

# Run tests in JavaScript/WASM mode
make test:js

# Run benchmark suite
make test:bench

# Run change detector tests (for breaking changes)
make test:changes

# Run single test (use gotestsum)
gotestsum --format pkgname -- ./internal/db -run TestSpecificTest

# Run tests with specific mutation type
DEFRA_MUTATION_TYPE=gql make test
DEFRA_MUTATION_TYPE=collection-named make test

# Run tests with SourceHub ACP
DEFRA_DOCUMENT_ACP_TYPE=source-hub make test

# Run tests via HTTP client
DEFRA_CLIENT_HTTP=true make test

# Run tests via CLI client
DEFRA_CLIENT_CLI=true make test

# Run tests via C bindings
DEFRA_CLIENT_C=true make test
```

### Linting and Code Quality

```bash
# Run all linters
make lint

# Auto-fix linting issues
make lint:fix

# Run comprehensive fix (deps, lint, tidy, mocks, docs)
make fix

# Run Go mod tidy
make tidy

# Generate mocks
make mocks

# Check for vulnerabilities
make deps:vulncheck
govulncheck ./...
```

### Dependencies

```bash
# Install all development dependencies
make deps

# Download Go modules
make deps:modules

# Install testing tools
make deps:test

# Install linters
make deps:lint

# Install Ollama (for AI features)
make deps:ollama
```

## Architecture Overview

### Core Components

- **`internal/db/`**: Core database logic, storage engine, transaction management, and query execution
- **`internal/planner/`**: Query planning and optimization for DQL queries
- **`internal/request/`**: Request parsing and GraphQL/DQL query handling
- **`internal/lens/`**: WebAssembly-based data transformations and migrations
- **`internal/encryption/`**: Encryption-at-rest implementation
- **`internal/kms/`**: Key management system for cryptographic keys

### Client and API

- **`client/`**: Go client library interface for interacting with DefraDB
- **`http/`**: HTTP API server implementation (REST and GraphQL endpoints)
- **`cli/`**: Command-line interface implementation
- **`cbindings/`**: C bindings for FFI integration
- **`js/`**: JavaScript/WASM bindings and adapters

### Networking

- **`node/`**: P2P node implementation using libp2p
- **`internal/db/p2p/`**: P2P synchronization and replication logic

### Access Control

- **`acp/`**: Access Control Policy (ACP) implementation
  - Document-level access control (DAC)
  - Node-level access control (NAC)
  - Uses Relation-Based Access Control (ReBAC) via SourceHub or local mode
  - Implements DPI (DefraDB Policy Interface) for policy validation

### Storage and Data

- **`internal/datastore/`**: Abstraction over underlying storage (Badger, Memory)
- **`internal/core/`**: Core data structures and interfaces
- **`crypto/`**: Cryptographic utilities for signing and key generation
- **`keyring/`**: Keyring management for storing encryption keys

### Testing

- **`tests/integration/`**: Integration tests organized by feature
- **`tests/bench/`**: Benchmark tests for performance tracking
- **`tests/change_detector/`**: Tests for detecting breaking changes in data format
- **`tests/clients/`**: Tests for different client types (HTTP, CLI, C)
- **`tests/lenses/`**: Lens (data migration) tests

## Configuration

DefraDB uses YAML configuration files. The default directory is `~/.defradb/` (can be changed with `--rootdir`).

Key configuration areas:
- **Datastore**: Backend selection (badger/memory), encryption settings
- **API**: HTTP address, TLS certificates, CORS settings
- **Network**: P2P addresses, pubsub settings, peer lists
- **Logging**: Level, format, output destination
- **Keyring**: Path, backend (file/system), namespace
- **ACP**: Document and node access control settings
- **Lens**: WASM runtime selection (wasm-time, wasmer, wazero)

See `docs/config.md` for detailed configuration options.

## Key Management

DefraDB uses a keyring system for managing cryptographic keys:

```bash
# Generate keys (interactive)
defradb keyring generate

# Import existing key
defradb keyring import <name> <private-key-hex>

# Set keyring secret via environment variable
export DEFRA_KEYRING_SECRET="your-secret"

# Or use .env file in working directory
echo "DEFRA_KEYRING_SECRET=your-secret" > .env

# Or specify secret file path
defradb start --secret-file /path/to/secret
```

Required keys loaded on start:
- `peer-key`: Ed25519 for P2P networking (required)
- `encryption-key`: AES key for encryption-at-rest (optional)
- `node-identity-key`: Secp256k1 for node identity and ACP (optional)

## Working with Schemas

```bash
# Add schema via CLI
defradb client schema add 'type User { name: String, age: Int }'

# Add schema from file
defradb client schema add -f examples/schema/bookauthpub.graphql

# Add schema with ACP policy
defradb client schema add '
  type Users @policy(
    id: "policy-id-here",
    resource: "users"
  ) {
    name: String
    age: Int
  }
'

# List schemas
defradb client schema describe
```

## Testing Patterns

### Integration Tests

Integration tests are located in `tests/integration/` and organized by feature area. They use a declarative test framework with action-based test definitions.

Common test patterns:
- Tests are defined in YAML-like structures with sequential actions
- Each test file focuses on a specific feature or scenario
- Use `tests/gen/` for generated test utilities

### Running Specific Tests

```bash
# Run all tests in a specific package
gotestsum --format pkgname -- ./internal/db

# Run a specific test by name
gotestsum --format pkgname -- ./internal/db -run TestSpecificFunction

# Run tests matching a pattern
gotestsum --format pkgname -- ./tests/integration/... -run "TestQuery.*"

# Run with verbose output
gotestsum --format standard-verbose -- ./path/to/tests

# Watch mode for development
make test:watch
```

## P2P Networking

DefraDB supports two types of P2P relationships:
1. **Pubsub peering**: Passive synchronization via document commit broadcasts
2. **Replicator peering**: Active pushing of collection changes to target peers

```bash
# Get peer info
defradb client p2p info

# Subscribe to collection updates
defradb client p2p collection add <collectionID>

# Set up replicator
defradb client p2p replicator set -c <CollectionName> <peer-multiaddr>
```

## Access Control (ACP)

### Document Access Control (DAC)

```bash
# Generate identity key
openssl ecparam -name secp256k1 -genkey | openssl ec -text -noout | head -n5 | tail -n3 | tr -d '\n:\ '

# Add policy
defradb client acp document policy add -f examples/policy/dac_policy.yml --identity <private-key-hex>

# Create private document
defradb client collection create --name Users '[{"name": "private"}]' --identity <private-key-hex>

# Add relationship to share document
defradb client acp document relationship add \
  --collection Users \
  --docID <docID> \
  --relation reader \
  --actor <did:key:...> \
  --identity <private-key-hex>

# Delete relationship to revoke access
defradb client acp document relationship delete \
  --collection Users \
  --docID <docID> \
  --relation reader \
  --actor <did:key:...> \
  --identity <private-key-hex>
```

### DPI (DefraDB Policy Interface) Requirements

Policies must include:
- Required `owner` relation (registerer)
- Required permissions: `read`, `update`, `delete`
- Owner must be first in permission expressions
- Union operations (+) only after owner relation

See `acp/README.md` and `tests/integration/acp/schema/add_dpi/` for examples.

## Development Workflow

Following the project's conventions:

1. **Issue-driven development**: Every PR links to one or more issues
2. **Squash and merge**: Commits are squashed before merging to `develop`
3. **Conventional commits**: Format: `<type>: <description>`
   - Types: `feat`, `fix`, `tools`, `docs`, `refactor`, `test`, `ci`, `chore`, `bot`
4. **Branch naming**: `<name>/<type>/<description>` (e.g., `alice/feat/new-index-type`)
5. **Breaking changes**: Include `BREAKING CHANGE` in commit body with description

### Prerequisites

- Go 1.24.6 or later
- Rust/Cargo (for building lenses)
- SourceHub (for ACP tests): Install via `make install` in sourcehub repo

### Before Submitting

```bash
# Run full test suite
make test

# Run linters
make lint

# For breaking changes, document in docs/data_format_changes/
# Then run change detector tests
make test:changes
```

## Special Build Tags

```bash
# Build with telemetry support
BUILD_TAGS=telemetry make build

# Run tests with specific features
go test -tags change_detector ./tests/change_detector/...
```

## Common Development Tasks

### Adding a New Feature

1. Create issue on GitHub
2. Create branch: `yourname/feat/feature-description`
3. Implement feature with tests
4. Run `make test` and `make lint`
5. Create PR targeting `develop` branch
6. Use conventional commit format when merging

### Debugging Tests

```bash
# Run with race detector (default in make test)
go test -race ./...

# Run without race detector for speed
make test:quick

# Run specific test with verbose output
gotestsum --format standard-verbose -- ./path -run TestName

# Generate coverage HTML
make test:coverage-html
```

### Working with Mocks

```bash
# Regenerate all mocks
make mocks

# Mocks are generated using mockery (see tools/configs/mockery.yaml)
```

## License

DefraDB uses the Business Source License (BSL). Include the BSL license header at the top of every code file. See `licenses/BSL.txt` for details.

## Additional Resources

- Documentation: https://docs.source.network/
- Discord: https://discord.gg/w7jYQVJ
- GitHub Issues: https://github.com/sourcenetwork/defradb/issues
- SIPs (Source Improvement Proposals): https://github.com/sourcenetwork/SIPs/
