// teos-lnd runs the native LND watchtower protocol without a Lightning wallet.
package main

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/btcsuite/btcd/chaincfg/v2"
	"github.com/btcsuite/btcd/chainhash/v2"
	"github.com/btcsuite/btcd/wire/v2"
	"github.com/lightningnetwork/lnd/brontide"
	"github.com/lightningnetwork/lnd/chainntnfs"
	"github.com/lightningnetwork/lnd/keychain"
	"github.com/lightningnetwork/lnd/kvdb"
	"github.com/lightningnetwork/lnd/watchtower/blob"
	"github.com/lightningnetwork/lnd/watchtower/lookout"
	"github.com/lightningnetwork/lnd/watchtower/wtdb"
	"github.com/lightningnetwork/lnd/watchtower/wtserver"
)

type config struct {
	Generated string `json:"_generated,omitempty"`
	Network   string `json:"network"`
	RPCURL    string `json:"rpc_url"`
	Cookie    string `json:"cookie"`
	Listen    string `json:"listen"`
	DataDir   string `json:"data_dir"`
	// StartHeight is required for importing a database without a scan cursor.
	StartHeight *int32 `json:"start_height,omitempty"`
}

type rpcError struct {
	Code    int
	Message string
}

func (e *rpcError) Error() string { return fmt.Sprintf("RPC error %d: %s", e.Code, e.Message) }

type rpc struct {
	url, cookie string
	client      *http.Client
	ctx         context.Context
}

func (r *rpc) call(method string, out any, args ...any) error {
	cookie, err := os.ReadFile(r.cookie)
	if err != nil {
		return err
	}
	user, pass, ok := strings.Cut(strings.TrimSpace(string(cookie)), ":")
	if !ok {
		return errors.New("invalid RPC cookie")
	}
	if args == nil {
		args = []any{}
	}
	body, _ := json.Marshal(map[string]any{"jsonrpc": "1.0", "id": "tower", "method": method, "params": args})
	req, err := http.NewRequestWithContext(r.ctx, "POST", r.url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.SetBasicAuth(user, pass)
	req.Header.Set("Content-Type", "application/json")
	res, err := r.client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	var v struct {
		Result json.RawMessage
		Error  *rpcError
	}
	if err = json.NewDecoder(io.LimitReader(res.Body, 64<<20)).Decode(&v); err != nil {
		return fmt.Errorf("invalid RPC response (HTTP %d)", res.StatusCode)
	}
	if v.Error != nil {
		return v.Error
	}
	if res.StatusCode != 200 {
		return fmt.Errorf("RPC HTTP %d", res.StatusCode)
	}
	return json.Unmarshal(v.Result, out)
}

type chainInfo struct {
	Chain                string
	Blocks               int32
	BestBlockHash        string
	InitialBlockDownload bool
}

func (r *rpc) info() (chainInfo, error) {
	var v chainInfo
	e := r.call("getblockchaininfo", &v)
	return v, e
}
func (r *rpc) hash(height int32) (*chainhash.Hash, error) {
	var s string
	if e := r.call("getblockhash", &s, height); e != nil {
		return nil, e
	}
	return chainhash.NewHashFromStr(s)
}
func (r *rpc) header(hash *chainhash.Hash) (*wire.BlockHeader, error) {
	var s string
	if e := r.call("getblockheader", &s, hash.String(), false); e != nil {
		return nil, e
	}
	b, e := hex.DecodeString(s)
	if e != nil {
		return nil, e
	}
	h := new(wire.BlockHeader)
	if e = h.Deserialize(bytes.NewReader(b)); e != nil {
		return nil, e
	}
	if h.BlockHash() != *hash {
		return nil, errors.New("header hash mismatch: wrong chain/hash implementation")
	}
	return h, nil
}
func (r *rpc) block(hash *chainhash.Hash) (*wire.MsgBlock, error) {
	var s string
	if e := r.call("getblock", &s, hash.String(), 0); e != nil {
		return nil, e
	}
	b, e := hex.DecodeString(s)
	if e != nil {
		return nil, e
	}
	v := new(wire.MsgBlock)
	if e = v.Deserialize(bytes.NewReader(b)); e != nil {
		return nil, e
	}
	if v.BlockHash() != *hash {
		return nil, errors.New("block hash mismatch")
	}
	return v, nil
}

type chain interface {
	hash(int32) (*chainhash.Hash, error)
	header(*chainhash.Hash) (*wire.BlockHeader, error)
	block(*chainhash.Hash) (*wire.MsgBlock, error)
}
type towerDB interface {
	GetLookoutTip() (*chainntnfs.BlockEpoch, error)
	SetLookoutTip(*chainntnfs.BlockEpoch) error
	QueryMatches([]blob.BreachHint) ([]wtdb.Match, error)
}
type scanner struct {
	chain   chain
	db      towerDB
	enqueue func(*wire.MsgTx) error
}

// step never advances past an unscanned block or a non-durable remedy.
func (s *scanner) step(best int32) (bool, error) {
	tip, err := s.db.GetLookoutTip()
	if err != nil {
		return false, err
	}
	if tip == nil {
		return false, errors.New("missing scan cursor")
	}
	// Walk orphan headers back to the common ancestor, including deeper reorgs.
	for {
		if tip.Height <= best {
			h, e := s.chain.hash(tip.Height)
			if e != nil {
				return false, e
			}
			if *h == *tip.Hash {
				break
			}
		}
		if tip.Height == 0 {
			return false, errors.New("genesis mismatch")
		}
		h, e := s.chain.header(tip.Hash)
		if e != nil {
			return false, e
		}
		tip = &chainntnfs.BlockEpoch{Height: tip.Height - 1, Hash: &h.PrevBlock}
	}
	if err = s.db.SetLookoutTip(tip); err != nil {
		return false, err
	}
	if tip.Height >= best {
		return false, nil
	}
	height := tip.Height + 1
	hash, err := s.chain.hash(height)
	if err != nil {
		return false, err
	}
	block, err := s.chain.block(hash)
	if err != nil {
		return false, err
	}
	if block.Header.PrevBlock != *tip.Hash {
		return false, errors.New("chain changed during scan; retrying")
	}
	hints := make([]blob.BreachHint, 0, len(block.Transactions))
	txs := make(map[blob.BreachHint]*wire.MsgTx)
	for _, tx := range block.Transactions {
		h := tx.TxHash()
		hint := blob.NewBreachHintFromHash(&h)
		hints = append(hints, hint)
		txs[hint] = tx
	}
	matches, err := s.db.QueryMatches(hints)
	if err != nil {
		return false, err
	}
	for _, m := range matches {
		tx := txs[m.Hint]
		if tx == nil {
			return false, errors.New("database returned unknown hint")
		}
		h := tx.TxHash()
		kit, e := blob.Decrypt(blob.NewBreachKeyFromHash(&h), m.EncryptedBlob, m.SessionInfo.Policy.BlobType)
		if e != nil {
			continue
		} // Hint collisions and invalid client ciphertext are not breaches.
		desc := lookout.JusticeDescriptor{BreachedCommitTx: tx, SessionInfo: m.SessionInfo, JusticeKit: kit}
		remedy, e := desc.CreateJusticeTxn()
		if e != nil {
			return false, e
		}
		if e = s.enqueue(remedy); e != nil {
			return false, e
		}
	}
	if err = s.db.SetLookoutTip(&chainntnfs.BlockEpoch{Height: height, Hash: hash}); err != nil {
		return false, err
	}
	return true, nil
}

var pendingBucket = []byte("teos-lnd-remedies-v1")

func enqueue(db kvdb.Backend, tx *wire.MsgTx) error {
	var b bytes.Buffer
	if e := tx.Serialize(&b); e != nil {
		return e
	}
	h := tx.TxHash()
	return kvdb.Update(db, func(t kvdb.RwTx) error {
		v, e := t.CreateTopLevelBucket(pendingBucket)
		if e != nil {
			return e
		}
		return v.Put(h[:], b.Bytes())
	}, func() {})
}

// Keep remedies and rebroadcast them after restarts/reorgs. No automatic pruning.
func broadcast(db kvdb.Backend, r *rpc) error {
	var raws [][]byte
	err := kvdb.View(db, func(t kvdb.RTx) error {
		b := t.ReadBucket(pendingBucket)
		if b == nil {
			return nil
		}
		return b.ForEach(func(_, v []byte) error { raws = append(raws, bytes.Clone(v)); return nil })
	}, func() { raws = nil })
	if err != nil {
		return err
	}
	var failures []error
	for _, raw := range raws {
		var id string
		err = r.call("sendrawtransaction", &id, hex.EncodeToString(raw))
		if err != nil {
			var e *rpcError
			if errors.As(err, &e) && e.Code == -27 {
				continue
			}
			failures = append(failures, err)
		}
	}
	return errors.Join(failures...)
}
func counts(db kvdb.Backend) (map[string]int, error) {
	result := map[string]int{}
	err := kvdb.View(db, func(t kvdb.RTx) error {
		for _, entry := range []struct{ name, label string }{{"sessions-bucket", "sessions"}, {"updates-bucket", "updates"}, {string(pendingBucket), "retained_remedies"}} {
			b := t.ReadBucket([]byte(entry.name))
			if b == nil {
				continue
			}
			if e := b.ForEach(func(k, v []byte) error {
				if v != nil {
					result[entry.label]++
					return nil
				}
				nested := b.NestedReadBucket(k)
				return nested.ForEach(func(_, v []byte) error {
					if v != nil {
						result[entry.label]++
					}
					return nil
				})
			}); e != nil {
				return e
			}
		}
		return nil
	}, func() { result = map[string]int{} })
	return result, err
}
func identity(path string, create bool) (*keychain.PrivKeyECDH, error) {
	b, err := os.ReadFile(path)
	if os.IsNotExist(err) && create {
		k, e := btcec.NewPrivateKey()
		if e != nil {
			return nil, e
		}
		f, e := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
		if e != nil {
			return nil, e
		}
		b = k.Serialize()
		_, e = f.Write(b)
		if e == nil {
			e = f.Sync()
		}
		ce := f.Close()
		if e != nil {
			return nil, e
		}
		if ce != nil {
			return nil, ce
		}
	} else if err != nil {
		return nil, err
	}
	st, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	if st.Mode().Perm()&0077 != 0 {
		return nil, errors.New("identity file must have mode 0600")
	}
	if len(b) != 32 {
		return nil, errors.New("invalid identity length")
	}
	k, _ := btcec.PrivKeyFromBytes(b)
	if k.Key.IsZero() || !bytes.Equal(k.Serialize(), b) {
		return nil, errors.New("invalid identity scalar")
	}
	return &keychain.PrivKeyECDH{PrivKey: k}, nil
}
func atomicJSON(path string, v any) error {
	b, e := json.MarshalIndent(v, "", "  ")
	if e != nil {
		return e
	}
	f, e := os.CreateTemp(filepath.Dir(path), ".status-")
	if e != nil {
		return e
	}
	defer os.Remove(f.Name())
	if _, e = f.Write(b); e == nil {
		e = f.Sync()
	}
	ce := f.Close()
	if e != nil {
		return e
	}
	if ce != nil {
		return ce
	}
	return os.Rename(f.Name(), path)
}

func run() error {
	path := flag.String("config", "tower.json", "Configuration file")
	initKey := flag.Bool("init", false, "Initialize a new tower identity; never use for migration")
	flag.Parse()
	f, err := os.Open(*path)
	if err != nil {
		return err
	}
	var cfg config
	dec := json.NewDecoder(f)
	dec.DisallowUnknownFields()
	err = dec.Decode(&cfg)
	f.Close()
	if err != nil {
		return err
	}
	if cfg.Network != "mainnet" && cfg.Network != "regtest" {
		return errors.New("network must be mainnet or regtest")
	}
	if !filepath.IsAbs(cfg.DataDir) || cfg.Cookie == "" || cfg.RPCURL == "" || cfg.Listen == "" {
		return errors.New("data_dir must be absolute; cookie, rpc_url and listen required")
	}
	if err = os.MkdirAll(cfg.DataDir, 0700); err != nil {
		return err
	}
	dbPath := filepath.Join(cfg.DataDir, "watchtower.db")
	if *initKey {
		if _, e := os.Stat(dbPath); !os.IsNotExist(e) {
			return errors.New("refusing identity initialization with existing database")
		}
	}
	key, err := identity(filepath.Join(cfg.DataDir, "tower.key"), *initKey)
	if err != nil {
		return err
	}
	if *initKey {
		fmt.Printf("Tower identity: %x\n", key.PubKey().SerializeCompressed())
		return nil
	}
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer stop()
	r := &rpc{cfg.RPCURL, cfg.Cookie, &http.Client{Timeout: 20 * time.Second}, ctx}
	info, err := r.info()
	if err != nil {
		return err
	}
	expected := "main"
	params := &chaincfg.MainNetParams
	if cfg.Network == "regtest" {
		expected = "regtest"
		params = &chaincfg.RegressionNetParams
	}
	if info.Chain != expected || info.InitialBlockDownload {
		return errors.New("Knots is on wrong network or still syncing")
	}
	// Genesis alone cannot distinguish XBT. Validate the current header locally.
	best, err := chainhash.NewHashFromStr(info.BestBlockHash)
	if err != nil {
		return err
	}
	header, err := r.header(best)
	if err != nil {
		return err
	}
	if cfg.Network == "mainnet" && !header.IsHeaderV2() {
		return errors.New("backend lacks XBT BLAKE2b headers")
	}
	_, statErr := os.Stat(dbPath)
	fresh := os.IsNotExist(statErr)
	backend, err := kvdb.GetBoltBackend(&kvdb.BoltBackendConfig{DBPath: cfg.DataDir, DBFileName: "watchtower.db", DBTimeout: 5 * time.Second})
	if err != nil {
		return err
	}
	defer backend.Close()
	db, err := wtdb.OpenTowerDB(backend)
	if err != nil {
		return err
	}
	tip, err := db.GetLookoutTip()
	if err != nil {
		return err
	}
	if tip != nil && cfg.StartHeight != nil && *cfg.StartHeight < tip.Height {
		if *cfg.StartHeight < 0 {
			return errors.New("invalid start_height")
		}
		h, e := r.hash(*cfg.StartHeight)
		if e != nil {
			return e
		}
		if e = db.SetLookoutTip(&chainntnfs.BlockEpoch{Height: *cfg.StartHeight, Hash: h}); e != nil {
			return e
		}
		tip, e = db.GetLookoutTip()
		if e != nil {
			return e
		}
	}
	if tip == nil {
		height := info.Blocks
		if cfg.StartHeight != nil {
			height = *cfg.StartHeight
		} else if !fresh {
			return errors.New("imported database has no cursor; supply start_height at or before first protected state")
		}
		if height < 0 || height > info.Blocks {
			return errors.New("invalid start_height")
		}
		h, e := r.hash(height)
		if e != nil {
			return e
		}
		if e = db.SetLookoutTip(&chainntnfs.BlockEpoch{Height: height, Hash: h}); e != nil {
			return e
		}
	}
	listener, err := brontide.NewListener(key, cfg.Listen, brontide.DisabledBanClosure)
	if err != nil {
		return err
	}
	defer listener.Close()
	server, err := wtserver.New(&wtserver.Config{DB: db, NodeKeyECDH: key, Listeners: []net.Listener{listener}, ChainHash: *params.GenesisHash, DisableReward: true, ReadTimeout: 15 * time.Second, WriteTimeout: 15 * time.Second})
	if err != nil {
		return err
	}
	if err = server.Start(); err != nil {
		return err
	}
	defer server.Stop()
	scan := scanner{r, db, func(tx *wire.MsgTx) error { return enqueue(backend, tx) }}
	log.Printf("native tower listening on %s, identity %x", listener.Addr(), key.PubKey().SerializeCompressed())
	for ctx.Err() == nil {
		info, err = r.info()
		if err == nil && (info.Chain != expected || info.InitialBlockDownload) {
			err = errors.New("chain backend not ready")
		}
		if err == nil {
			for i := 0; i < 100; i++ {
				var more bool
				more, err = scan.step(info.Blocks)
				if err != nil || !more {
					break
				}
			}
		}
		if err == nil {
			err = broadcast(backend, r)
		}
		tip, e := db.GetLookoutTip()
		if e != nil {
			return e
		}
		detail := ""
		if err != nil {
			detail = err.Error()
			log.Printf("not ready: %s", detail)
		}
		totals, e := counts(backend)
		if e != nil {
			return e
		}
		status := map[string]any{"storage": totals, "updated": time.Now().Unix(), "ready": err == nil && tip.Height == info.Blocks, "network": cfg.Network, "identity_pubkey": fmt.Sprintf("%x", key.PubKey().SerializeCompressed()), "scan_height": tip.Height, "scan_hash": tip.Hash.String(), "chain_height": info.Blocks, "listeners": []string{listener.Addr().String()}, "error": detail}
		if e = atomicJSON(filepath.Join(cfg.DataDir, "status.json"), status); e != nil {
			return e
		}
		if err == nil && tip.Height < info.Blocks {
			continue
		}
		select {
		case <-ctx.Done():
		case <-time.After(5 * time.Second):
		}
	}
	return nil
}
func main() {
	if err := run(); err != nil {
		log.Print(err)
		os.Exit(1)
	}
}
