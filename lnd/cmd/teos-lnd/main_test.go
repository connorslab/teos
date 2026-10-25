package main

import (
	"bytes"
	"errors"
	"github.com/btcsuite/btcd/chaincfg/v2"
	"github.com/lightningnetwork/lnd/brontide"
	"github.com/lightningnetwork/lnd/lnwire"
	"github.com/lightningnetwork/lnd/watchtower/wtserver"
	"github.com/lightningnetwork/lnd/watchtower/wtwire"
	"net"
	"path/filepath"
	"testing"
	"time"

	"github.com/btcsuite/btcd/chainhash/v2"
	"github.com/btcsuite/btcd/wire/v2"
	"github.com/lightningnetwork/lnd/chainntnfs"
	"github.com/lightningnetwork/lnd/kvdb"
	"github.com/lightningnetwork/lnd/watchtower/blob"
	"github.com/lightningnetwork/lnd/watchtower/wtdb"
)

type fakeChain struct {
	blocks []*wire.MsgBlock
	all    map[chainhash.Hash]*wire.MsgBlock
	fail   bool
}

func (c *fakeChain) hash(n int32) (*chainhash.Hash, error) {
	if int(n) >= len(c.blocks) {
		return nil, errors.New("height")
	}
	h := c.blocks[n].BlockHash()
	return &h, nil
}
func (c *fakeChain) header(h *chainhash.Hash) (*wire.BlockHeader, error) {
	return &c.all[*h].Header, nil
}
func (c *fakeChain) block(h *chainhash.Hash) (*wire.MsgBlock, error) {
	if c.fail {
		return nil, errors.New("offline")
	}
	return c.all[*h], nil
}
func (c *fakeChain) add(nonce uint32) {
	var prev chainhash.Hash
	if len(c.blocks) > 0 {
		prev = c.blocks[len(c.blocks)-1].BlockHash()
	}
	b := &wire.MsgBlock{Header: wire.BlockHeader{PrevBlock: prev, Nonce: nonce}}
	c.blocks = append(c.blocks, b)
	c.all[b.BlockHash()] = b
}

type fakeDB struct {
	tip     *chainntnfs.BlockEpoch
	fail    bool
	visited []int32
}

func (d *fakeDB) GetLookoutTip() (*chainntnfs.BlockEpoch, error) { return d.tip, nil }
func (d *fakeDB) SetLookoutTip(e *chainntnfs.BlockEpoch) error {
	if d.fail {
		return errors.New("disk full")
	}
	d.tip = e
	d.visited = append(d.visited, e.Height)
	return nil
}
func (d *fakeDB) QueryMatches([]blob.BreachHint) ([]wtdb.Match, error) { return nil, nil }
func TestCatchupOutageAndDeepReorg(t *testing.T) {
	c := &fakeChain{all: make(map[chainhash.Hash]*wire.MsgBlock)}
	for i := 0; i < 5; i++ {
		c.add(uint32(i))
	}
	h, _ := c.hash(0)
	d := &fakeDB{tip: &chainntnfs.BlockEpoch{Height: 0, Hash: h}}
	s := scanner{chain: c, db: d}
	c.fail = true
	if _, e := s.step(4); e == nil || d.tip.Height != 0 {
		t.Fatal("failed block must not advance")
	}
	c.fail = false
	for i := 0; i < 4; i++ {
		if _, e := s.step(4); e != nil {
			t.Fatal(e)
		}
	}
	if d.tip.Height != 4 {
		t.Fatal("catchup")
	}
	c.blocks = c.blocks[:2]
	for i := 10; i < 13; i++ {
		c.add(uint32(i))
	}
	if _, e := s.step(4); e != nil {
		t.Fatal(e)
	}
	if d.tip.Height != 2 {
		t.Fatal("must replay from common ancestor")
	}
	for d.tip.Height < 4 {
		if _, e := s.step(4); e != nil {
			t.Fatal(e)
		}
	}
	want, _ := c.hash(4)
	if *want != *d.tip.Hash {
		t.Fatal("wrong fork")
	}
}
func TestCursorWriteFailure(t *testing.T) {
	c := &fakeChain{all: make(map[chainhash.Hash]*wire.MsgBlock)}
	c.add(0)
	c.add(1)
	h, _ := c.hash(0)
	d := &fakeDB{tip: &chainntnfs.BlockEpoch{Height: 0, Hash: h}, fail: true}
	s := scanner{chain: c, db: d}
	if _, e := s.step(1); e == nil || d.tip.Height != 0 {
		t.Fatal("database failure ignored")
	}
}
func TestIdentityStableAndMissingFails(t *testing.T) {
	p := filepath.Join(t.TempDir(), "tower.key")
	if _, e := identity(p, false); e == nil {
		t.Fatal("missing identity silently replaced")
	}
	a, e := identity(p, true)
	if e != nil {
		t.Fatal(e)
	}
	b, e := identity(p, false)
	if e != nil || !a.PubKey().IsEqual(b.PubKey()) {
		t.Fatal("identity changed")
	}
}
func TestDurableRemedyAndCursorRestart(t *testing.T) {
	dir := t.TempDir()
	open := func() kvdb.Backend {
		b, e := kvdb.GetBoltBackend(&kvdb.BoltBackendConfig{DBPath: dir, DBFileName: "watchtower.db", DBTimeout: time.Second})
		if e != nil {
			t.Fatal(e)
		}
		return b
	}
	b := open()
	d, e := wtdb.OpenTowerDB(b)
	if e != nil {
		t.Fatal(e)
	}
	h := chainhash.Hash{1}
	if e = d.SetLookoutTip(&chainntnfs.BlockEpoch{Height: 42, Hash: &h}); e != nil {
		t.Fatal(e)
	}
	tx := wire.NewMsgTx(2)
	if e = enqueue(b, tx); e != nil {
		t.Fatal(e)
	}
	b.Close()
	b = open()
	defer b.Close()
	d, e = wtdb.OpenTowerDB(b)
	if e != nil {
		t.Fatal(e)
	}
	tip, e := d.GetLookoutTip()
	if e != nil || tip.Height != 42 {
		t.Fatal("cursor lost")
	}
	count := 0
	e = kvdb.View(b, func(t kvdb.RTx) error {
		return t.ReadBucket(pendingBucket).ForEach(func(_, v []byte) error { count++; return nil })
	}, func() { count = 0 })
	if e != nil || count != 1 {
		t.Fatal("remedy lost")
	}
}

func TestNativeEnrollmentAndAcknowledgement(t *testing.T) {
	dir := t.TempDir()
	key, e := identity(filepath.Join(dir, "tower.key"), true)
	if e != nil {
		t.Fatal(e)
	}
	client, e := identity(filepath.Join(dir, "client.key"), true)
	if e != nil {
		t.Fatal(e)
	}
	backend, e := kvdb.GetBoltBackend(&kvdb.BoltBackendConfig{DBPath: dir, DBFileName: "watchtower.db", DBTimeout: time.Second})
	if e != nil {
		t.Fatal(e)
	}
	defer backend.Close()
	db, e := wtdb.OpenTowerDB(backend)
	if e != nil {
		t.Fatal(e)
	}
	listener, e := brontide.NewListener(key, "127.0.0.1:0", brontide.DisabledBanClosure)
	if e != nil {
		t.Fatal(e)
	}
	server, e := wtserver.New(&wtserver.Config{DB: db, NodeKeyECDH: key, Listeners: []net.Listener{listener}, ChainHash: *chaincfg.RegressionNetParams.GenesisHash, DisableReward: true, ReadTimeout: time.Second * 5, WriteTimeout: time.Second * 5})
	if e != nil {
		t.Fatal(e)
	}
	if e = server.Start(); e != nil {
		t.Fatal(e)
	}
	defer server.Stop()
	conn, e := brontide.Dial(client, &lnwire.NetAddress{IdentityKey: key.PubKey(), Address: listener.Addr()}, 5*time.Second, net.DialTimeout)
	if e != nil {
		t.Fatal(e)
	}
	defer conn.Close()
	conn.SetDeadline(time.Now().Add(10 * time.Second))
	exchange := func(msg wtwire.Message) wtwire.Message {
		var b bytes.Buffer
		if _, e := wtwire.WriteMessage(&b, msg, 0); e != nil {
			t.Fatal(e)
		}
		if _, e := conn.Write(b.Bytes()); e != nil {
			t.Fatal(e)
		}
		raw, e := conn.ReadNextMessage()
		if e != nil {
			t.Fatal(e)
		}
		reply, e := wtwire.ReadMessage(bytes.NewReader(raw), 0)
		if e != nil {
			t.Fatal(e)
		}
		return reply
	}
	init := exchange(wtwire.NewInitMessage(lnwire.NewRawFeatureVector(), *chaincfg.RegressionNetParams.GenesisHash)).(*wtwire.Init)
	if !init.ConnFeatures.IsSet(wtwire.Blake2bRequired) {
		t.Fatal("missing XBT handshake feature")
	}
	reply := exchange(&wtwire.CreateSession{BlobType: blob.TypeAltruistCommit, MaxUpdates: 10, SweepFeeRate: 2500}).(*wtwire.CreateSessionReply)
	if reply.Code != 0 {
		t.Fatalf("session rejected: %v", reply.Code)
	}
	conn.Close()
	time.Sleep(20 * time.Millisecond) // CreateSession is a one-request connection.
	conn, e = brontide.Dial(client, &lnwire.NetAddress{IdentityKey: key.PubKey(), Address: listener.Addr()}, 5*time.Second, net.DialTimeout)
	if e != nil {
		t.Fatal(e)
	}
	defer conn.Close()
	conn.SetDeadline(time.Now().Add(10 * time.Second))
	exchange(wtwire.NewInitMessage(lnwire.NewRawFeatureVector(), *chaincfg.RegressionNetParams.GenesisHash))
	hint := blob.BreachHint{1}
	ciphertext := make([]byte, blob.NonceSize+blob.V0PlaintextSize+blob.CiphertextExpansion)
	ack := exchange(&wtwire.StateUpdate{SeqNum: 1, Hint: hint, EncryptedBlob: ciphertext}).(*wtwire.StateUpdateReply)
	if ack.Code != 0 || ack.LastApplied != 1 {
		t.Fatalf("no acknowledgement: %+v", ack)
	}
	matches, e := db.QueryMatches([]blob.BreachHint{hint})
	if e != nil || len(matches) != 1 {
		t.Fatal("acknowledged update not durable")
	}
}
