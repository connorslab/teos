// Offline identity export for migrating a dedicated LND tower.
package main

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/btcsuite/btcd/btcutil/v2/hdkeychain"
	"github.com/btcsuite/btcd/chaincfg/v2"
	"github.com/lightningnetwork/lnd/aezeed"
)

func run() error {
	seedFile := flag.String("seed-file", "", "Local JSON containing cipher_seed_mnemonic (24 words)")
	passFile := flag.String("seed-passphrase-file", "", "Optional aezeed passphrase file (not wallet unlock password)")
	expected := flag.String("expected-pubkey", "", "Required existing tower public key")
	out := flag.String("out", "", "New tower.key path; must not exist")
	network := flag.String("network", "mainnet", "mainnet or regtest")
	flag.Parse()
	if *seedFile == "" || *out == "" || len(*expected) != 66 {
		return errors.New("seed-file, out and expected-pubkey required")
	}
	b, e := os.ReadFile(*seedFile)
	if e != nil {
		return e
	}
	var v struct {
		Words []string `json:"cipher_seed_mnemonic"`
	}
	if e = json.Unmarshal(b, &v); e != nil {
		return errors.New("invalid seed JSON")
	}
	if len(v.Words) != 24 {
		return errors.New("expected 24 aezeed words")
	}
	var m aezeed.Mnemonic
	copy(m[:], v.Words)
	var pass []byte
	if *passFile != "" {
		pass, e = os.ReadFile(*passFile)
		if e != nil {
			return e
		}
		pass = []byte(strings.TrimRight(string(pass), "\r\n"))
	}
	seed, e := m.ToCipherSeed(pass)
	if e != nil {
		return errors.New("cannot decode aezeed")
	}
	params := &chaincfg.MainNetParams
	coin := uint32(0)
	if *network == "regtest" {
		params = &chaincfg.RegressionNetParams
		coin = 1
	} else if *network != "mainnet" {
		return errors.New("unsupported network")
	}
	k, e := hdkeychain.NewMaster(seed.Entropy[:], params)
	if e != nil {
		return e
	}
	for _, i := range []uint32{1017 + hdkeychain.HardenedKeyStart, coin + hdkeychain.HardenedKeyStart, 9 + hdkeychain.HardenedKeyStart, 0, 0} {
		k, e = k.Derive(i)
		if e != nil {
			return e
		}
	}
	priv, e := k.ECPrivKey()
	if e != nil {
		return e
	}
	pub := hex.EncodeToString(priv.PubKey().SerializeCompressed())
	if pub != *expected {
		return errors.New("derived identity does not match expected tower; no output written")
	}
	f, e := os.OpenFile(*out, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if e != nil {
		return e
	}
	_, e = f.Write(priv.Serialize())
	if e == nil {
		e = f.Sync()
	}
	ce := f.Close()
	if e != nil {
		return e
	}
	if ce != nil {
		return ce
	}
	fmt.Println("Exported verified tower identity:", pub)
	return nil
}
func main() {
	if e := run(); e != nil {
		fmt.Fprintln(os.Stderr, e)
		os.Exit(1)
	}
}
