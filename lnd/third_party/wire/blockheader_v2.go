// Copyright (c) 2026 The btcsuite developers
// Use of this source code is governed by an ISC license in LICENSE.

package wire

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"

	"github.com/btcsuite/btcd/chainhash/v2"
	"golang.org/x/crypto/blake2b"
)

// HeaderV2 is the 84-byte extension specified by Bitcoin Knots.
// Fixed arrays contain serialized bytes, not display-order hash strings.
type HeaderV2 struct {
	Nonce2              uint32
	Nonce3              uint32
	ExtraNonce          [16]byte
	TimeOffset          uint32
	TxCount             uint16
	Flags               uint8
	XORKeyMaskClearBits uint8
	XORKey              [16]byte
	Height              int32
	MergeMiningRHS      [32]byte
}

// IsHeaderV2 reports whether the wire version selects the extended header.
func (h *BlockHeader) IsHeaderV2() bool {
	return uint32(h.Version)&0x80000000 != 0
}

// SerializeSize returns this header's encoded length.
func (h *BlockHeader) SerializeSize() int {
	if h.IsHeaderV2() {
		return MaxBlockHeaderPayload
	}
	return blockHeaderLen
}

func (h *BlockHeader) timeOnWire() uint32 {
	t := uint32(h.Timestamp.Unix())
	if h.IsHeaderV2() && h.HeaderV2.Flags&4 != 0 {
		t -= h.HeaderV2.TimeOffset
	}
	return t
}

func headerTaggedHash(tag string, fields ...interface{}) [32]byte {
	t := sha256.Sum256([]byte(tag))
	var b bytes.Buffer
	b.Write(t[:])
	b.Write(t[:])
	for _, field := range fields {
		// All callers supply fixed-size integers or byte arrays. bytes.Buffer
		// cannot fail, and unsupported fields are programming errors.
		if err := binary.Write(&b, binary.LittleEndian, field); err != nil {
			panic(err)
		}
	}
	return sha256.Sum256(b.Bytes())
}

func (h *BlockHeader) blake2bHash() chainhash.Hash {
	v := &h.HeaderV2
	keyHash := headerTaggedHash("Bitcoin block hash PoW XOR key", v.XORKey)
	var mask [32]byte
	if v.XORKey != [16]byte{} {
		mask = headerTaggedHash("Bitcoin block hash PoW XOR mask", v.XORKey)
		n := int(v.XORKeyMaskClearBits) / 8
		clear(mask[:n])
		mask[n] &= 0xff >> (v.XORKeyMaskClearBits % 8)
	}
	var prev [32]byte
	for i := range prev {
		prev[i] = h.PrevBlock[31-i]
	}
	hidden := headerTaggedHash("Bitcoin prevblock header, hashed", prev)
	h1 := headerTaggedHash("Bitcoin block header 1", uint32(h.Version), prev,
		v.Height, h.MerkleRoot, h.timeOnWire(), uint8(0), h.Bits,
		uint32(v.TxCount), v.Flags, v.XORKeyMaskClearBits, keyHash)
	h2 := headerTaggedHash("Merge-mining hook", h1, [32]byte{}, v.MergeMiningRHS)
	var b bytes.Buffer
	put := func(fields ...interface{}) {
		for _, field := range fields {
			if err := binary.Write(&b, binary.LittleEndian, field); err != nil {
				panic(err)
			}
		}
	}
	put(uint32(0), h2, v.ExtraNonce)
	hash := blake2b.Sum256(b.Bytes())
	b.Reset()
	switch v.Flags & 3 {
	case 3:
		put([32]byte{})
		fallthrough
	case 2:
		put([48]byte{}, h2, h.Nonce, v.Nonce2, v.TimeOffset, v.Nonce3, hash)
	case 0:
		clear(hidden[:6])
		put(hidden, h.Nonce, v.Nonce2, v.TimeOffset, v.Nonce3, hash)
	case 1:
		put(h.Nonce, v.Nonce2, v.Nonce3, v.TimeOffset, hash, h2)
	}
	hash = blake2b.Sum256(b.Bytes())
	var result chainhash.Hash
	for i := range hash {
		result[31-i] = hash[i] ^ mask[i]
	}
	return result
}
