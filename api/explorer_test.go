package api

import (
	"testing"
)

// TestIntegrationExplorerGET probes the GET call to /explorer.
func TestIntegrationExplorerGET(t *testing.T) {
	if testing.Short() {
		t.SkipNow()
	}
	st, err := createServerTester("TestIntegrationExplorerGET")
	if err != nil {
		t.Fatal(err)
	}

	var eg ExplorerGET
	err = st.getAPI("/explorer", &eg)
	if err != nil {
		t.Fatal(err)
	}
	if eg.Height != st.server.blockchainHeight {
		t.Error("height not accurately reported by explorer")
	}
	if eg.MinerPayoutCount == 0 {
		t.Error("Miner payout count is incorrect")
	}
}

// TestIntegrationExplorerBlockGET probes the GET call to /explorer/block.
func TestIntegrationExplorerBlockGET(t *testing.T) {
	if testing.Short() {
		t.SkipNow()
	}
	st, err := createServerTester("TestIntegrationExplorerBlockGET")
	if err != nil {
		t.Fatal(err)
	}

	var ebg ExplorerBlockGET
	err = st.getAPI("/explorer/block?height=0", &ebg)
	if err != nil {
		t.Fatal(err)
	}
	if ebg.Block.ID != ebg.Block.RawBlock.ID() {
		t.Error("block id and block do not match up from api call")
	}
	if st.server.cs.GenesisBlock().ID() != ebg.Block.ID {
		t.Error("wrong block returned by /explorer/block?height=0")
	}
}

// TestIntegrationExplorerHashGET probes the GET call to /explorer/hash.
func TestIntegrationExplorerHashGET(t *testing.T) {
	if testing.Short() {
		t.SkipNow()
	}
	st, err := createServerTester("TestIntegrationExplorerHashGET")
	if err != nil {
		t.Fatal(err)
	}

	var ehg ExplorerHashGET
	gb := st.server.cs.GenesisBlock()
	err = st.getAPI("/explorer/hash?hash="+gb.ID().String(), &ehg)
	if err != nil {
		t.Fatal(err)
	}
	if ehg.HashType != "blockid" {
		t.Error("wrong hash type returned when requesting block hash")
	}
	if ehg.Block.ID != gb.ID() {
		t.Error("wrong block type returned")
	}
}