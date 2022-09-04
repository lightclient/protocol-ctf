package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/lightclient/protocol-challenges/internal/harness"
)

func main() {
	var (
		logLevel    = flag.String("loglevel", "info", "Log level")
		quiet       = flag.Bool("quiet", false, "Don't print any client logs")
		skipCompile = flag.Bool("skip-compile", false, "Skips compilation")
	)
	flag.Parse()

	if err := checkFlag(*logLevel, *skipCompile, !*quiet); err != nil {
		exit(err)
	}
	fmt.Println("Flag captured.")
}

func checkFlag(logLevelStr string, skipCompile, verbose bool) error {
	logLevel, err := harness.ParseLogLevel(logLevelStr)
	if err != nil {
		return err
	}
	// Make directory to store client data.
	if err := makeDataDir("datadir"); err != nil {
		return err
	}
	defer os.RemoveAll("datadir")

	// Create client configuration
	args := &harness.ClientArgs{
		FakePow:     true,
		LogLevel:    logLevel,
		DataDir:     "datadir",
		GenesisPath: "genesis.json",
		ChainPath:   "chain.rlp",
	}
	client, err := harness.NewClient(harness.Geth, "go-ethereum", args)
	if err != nil {
		return err
	}

	// Compile client.
	ctx := context.Background()
	if skipCompile {
		fmt.Println("skipping compilation...")
	} else {
		if err := client.Compile(ctx, verbose); err != nil {
			return err
		}
	}

	// Initialize client.
	if err := client.Init(ctx, verbose); err != nil {
		return err
	}

	// Start client.
	if err := client.Start(ctx, verbose); err != nil {
		return err
	}
	defer client.Close()

	// Block until client is running.
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	if ok := client.Running(ctx); !ok {
		return fmt.Errorf("unable to connect to client")
	}

	eth, err := ethclient.DialContext(ctx, client.HttpAddr())
	if err != nil {
		return err
	}

	// Verify flag.
	num, err := eth.BlockNumber(ctx)
	if err != nil {
		return err
	}
	if num != 1 {
		return fmt.Errorf("chain not loaded correctly")
	}

	return nil
}

func exit(msg error) {
	fmt.Fprintf(os.Stderr, "%s\n", msg)
	os.Exit(1)
}

func makeDataDir(path string) error {
	if err := os.RemoveAll(path); err != nil {
		return err
	}
	if err := os.Mkdir(path, os.ModePerm); err != nil {
		return err
	}
	return nil
}
