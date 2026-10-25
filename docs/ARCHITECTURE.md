# Architecture and implementation status

TEOS serves CLN appointments. Optional LND support serves the separate native
LND watchtower protocol. These protocols require different session handling,
cryptography, identities and persistence; they are not interchangeable.

## Implemented

- One `watchtower.toml` owns network, Knots RPC endpoint/cookie and public bind.
- `tools/configure.py` renders the selected backend configurations without
  restarting processes. LND generation is opt-in with `[lnd].enabled = true`.
- CLN-only builds and deployments do not require the Go toolchain or LND daemon.
- Both backends can operate simultaneously, with independent databases and IDs.

## Current limitation

The native-protocol backend still builds and runs the full LND daemon from `lnd/`.
It requires wallet initialization and unlock even though tower operation needs no
channel funding. Bundling this source and generating its configuration is not a
standalone LND watchtower integration. No shared database, shared chain scanner,
or single-process runtime is implemented.

## Standalone backend work remaining

The intended next step is a tower-only Go executable using LND's watchtower
packages, with explicit identity persistence, chain notification and transaction
broadcast adapters. It must avoid initializing a channel wallet or peer node.
Before replacing the current backend, verify existing tower identity/database
compatibility, native client enrollment and acknowledgements, restart/reorg
recovery and breach recovery on disposable XBT regtest networks. Existing tower
identities and acknowledged sessions must survive migration. Until that is
implemented and tested, the setup guide deliberately documents the full-daemon
requirement rather than advertising a standalone feature that does not exist.
