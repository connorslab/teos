package wire

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"os"
	"testing"
	"time"
)

func TestKnotsHeaderVectors(t *testing.T) {
	raw, err := os.ReadFile("testdata/knots_block_header_v2.json")
	if err != nil {
		t.Fatal(err)
	}
	var vectors struct {
		Headers []struct {
			Name       string `json:"name"`
			Serialized string `json:"serialized"`
			BlockHash  string `json:"block_hash"`
			Fields     struct {
				Time uint32 `json:"nTime"`
			} `json:"fields"`
		} `json:"headers"`
	}
	if err := json.Unmarshal(raw, &vectors); err != nil {
		t.Fatal(err)
	}
	if len(vectors.Headers) == 0 {
		t.Fatal("no vectors")
	}
	for _, v := range vectors.Headers {
		t.Run(v.Name, func(t *testing.T) {
			encoded, err := hex.DecodeString(v.Serialized)
			if err != nil {
				t.Fatal(err)
			}
			var h BlockHeader
			if err := h.Deserialize(bytes.NewReader(encoded)); err != nil {
				t.Fatal(err)
			}
			if got := h.BlockHash().String(); got != v.BlockHash {
				t.Fatalf("hash %s, want %s", got, v.BlockHash)
			}
			if uint32(h.Timestamp.Unix()) != v.Fields.Time {
				t.Fatal("incorrect adjusted time")
			}
			var b bytes.Buffer
			if err := h.Serialize(&b); err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(b.Bytes(), encoded) || h.SerializeSize() != len(encoded) {
				t.Fatal("round trip mismatch")
			}
			for n := 0; n < len(encoded); n++ {
				var truncated BlockHeader
				if truncated.Deserialize(bytes.NewReader(encoded[:n])) == nil {
					t.Fatalf("accepted %d bytes", n)
				}
			}
			block := MsgBlock{Header: h, Transactions: blockOne.Transactions}
			b.Reset()
			if err := block.Serialize(&b); err != nil {
				t.Fatal(err)
			}
			if b.Len() != block.SerializeSize() || b.Len() != block.SerializeSizeStripped() {
				t.Fatal("block size mismatch")
			}
			var decoded MsgBlock
			if err := decoded.Deserialize(&b); err != nil {
				t.Fatal(err)
			}
			if len(decoded.Transactions) != 1 || decoded.BlockHash() != h.BlockHash() {
				t.Fatal("block round trip mismatch")
			}
			legacy := BlockHeader{Version: 1, Timestamp: time.Unix(123, 0)}
			b.Reset()
			if err := legacy.Serialize(&b); err != nil {
				t.Fatal(err)
			}
			if b.Len() != 80 {
				t.Fatal("legacy size changed")
			}
			if err := h.Deserialize(&b); err != nil {
				t.Fatal(err)
			}
			if h.HeaderV2 != (HeaderV2{}) || h.BlockHash() != legacy.BlockHash() {
				t.Fatal("stale extension after legacy decode")
			}
		})
	}
}
