package blockcache

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"os"
	"testing"

	"github.com/btcsuite/btcd/chainhash/v2"
	"github.com/btcsuite/btcd/wire/v2"
	"github.com/stretchr/testify/require"
)

// Exercise the actual LND cache, including the btcutil/neutrino wrappers,
// against Knots' independently generated header hashes.
func TestBlake2bBlockCache(t *testing.T) {
	data, err := os.ReadFile("../third_party/wire/testdata/knots_block_header_v2.json")
	require.NoError(t, err)
	var vectors struct {
		Headers []struct {
			Serialized string `json:"serialized"`
			Hash       string `json:"block_hash"`
		} `json:"headers"`
	}
	require.NoError(t, json.Unmarshal(data, &vectors))
	require.NotEmpty(t, vectors.Headers)
	cache := NewBlockCache(100000)
	for _, v := range vectors.Headers {
		encoded, err := hex.DecodeString(v.Serialized)
		require.NoError(t, err)
		var block wire.MsgBlock
		require.NoError(t, block.Header.Deserialize(bytes.NewReader(encoded)))
		hash, err := chainhash.NewHashFromStr(v.Hash)
		require.NoError(t, err)
		calls := 0
		fetch := func(requested *chainhash.Hash) (*wire.MsgBlock, error) {
			calls++
			require.Equal(t, *hash, *requested)
			return &block, nil
		}
		for i := 0; i < 2; i++ {
			got, err := cache.GetBlock(hash, fetch)
			require.NoError(t, err)
			require.Equal(t, *hash, got.BlockHash())
			require.Equal(t, block.Header, got.Header)
		}
		require.Equal(t, 1, calls)
	}
}
