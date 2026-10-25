# Architecture

One configuration selects two independent protocol backends:

| Backend | Clients | Runtime |
| --- | --- | --- |
| `teosd` | CLN watchtower plugin | Rust TEOS appointment server |
| `teos-lnd` (optional) | XBT LND watchtower clients | Go tower-only server |

The shared renderer owns network, Knots RPC authentication and public bind.
Neither backend starts or restarts the other. CLN-only installations need no Go
runtime or LND executable. Separate identities and databases are intentional:
the protocols have different authentication, sessions and encrypted payloads.

## Native LND backend

`teos-lnd` uses LND's Brontide transport, `wtserver`, `wtdb`, encrypted blob decoding
and justice-transaction construction. It supplies its own persistent identity,
Knots RPC adapter and sequential block scanner. It does not initialize a wallet,
channel manager, routing graph, gossip service or LND RPC/REST listeners.

The pinned LND library source remains in-tree. This is library reuse, not a wrapper
that launches the full LND daemon. Some transitive library dependencies remain;
source-tree size does not represent the running services.

A new database records the current chain tip before accepting clients. An imported
database uses its existing cursor, or requires an explicit conservative starting
height when no cursor exists. Blocks are scanned in order. Failed reads and writes
leave the cursor behind the unprocessed block. Reorganizations walk back through
orphan headers to the common ancestor and replay the replacement chain.

For matching encrypted backups, reconstructed recovery transactions are committed
to a separate bucket in the same Bolt database before advancing the cursor.
Transactions are rebroadcast after restart and reorganization. Already-confirmed
RPC responses are accepted; other failures are reported through readiness. There
is no automatic fee replacement or pruning of retained remedies.

## Limits

This remains experimental. See COVERAGE.md for commitment-type limits. It supports
altruist sessions, not paid/reward sessions, and does not provision Tor services.
The RPC backend must retain all blocks needed for catch-up. There is no shared
scanner or shared database between CLN and LND, and no automatic migration of an
existing full-daemon tower. Follow MIGRATION.md before replacing one.
