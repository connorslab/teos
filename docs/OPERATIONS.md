# Operating and upgrading

Keep chain RPC and tower administration private. Give each backend an independent
OS user, identity, storage directory and resource limits. Neither backend should
require restarting the other or a channel node. Monitor both separately.

## Monitoring

Track node synchronization, backend availability, TEOS scanner progress, storage
headroom, backup freshness, pending remedies and broadcast/confirmation failures.
LND's node sync and listener readiness do not independently prove the native
lookout's durable cursor. Inspect its logs and verify client acknowledgements.
On clients, monitor registration expiry, slots, sessions and pending/invalid work.
Session existence or TCP reachability alone is not a complete protection check.

## Backups

Back up identities and databases together using a consistent snapshot. Stop only
the affected tower for a cold backup, or use a validated filesystem/database
snapshot method. Never copy live database files casually. Encrypt backups, store
them independently of the tower, verify integrity, and rehearse restoration in
isolation. Include client identities and receipt/queue databases.

A restored backup can lack appointments acknowledged afterward. Reconcile client
records, queues and chain catch-up before treating a restored tower as current.
Never restore an old production channel database as part of tower recovery.

## Upgrades

Build and test before stopping anything. Preserve identities and data, back up,
then upgrade one backend at a time. Verify both node and dependent client readiness.
Do not describe an upgrade as complete solely because the process is running.

TEOS adds a `pending_broadcast` tracker column on startup. Older binaries may
ignore pending state and discard remedies after fee-policy rejection. Do not run
older TEOS against upgraded data. A binary rollback is not a database rollback.

## Admission

Use authentication, TLS for the TEOS endpoint, bounded requests, storage quotas
and explicit enrollment policy. The identity cap is not a request-rate limit.
The LND native port needs network admission controls; it is not an HTTP endpoint.
Do not deploy open registration based only on the defaults in this repository.
