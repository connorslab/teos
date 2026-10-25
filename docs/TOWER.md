# Run both backends

Build the binaries using BUILD.md. Use synchronized XBT-aware Knots with RPC
enabled. Stock SHA256 Bitcoin is incompatible. A pruned backend must retain every
block the tower still needs to scan; node synchronization alone is insufficient.

Run each backend as a separate unprivileged account with private storage. Commands
below run as the account assigned to that backend, from the source checkout.
The two backends do not need channel funding.

## CLN / TEOS backend

```sh
mkdir -m 700 "$HOME/.teos"
cp examples/teos.toml "$HOME/.teos/teos.toml"
chmod 600 "$HOME/.teos/teos.toml"
```

Edit the RPC cookie path and grant this account read access to that file.
Alternatively configure a dedicated RPC username/password. Do not grant general
access to a node's wallet directory. Start the daemon:

```sh
target/release/teosd --datadir "$HOME/.teos"
```

In another terminal, using the same account:

```sh
target/release/teos-cli --datadir "$HOME/.teos" gettowerinfo
```

Record the generated tower ID. The appointment API is on loopback port 9814.
Administrative RPC on 8814 and the internal API on 50051 must remain private.
For remote clients, put a TLS reverse proxy in front of 9814 with a valid
certificate, admission policy, and request-size/rate limits. Do not proxy the
administrative or internal APIs. Test the proxy with your supported appointments.

`max_registered_users` bounds identities, not HTTP request frequency. With
`renewal_slot_topups=false`, renewal extends expiry without adding slots.
Monitor remaining slots and define a capacity policy before onboarding.

## Native LND backend

```sh
mkdir -m 700 "$HOME/.lnd-tower"
cp examples/lnd.conf "$HOME/.lnd-tower/lnd.conf"
chmod 600 "$HOME/.lnd-tower/lnd.conf"
```

Edit the XBT RPC cookie path, then start:

```sh
bin/lnd --lnddir="$HOME/.lnd-tower"
```

In another terminal initialize a **new** wallet:

```sh
bin/lncli --lnddir="$HOME/.lnd-tower" create
```

Save recovery material securely. Never import a channel wallet or use
`noseedbackup` for a production tower. No wallet funding is required.
After later restarts, unlock using LND's unlock mechanism and protect any
unattended-unlock credential.

```sh
bin/lncli --lnddir="$HOME/.lnd-tower" getinfo
bin/lncli --lnddir="$HOME/.lnd-tower" tower info
```

Wait for synchronization and listener readiness. Record the tower public key.
For remote clients, change `watchtower.listen` to the intended address,
configure `watchtower.externalip`, and initially allow TCP 9911 only from intended
clients. This is an encrypted native protocol, not HTTP. Keep RPC/REST private.

A global LND session admission cap has not been established. Restricted access,
storage quotas and monitoring are needed before considering open registration.

## Verify both independently

Check TEOS backend reachability and scanner progress, plus LND readiness and
tower logs. Follow CLIENTS.md and verify client acknowledgements. A listening
port or registration is not proof that a revoked state was stored. Keep both
processes independently restartable. See OPERATIONS.md for recovery guidance.
