# Run the tower and optional LND backend

Build the binaries using BUILD.md. Use synchronized XBT-aware Knots with RPC
enabled. Stock SHA256 Bitcoin is incompatible. A pruned backend must retain every
block the tower still needs to scan; node synchronization alone is insufficient.

Run each backend as a separate unprivileged account with private storage. Commands
below run as the account assigned to that backend, from the source checkout.
The two backends do not need channel funding.

## Configure common settings once

Python 3.11+ is required for the configuration tool. From the repository root:

```sh
cp examples/watchtower.toml watchtower.toml
```

Edit the network, Knots host/port and cookie path once in `watchtower.toml`.
For LND client support, set `enabled = true` in `[lnd]`; it defaults to false. Then:

```sh
python3 tools/configure.py watchtower.toml --check
python3 tools/configure.py watchtower.toml --output generated
```

This creates `generated/teos/teos.toml` and, when LND support is enabled,
`generated/lnd/lnd.conf`, with private
file permissions on POSIX. It does not contact Knots, read the cookie, initialize
wallets, change ownership or restart processes. Both service accounts need read
access to the same RPC cookie file. Cookie authentication is the supported shared
mode; custom username/password deployments can use the native configs instead.

Install each generated file into its backend account's data directory with owner
read/write permissions. Do not share the data directories themselves. For example,
under each respective account, use the appropriate command below instead of
copying a hand-edited native template. An administrator can install on behalf of
separate accounts using `install -o ACCOUNT -g GROUP -m 600 SOURCE DESTINATION`.

After changing shared settings, render again using `--overwrite`. This replaces
only files marked as generated and refuses unmanaged configs. Install the enabled outputs
and perform your normal controlled restarts, one backend at a time. The renderer
never reloads running daemons. Do not edit generated files as the source of truth.
If writing fails, rerun successfully before installing either output.

Shared settings cover XBT mainnet/regtest, Knots RPC and the public listener bind
address. Backend sections contain port overrides and TEOS admission/subscription
settings. Administrative listeners always stay on loopback. IPv4 and DNS hosts
are supported by this initial renderer; IPv6 requires native configuration.
All enabled local listener ports must differ, including the Knots RPC port.

## CLN / TEOS backend

```sh
mkdir -m 700 "$HOME/.teos"
cp generated/teos/teos.toml "$HOME/.teos/teos.toml"
chmod 600 "$HOME/.teos/teos.toml"
```

Grant this account read access to the configured RPC cookie file.
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

## Optional native LND backend

Skip this section unless `[lnd].enabled = true`. This currently runs the full LND
daemon, including wallet initialization. It is not a standalone watchtower executable.

```sh
mkdir -m 700 "$HOME/.lnd-tower"
cp generated/lnd/lnd.conf "$HOME/.lnd-tower/lnd.conf"
chmod 600 "$HOME/.lnd-tower/lnd.conf"
```

After installing the generated configuration, start:

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
For remote clients, set `[listeners].bind` in the shared source and regenerate.
Allow TCP 9911 only from intended clients. The renderer does not configure
external-address advertisement; give clients the reachable tower address explicitly. This is an encrypted native protocol, not HTTP. Keep RPC/REST private.

A global LND session admission cap has not been established. Restricted access,
storage quotas and monitoring are needed before considering open registration.

## Verify both independently

Check TEOS backend reachability and scanner progress, plus LND readiness and
tower logs. Follow CLIENTS.md and verify client acknowledgements. A listening
port or registration is not proof that a revoked state was stored. Keep both
processes independently restartable. See OPERATIONS.md for recovery guidance.

Disabling `[lnd].enabled` stops generation of its config; it does not delete an
existing output or stop a running backend. Retire an existing deployment separately
only after accounting for its stored sessions and protection obligations.
