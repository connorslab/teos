# Building

The combined setup guide targets Linux. Other platforms have not been validated.
Install Git, a C/C++ toolchain, pkg-config, OpenSSL development headers, Protocol
Buffers (`protoc`), Rust through rustup, and Python **3.11+** for shared configuration. Go **1.25.13** is needed only
for optional LND support.

On Debian/Ubuntu, install native dependencies with:

```sh
sudo apt-get update
sudo apt-get install build-essential pkg-config libssl-dev protobuf-compiler git ca-certificates
```

Rust **1.88.0** is selected by `rust-toolchain.toml`. Run from the repository root:

```sh
git clone https://github.com/connorslab/teos.git
cd teos
cargo build --locked --release --workspace
```

The default CLN/TEOS build is complete. For optional LND support, also run:

```sh
mkdir -p bin
(cd lnd && GOTOOLCHAIN=local go build -mod=readonly \
  -o ../bin/ ./cmd/teos-lnd ./cmd/teos-lnd-migrate)
```

Outputs: `target/release/teosd`, `target/release/teos-cli`,
`target/release/watchtower-client`, and optionally `bin/teos-lnd` and
`bin/teos-lnd-migrate`. The migration helper is an offline tool, not a service.
No external patches are required. The pinned LND source remains in-tree for its
libraries and client tooling, but the full LND daemon is not built by these steps.

For constrained builds use `CARGO_BUILD_JOBS=1`, `GOMAXPROCS=1`, `GOFLAGS=-p=1`,
`GOMEMLIMIT=600MiB`, and `GOGC=50`. These reduce concurrency, but do not guarantee
that a given memory limit is sufficient.

## Tests

```sh
cargo test --locked --workspace
(cd lnd && GOTOOLCHAIN=local go test -mod=readonly ./cmd/teos-lnd ./cmd/teos-lnd-migrate ./watchtower/...)
```

The XBT polling integration tests use mock RPC fixtures. Live breach tests require
disposable XBT regtest nodes. Never use live channels for testing. See COVERAGE.md.

Keep Cargo.lock and Go lockfiles unchanged when reproducing these builds. Pinned
inputs improve repeatability; bit-for-bit reproducible binaries are not claimed.
