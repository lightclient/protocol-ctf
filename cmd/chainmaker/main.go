package main

import (
	"flag"
	"fmt"
	"math/big"
	"os"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus/ethash"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"
)

func main() {
	chainFilename := flag.String("chain", "chain.rlp", "path to write chain file")
	genesisFilename := flag.String("genesis", "genesis.json", "path to write genesis file")
	flag.Parse()

	// Idea:
	// * programatically define genesis file
	// * write genesis file
	// * sketch out chain maker that can be edited on-demand
	// * write chain to rlp file for import in client

	var (
		gendb   = rawdb.NewMemoryDatabase()
		key, _  = crypto.HexToECDSA("b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291")
		address = crypto.PubkeyToAddress(key.PublicKey)
		aa      = common.Address{0xaa}
		funds   = big.NewInt(1000000000000000)
		alloc   = core.GenesisAlloc{
			address: {Balance: funds},
			aa: {
				Balance: common.Big0,
				Nonce:   1,
				Code: []byte{
					byte(vm.PUSH1),
					0x41,
					byte(vm.PUSH1),
					0x01,
					byte(vm.ADD),
				},
			},
		}
		gspec = &core.Genesis{
			Config:     params.TestChainConfig,
			Alloc:      alloc,
			BaseFee:    big.NewInt(params.InitialBaseFee),
			Difficulty: big.NewInt(1234),
		}
		genesis = gspec.MustCommit(gendb)
	)

	// Build chain.
	blocks, _ := core.GenerateChain(gspec.Config, genesis, ethash.NewFaker(), gendb, 1, func(i int, block *core.BlockGen) {
		tx := types.NewTransaction(
			0,
			aa,
			big.NewInt(0),
			100000,
			block.BaseFee(),
			nil,
		)
		x, _ := types.SignTx(tx, types.HomesteadSigner{}, key)
		block.AddTx(x)
	})
	blocks = append([]*types.Block{genesis}, blocks...)

	// Write to disk.
	err := writeGenesis(gspec, *genesisFilename)
	if err != nil {
		exit(fmt.Errorf("unable to write genesis file: %s", err))
	}
	err = writeChain(blocks, *chainFilename)
	if err != nil {
		exit(fmt.Errorf("unable to write chain to disk: %s", err))
	}

	fmt.Printf("wrote %d blocks to disk", len(blocks))
}

func writeChain(chain []*types.Block, filename string) error {
	w, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer w.Close()
	for _, b := range chain {
		b.EncodeRLP(w)
	}
	return nil
}

func writeGenesis(gspec *core.Genesis, filename string) error {
	w, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer w.Close()
	raw, err := gspec.MarshalJSON()
	if err != nil {
		return err
	}
	_, err = w.Write(raw)
	if err != nil {
		return err
	}
	return nil
}

func exit(msg error) {
	fmt.Fprintf(os.Stderr, "%s", msg)
	os.Exit(1)
}
