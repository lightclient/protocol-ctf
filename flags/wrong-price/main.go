package main

import (
	"compress/gzip"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus/ethash"
	"github.com/ethereum/go-ethereum/console"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/eth"
	"github.com/ethereum/go-ethereum/eth/ethconfig"
	"github.com/ethereum/go-ethereum/eth/tracers"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rlp"
)

func main() {
	var (
		logLevel    = flag.String("loglevel", "info", "Log level")
		quiet       = flag.Bool("quiet", false, "Don't print any client logs")
		skipCompile = flag.Bool("skip-compile", false, "Skips compilation")
		devMode     = flag.Bool("dev", false, "Leaves client open after flag check")
	)

	flag.Parse()

	if err := checkFlag(*logLevel, *skipCompile, !*quiet, *devMode); err != nil {
		fmt.Fprintf(os.Stderr, "Flag not captured: %s\n", err)
		os.Exit(1)
	}

	fmt.Println("Flag captured.")
}

func checkFlag(logLevelStr string, skipCompile, verbose, devMode bool) error {
	// Start geth.
	node, err := runGeth()
	if err != nil {
		fmt.Println("erroring out")
		return err
	}

	if devMode {
		err = startConsole(node)
		if err != nil {
			return err
		}
		fmt.Println()
	}

	rpc, err := node.Attach()
	if err != nil {
		return nil
	}
	eth := ethclient.NewClient(rpc)

	// Verify flag.
	block, err := eth.BlockByNumber(context.Background(), common.Big1)
	if err != nil {
		return err
	}
	if block.Hash() != common.HexToHash("0x31553f1bb856b900a24d456f51ac4372fa57e08c5a16812db3ff87e63320bf26") {
		return fmt.Errorf("could not load chain")
	}

	return nil
}

// runGeth creates and starts a geth node
func runGeth() (*node.Node, error) {
	stack, err := node.New(&node.Config{
		P2P: p2p.Config{
			ListenAddr:  "127.0.0.1:0",
			NoDiscovery: true,
			NoDial:      true,
		},
	})
	if err != nil {
		return nil, err
	}

	fmt.Println("setting up geth")
	chain, err := loadChain("chain.rlp", "genesis.json")
	if err != nil {
		stack.Close()
		return nil, err
	}
	backend, err := eth.New(stack, &ethconfig.Config{
		Genesis:   &chain.genesis,
		NetworkId: chain.genesis.Config.ChainID.Uint64(),
		Ethash: ethash.Config{
			PowMode: ethash.ModeFake,
		},
	})
	if err != nil {
		stack.Close()
		return nil, err
	}
	stack.RegisterAPIs(tracers.APIs(tracers.Backend(backend.APIBackend)))

	_, err = backend.BlockChain().InsertChain(chain.blocks[1:])
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to import chain: %s", err)
	}

	if err = stack.Start(); err != nil {
		stack.Close()
		return nil, err
	}
	return stack, nil
}

type Chain struct {
	genesis     core.Genesis
	blocks      []*types.Block
	chainConfig *params.ChainConfig
}

func loadChain(chainfile string, genesis string) (*Chain, error) {
	gen, err := loadGenesis(genesis)
	if err != nil {
		return nil, err
	}
	gblock := gen.ToBlock()

	blocks, err := blocksFromFile(chainfile, gblock)
	if err != nil {
		return nil, err
	}

	c := &Chain{genesis: gen, blocks: blocks, chainConfig: gen.Config}
	return c, nil
}

func loadGenesis(genesisFile string) (core.Genesis, error) {
	chainConfig, err := os.ReadFile(genesisFile)
	if err != nil {
		return core.Genesis{}, err
	}
	var gen core.Genesis
	if err := json.Unmarshal(chainConfig, &gen); err != nil {
		return core.Genesis{}, err
	}
	return gen, nil
}

func blocksFromFile(chainfile string, gblock *types.Block) ([]*types.Block, error) {
	fh, err := os.Open(chainfile)
	if err != nil {
		return nil, err
	}
	defer fh.Close()
	var reader io.Reader = fh
	if strings.HasSuffix(chainfile, ".gz") {
		if reader, err = gzip.NewReader(reader); err != nil {
			return nil, err
		}
	}
	stream := rlp.NewStream(reader, 0)
	var blocks = make([]*types.Block, 1)
	blocks[0] = gblock
	for i := 0; ; i++ {
		var b types.Block
		if err := stream.Decode(&b); err == io.EOF {
			break
		} else if err != nil {
			return nil, fmt.Errorf("at block index %d: %v", i, err)
		}
		if b.NumberU64() != uint64(i+1) {
			return nil, fmt.Errorf("block at index %d has wrong number %d", i, b.NumberU64())
		}
		blocks = append(blocks, &b)
	}
	return blocks, nil
}

func startConsole(stack *node.Node) error {
	client, err := stack.Attach()
	if err != nil {
		return fmt.Errorf("Failed to attach to geth: %v", err)
	}
	config := console.Config{
		DataDir: "datadir",
		Client:  client,
	}
	console, err := console.New(config)
	if err != nil {
		return fmt.Errorf("Failed to start console: %v", err)
	}
	defer console.Stop(false)

	go func() {
		stack.Wait()
		console.StopInteractive()
	}()

	console.Welcome()
	console.Interactive()
	return nil
}
