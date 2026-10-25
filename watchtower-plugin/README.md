# CLN watchtower client

See the [client guide](../docs/CLIENTS.md) for setup and registration,
and [coverage](../docs/COVERAGE.md) for limitations.

The plugin receives `commitment_revocation` notifications and sends signed,
encrypted appointments to a TEOS endpoint. It cannot speak the native LND protocol.
Its receipts and queue are stored in its own data directory. Default server renewal
extends expiry without replenishing slots.
