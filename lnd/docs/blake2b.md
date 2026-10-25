# Bitcoin Knots BLAKE2b headers

This branch adds Bitcoin Knots header-v2 decoding, encoding and block identifiers
to LND through a local replacement of `github.com/btcsuite/btcd/wire/v2`.
Use the `bitcoind` backend connected to a Bitcoin Knots node which validates the
chain. Header-v2 is selected by version bit 31; historical 80-byte headers retain
SHA256d, while extended headers occupy 164 bytes.

The implementation follows Bitcoin Knots commit
`8c85b1585dac23f964e2dd32045624de7f02aa58`, specifically
`src/primitives/block.h` and `src/primitives/block.cpp`. It implements all four
ASIC profiles, tagged SHA256 commitments, both BLAKE2b-256 stages, XOR masking,
hash byte order and wrapping timestamp offsets. Transaction hashing is unchanged.

## Build and test

Clone this repository's `lnd-blake2b` branch, then run from its root:

```sh
go build -o ./bin/ -tags="peersrpc signrpc walletrpc chainrpc invoicesrpc" ./cmd/lnd ./cmd/lncli
go test ./blockcache ./chainntnfs ./chainntnfs/bitcoindnotify ./lnwallet/btcwallet ./peer
cd third_party/wire
go test ./...
```

The local `replace` in `go.mod` is required. Build from the checkout; a versioned
`go install` does not support this local replacement. Existing LND RPC and wallet
configuration instructions still apply. Configure `bitcoin.node=bitcoind` and
point its RPC and ZMQ settings to the intended Knots node (or use LND's supported
RPC polling mode). Match the network settings to that node.

## Scope

This is header compatibility through the full-node backend, not an implementation
of all Knots consensus rules in btcd. Knots remains responsible for activation,
difficulty, transaction and script validation. Neutrino header storage and SPV
validation have not been adapted for this chain; do not use Neutrino for it.
An unmodified btcd backend cannot validate the BLAKE2b chain either.

Tests cover Knots' five reference vectors, all truncated header lengths,
historical decoding after an extended header, full block serialization, and
LND's block cache through btcutil/neutrino cache wrappers. Peer ping payloads
retain their actual serialized header length. End-to-end Knots synchronization,
reorganizations and channel funding/payment/closing still require validation
against a running Knots test network. This change makes no claim of complete
Lightning compatibility or transaction replay protection.

LNDK delegates chain handling to LND. See the separate
[lndk-blake2b repository](https://github.com/connorslab/lndk-blake2b)
and its `docs/blake2b.md` for its integration boundary.
