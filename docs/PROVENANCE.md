# Provenance and licensing

The repository starts with a source snapshot, without private operational history.
Original notices remain in source and license files.

- TEOS base: `talaia-labs/rust-teos` revision
  `be344ecc5286dd9436bf343d30954135da8ad4ac` (MIT).
- LND base: `connorslab/lnd-blake2b` revision
  `5119a19ef439106d96d5fe1055c1dd0c4ee8c46b` (MIT).
- The XBT watchtower handshake advertises/requires feature bits 512/513, adapted
  from Lightning Fork revision `eb0848bad4abdfdbe90b4970be1e4511461fc553`.
  This snapshot preserves its LND base's v2 module imports.
- `vendor/bitcoin`: bitcoin 0.32.9 with extended XBT headers and BLAKE2b block
  hashing; original CC0 notices retained.
- `vendor/lightning-block-sync-0.1.0`: upstream 0.1.0 with full-header parsing,
  XBT activation validation and ordinary chain validation retained; MIT/Apache-2.0.

The Rust changes add XBT fixtures, bounded registration/renewal, durable
pending-broadcast handling, fee-policy retry, and fail-stop storage behavior that
prevents advancing the saved scanner cursor after a tracker write failure.
Cargo.lock and Go lockfiles record transitive dependencies.
Upstream attribution does not imply upstream endorsement.
