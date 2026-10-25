# Source provenance

This directory is copied from `btcsuite/btcd` commit
`280d4d2db6400136f284f121189d9c2942425a78` (`v0.26.0`), subdirectory `wire`.
Its Go source matches the `wire/v2.0.0` tag; that tag's module metadata differs.
The upstream ISC license is retained in LICENSE.

Local changes: extended header fields and serialization, BLAKE2b block hashing,
dynamic block sizes, headers-message capacity, and regression tests.

`testdata/knots_block_header_v2.json` is copied verbatim from
`bitcoinknots/bitcoin` commit `8c85b1585dac23f964e2dd32045624de7f02aa58`,
`src/test/data/block_header_v2.json`, under Bitcoin Knots' MIT license, retained
as `testdata/KNOTS_LICENSE`.

Hash algorithm reference: the same commit's `src/primitives/block.cpp`;
serialization reference: `src/primitives/block.h`.
