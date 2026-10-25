# Native XBT watchtower support

`cmd/teos-lnd` is a tower-only executable using the native LND watchtower protocol.
It does not launch the LND wallet, channel node, routing or gossip subsystems.
`cmd/teos-lnd-migrate` is an offline identity migration helper.

This directory retains pinned LND source for libraries and client tooling.
Follow the [build guide](../docs/BUILD.md), [tower setup](../docs/TOWER.md),
[migration guide](../docs/MIGRATION.md) and [client guide](../docs/CLIENTS.md).
Only an XBT-aware Knots RPC backend is supported by the tower executable.

The [upstream README](README.upstream.md) is retained for attribution and background;
its full-node setup instructions are not the tower-only installation procedure.
See [LICENSE](LICENSE).
