# Connect clients

Use XBT clients and obtain the tower identity and endpoint through a trusted
channel. This repository has no default hosted endpoint.

## Core Lightning

Build `watchtower-client`. Your XBT CLN must provide the
`commitment_revocation` hook. Load the plugin using the node's account:

```sh
lightning-cli plugin start /absolute/path/to/watchtower-client
lightning-cli plugin list
```

For persistence add `plugin=/absolute/path/to/watchtower-client` to CLN's config.
Dynamic loading does not require a node restart. The plugin's identity and receipt
database live in `~/.watchtower`; set `TOWERS_DATA_DIR` in the node environment
to use another directory. Give each client its own writable directory.
Minimal containers need OpenSSL and a CA trust bundle for HTTPS.

Replace these placeholders with the TEOS identity and endpoint:

```sh
lightning-cli registertower TOWER_ID https://tower.example 443
lightning-cli pingtower TOWER_ID
lightning-cli getsubscriptioninfo TOWER_ID
lightning-cli gettowerinfo TOWER_ID
```

Check receipts, expiry, remaining slots and pending/invalid appointments.
Confirm appointment receipts as eligible revocations occur. `retrytower TOWER_ID`
retries pending work. Do not use `abandontower` as a troubleshooting shortcut:
it deletes local records. Renew before expiry; the default server does not add
slots on renewal.

## LND

The bundled LND includes the native client and XBT handshake. Other clients must
support the same network feature bits and channel formats. Follow the existing
node's upgrade procedure before changing its binary.

Add to the client node's configuration:

```ini
[wtclient]
wtclient.active=true
```

Activation requires a node restart. Schedule it around in-flight activity,
preserve identity/databases, and verify channels afterward. Using that node's
normal CLI connection options:

```sh
lncli wtclient add TOWER_PUBKEY@tower.example:9911
lncli wtclient towers
lncli wtclient stats
```

Confirm acquired sessions, not merely `active_session_candidate=true`.
Failed negotiations retry with backoff. Both ends must advertise XBT support;
never remove the network check to connect an incompatible peer.
Use `lncli wtclient tower --help` for session-detail options and verify
acknowledged updates; aggregate task counts alone are insufficient.

Enrollment does not automatically import historical revoked states. Only eligible
data delivered and acknowledged is covered. No payment is necessary to register.
Test breaches only on disposable regtest channels.
