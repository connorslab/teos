# The Eye of Satoshi — XBT watchtowers

Watchtower software for Bitcoin BLAKE2b (XBT), with Core Lightning and LND support
in one source repository. **Experimental; coverage is limited to tested scenarios.**

| Client | Backend | Protocol |
| --- | --- | --- |
| Core Lightning | `teosd` and `watchtower-client` | TEOS signed HTTP appointments, based on the BOLT13 proposal |
| LND | Bundled native LND tower | LND encrypted watchtower sessions |

Both backends can run simultaneously, with separate identities, storage and
listeners. Their protocols are not interchangeable. BOLT13 here means the proposal
underlying TEOS, not a finalized universal watchtower interoperability standard.

## Getting started

1. [Build the software](docs/BUILD.md).
2. [Run a tower](docs/TOWER.md).
3. [Connect CLN and LND clients](docs/CLIENTS.md).
4. Read [coverage and limitations](docs/COVERAGE.md) and
   [backup, monitoring and upgrade guidance](docs/OPERATIONS.md).

Use an XBT-aware Knots RPC backend. Stock Bitcoin's SHA256 chain is incompatible,
even though the networks share their genesis hash. RPC and administrative APIs
must remain private. No hosted endpoint is configured by default.

## Source layout

- `teos/`: appointment server and administrative CLI.
- `teos-common/`: shared protocol types and cryptography.
- `watchtower-plugin/`: CLN client using `commitment_revocation`.
- `lnd/`: complete XBT LND source, including native client and tower.
- `vendor/`: pinned Rust consensus/header dependencies.
- `examples/`: local-only configuration templates.

XBT uses BLAKE2b for consensus block hashing. Transaction identifiers and
appointment encryption retain their protocol-defined hashing.

## Attribution

Based on [The Eye of Satoshi](https://github.com/talaia-labs/rust-teos) and
[LND](https://github.com/lightningnetwork/lnd). Copyright and license notices are
retained. See [provenance](docs/PROVENANCE.md), [LICENSE](LICENSE), and
[lnd/LICENSE](lnd/LICENSE). Upstream projects do not endorse this fork.

See [SECURITY.md](SECURITY.md) for private vulnerability reporting.
