# Xatu Development Tools

This directory contains development and testing utilities for the Xatu project.

## Tools

### ethstats-client

A test client that simulates an Ethereum node reporting to the ethstats server. Used for testing and development of the ethstats server implementation.

**Usage:**
```bash
cd tools/ethstats-client
go run -tags tools main.go --ethstats username:password@localhost:8081
```

See [ethstats-client/README.md](ethstats-client/README.md) for detailed documentation.

## Building Tools

All tools can be built individually:

```bash
# Build ethstats-client
go build -o bin/ethstats-client ./tools/ethstats-client

# Or build all tools
make tools
```

## Contributing

When adding new development tools:

1. Create a new subdirectory under `tools/`
2. Include a README.md with usage instructions
3. Update this main README with a description
4. Consider adding a Makefile target if the tool should be built regularly

## Note

These tools are for development and testing purposes only. They are not included in production builds or releases.