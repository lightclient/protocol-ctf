package harness

import (
	"context"
	"fmt"
	"os"
	"os/exec"

	"github.com/ethereum/go-ethereum/ethclient"
)

const (
	NETWORKPORT = "33333"
	HOST        = "localhost"
	PORT        = "8545"
)

type ClientType int

const (
	Geth ClientType = iota
)

type LogLevel int

const (
	None = iota
	Err
	Warn
	Info
	Debug
	Trace
)

func ParseLogLevel(s string) (LogLevel, error) {
	switch s {
	case "none":
		return None, nil
	case "error":
		return Err, nil
	case "warn":
		return Warn, nil
	case "info":
		return Info, nil
	case "debug":
		return Debug, nil
	case "Trace":
		return Trace, nil
	default:
		return 0, fmt.Errorf("unknown log level: %s", s)
	}
}

// Client is an interface for generically interacting with Ethereum clients.
type Client interface {
	// Compiles the client.
	Compile(ctx context.Context, verbose bool) error

	// Init initializes client.
	Init(ctx context.Context, verbose bool) error

	// Start starts client, but does not wait for the command to exit.
	Start(ctx context.Context, verbose bool) error

	// Running returns whether the client is running.
	Running(ctx context.Context) bool

	// HttpAddr returns the address where the client is servering its
	// JSON-RPC.
	HttpAddr() string

	// Close closes the client.
	Close() error
}

type ClientArgs struct {
	FakePow     bool
	LogLevel    LogLevel
	DataDir     string
	GenesisPath string
	ChainPath   string
}

// NewClient creates a new Client object.
func NewClient(t ClientType, path string, args *ClientArgs) (Client, error) {
	var (
		client Client
		err    error
	)
	switch t {
	case Geth:
		client, err = newGethClient(path, args)
	default:
		return nil, fmt.Errorf("client type unimplemented")
	}
	return client, err
}

// gethClient is a wrapper around a go-ethereum instance on a separate thread.
type gethClient struct {
	cmd  *exec.Cmd
	path string
	args *ClientArgs
}

// newGethClient instantiates a new gethClient.
func newGethClient(path string, args *ClientArgs) (*gethClient, error) {
	return &gethClient{path: path, args: args}, nil
}

// Compile compiles the go-ethereum project rooted at path.
func (g *gethClient) Compile(ctx context.Context, verbose bool) error {
	dir, err := os.Getwd()
	if err != nil {
		return err
	}
	defer os.Chdir(dir)
	os.Chdir(g.path)

	if err := runCmd(ctx, "go", verbose, "run", "build/ci.go", "install", "./cmd/geth"); err != nil {
		return err
	}
	return nil
}

// Init initializes geth.
func (g *gethClient) Init(ctx context.Context, verbose bool) error {
	var (
		isFakepow = g.args.FakePow
		datadir   = fmt.Sprintf("--datadir=%s", g.args.DataDir)
		loglevel  = fmt.Sprintf("--verbosity=%d", g.args.LogLevel)
	)

	// Run geth init.
	options := []string{datadir, loglevel, "init", g.args.GenesisPath}
	options = maybePrepend(isFakepow, options, "--fakepow")
	err := runCmd(ctx, gethBin(g.path), verbose, options...)
	if err != nil {
		return err
	}

	// Run geth import.
	options = []string{datadir, loglevel, "import", g.args.ChainPath}
	options = maybePrepend(isFakepow, options, "--fakepow")
	err = runCmd(ctx, gethBin(g.path), verbose, options...)
	if err != nil {
		return err
	}

	return nil
}

// Start starts geth, but does not wait for the command to exit.
func (g *gethClient) Start(ctx context.Context, verbose bool) error {
	// Start geth.
	options := []string{
		fmt.Sprintf("--datadir=%s", g.args.DataDir),
		fmt.Sprintf("--verbosity=%d", g.args.LogLevel),
		fmt.Sprintf("--port=%s", NETWORKPORT),
		"--nodiscover",
		"--http",
		"--http.api=admin,eth,debug",
		fmt.Sprintf("--http.addr=%s", HOST),
		fmt.Sprintf("--http.port=%s", PORT),
	}
	options = maybePrepend(g.args.FakePow, options, "--fakepow")
	g.cmd = exec.CommandContext(
		ctx,
		gethBin(g.path),
		options...,
	)
	if verbose {
		g.cmd.Stdout = os.Stdout
		g.cmd.Stderr = os.Stderr
	}
	if err := g.cmd.Start(); err != nil {
		return err
	}
	return nil
}

// HttpAddr returns the address where the client is servering its JSON-RPC.
func (g *gethClient) HttpAddr() string {
	return fmt.Sprintf("http://%s:%s", HOST, PORT)
}

// Close closes the client.
func (g *gethClient) Close() error {
	g.cmd.Process.Kill()
	g.cmd.Wait()
	os.RemoveAll(g.args.DataDir)
	return nil
}

func (g *gethClient) Running(ctx context.Context) bool {
	for {
		select {
		case <-ctx.Done():
			return false
		default:
			client, err := ethclient.DialContext(ctx, g.HttpAddr())
			if err != nil {
				// Client may still be starting.
				continue
			}
			if _, err := client.BlockNumber(ctx); err == nil {
				return true
			}
		}
	}
}

// runCmd runs a command and outputs the command's stdout and stderr to the
// caller's stdout and stderr if verbose is set.
func runCmd(ctx context.Context, path string, verbose bool, args ...string) error {
	cmd := exec.CommandContext(ctx, path, args...)
	if verbose {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	}
	if err := cmd.Run(); err != nil {
		return err
	}
	return nil
}

func gethBin(root string) string {
	return fmt.Sprintf("%s/build/bin/geth", root)
}

func maybePrepend(shouldAdd bool, options []string, maybe ...string) []string {
	if shouldAdd {
		options = append(maybe, options...)
	}
	return options
}
