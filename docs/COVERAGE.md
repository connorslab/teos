# Coverage and limitations

This is an experimental XBT fork. Enrollment, sessions and acknowledgements are
different states. Only acknowledged eligible revocations are available for recovery.

## Recorded validation of the source lineage

Disposable XBT regtest runs demonstrated:

- CLN penalty recovery and native LND anchor-channel recovery, with protected
  nodes offline and recovered outputs independently identified by their wallets.
- Concurrent CLN/LND recovery after both towers restarted with acknowledged data.
- Recovery after missed blocks and restored tower backups, including recovery
  while the other protocol backend remained unavailable.
- A shallow reorg and cooperative closes without an unexpected penalty in the
  observation window.
- TEOS authentication rejection, registration caps and renewal without slot top-ups.
- TEOS pending remedies surviving injected RPC fee-policy rejection and restart.
- TEOS filesystem exhaustion stopping processing before advancing the saved
  scanner cursor; replay after storage recovery confirmed the remedy.
- Native XBT handshake encoding and rejection of peers without the XBT feature bit.

These are historical test observations, not a guarantee for all configurations.
The software-only export retains unit tests and XBT mock-RPC fixtures. Host-specific
live test harnesses and private operational records are not included; the live
scenarios above are not automatically reproduced by the CI workflow.

## Tower-only backend validation

The tower-only executable adds tests for native encrypted session negotiation and
backup acknowledgement, durable cursor/remedy storage across reopening, missed
blocks, failed block reads, failed cursor writes, and replacement-chain replay.
These tests are included in CI.

An isolated regtest run additionally replayed an existing acknowledged breach
fixture, reconstructed its expected recovery transaction, persisted the remedy,
and caught up after restart, an RPC outage and a chain reorganization. This replay
used an already-confirmed recovery transaction; it is not a new live-channel breach
experiment and does not extend the channel-type coverage listed above.

## Not established

- Exhaustive channel-type coverage, splicing coverage, and taproot breach recovery.
  Negotiating a taproot session is not a taproot recovery test.
- LND HTLC-output recovery. Native tower coverage is limited to supported main
  commitment outputs; do not assume HTLC outputs are protected.
- Fee bumping or confirmation during sustained real mempool congestion. Retrying
  a signed remedy after rejection does not increase its fee.
- Open-registration abuse resistance, global native LND session admission limits,
  or capacity guarantees.
- Retroactive protection of states revoked before enrollment.
- Recovery of acknowledgements newer than a restored tower backup.

Keep channel backups and node security practices. Test your actual channel types
in isolated regtest before relying on them. Never bypass network, signature,
proof-of-work or feature checks to make a test pass.
