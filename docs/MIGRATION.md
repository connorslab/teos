# Migrate a dedicated LND tower

Migrate only a dedicated tower deployment. Never copy, restore or modify a live
Lightning channel database as part of this procedure. Keep the previous deployment
available until the replacement has passed readiness and client checks.

1. Record the existing tower public key, native listener, chain and client sessions.
   Back up the tower identity recovery material and tower database securely.
2. Build `teos-lnd` and `teos-lnd-migrate`. Export the dedicated tower identity
   **offline**, on a trusted machine, using its existing aezeed recovery words:

   Create the empty destination directory with private permissions first.

   ```sh
   install -d -m 700 /private/new-tower
   bin/teos-lnd-migrate --seed-file /private/tower-seed.json \
     --network mainnet --expected-pubkey EXISTING_TOWER_PUBLIC_KEY \
     --out /private/new-tower/tower.key
   ```

   The JSON must contain `cipher_seed_mnemonic`, an array of 24 words. If the seed
   uses an aezeed passphrase, pass `--seed-passphrase-file`; this is different from
   the wallet unlock password. Never put seed material or passwords in command
   arguments. The helper derives `m/1017'/coin_type'/9'/0/0`, verifies the public
   key, creates a mode-0600 file and refuses to overwrite an existing output.
   It does not print private keys. A mismatch stops migration.
3. Stop only the old tower. Make a coherent copy of its `watchtower.db` into the
   new data directory beside `tower.key`, owned by the new backend account.
   Do not copy `channel.db` or initialize a replacement identity.
4. Configure the same network, native port and RPC access. If the old tower has
   no persisted scan cursor, set `start_height` in the native JSON to a height
   **before** the first protected state. This means the last processed block;
   scanning begins at the following height. Never substitute today's tip.
   An explicit earlier height also rewinds an existing cursor. Remove the override
   after verified catch-up to avoid rescanning on every restart.
5. Start `teos-lnd`. Require fresh healthy `status.json`, the original tower public
   key, preserved session/update counts, complete catch-up and client connectivity.
   Verify acknowledgements when clients naturally produce new revoked states.
6. Update monitoring and backups to include the new executable, key, database and
   configuration. Disable the obsolete full-daemon startup only after validation.

The backend retains LND's tower database schema and adds a durable recovery queue.
An older daemon ignores that queue. A binary rollback is therefore not a complete
recovery procedure: preserve the newest database and pending remedies, and never
restore a stale snapshot over newly acknowledged client updates.

Do not run old and new backends against the same database or native listener.
Seed files remain sensitive even after migration; keep them out of repositories,
logs and ordinary backups. A fresh standalone deployment needs only `--init` and
never needs an LND seed or wallet.
