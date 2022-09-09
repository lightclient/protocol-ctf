# Protocol Capture the Flag

The Protocol CTF is a list of challenges that simulate real-world challenges
faced by Ethereum core developers. Each challenge has a flag that can be
captured. To verify capture, run the verifier program in the challenge
directory.

## The Challenges

### Easy
- [The Price is Wrong][wrong-price]


## Getting Started

First install `go`. Either use your favorite package manager or download
a pre-built binary [here][gobin].

Verify the installation was successful.

```console
$ go version
go version go1.19 linux/amd64
```

Next clone this repository.
```console
git clone https://github.com/lightclient/protocol-ctf.git
```

Next, decide a challenge to attempt. The recommended starter
challenge is [The Price is Wrong][wrong-price].

Each challenge will have `README.md` explaining the challenge. Generally there
will also be a `go-ethereum` directory which will need to be modified to
capture the flag.

To verify the flag is captured, run `main.go` program in the challenge
directory.

```console
go run main.go --quiet
Flag captured.
```

## Challenge Structure

Most challenges have a similar set of files and directories. Let's examine
each.

- `chain.rlp`    - The chain that will be imported into the client.
- `genesis.json` - The genesis configuration for the chain, including
                   preallocated accounts.
- `go-ethereum`  - This directory contains the [`go-ethereum`][geth] codebase
                   that will be used to verify the challenge.
- `main.go`      - Program to verify the challenge. It works by first compiling
                   `go-ethereum`, then initializes it with `genesis.json` and
                   `chain.rlp`. Finally, it starts the client and checks the
                   flag condition is met via [JSON-RPC][jsonrpc].
- `README.md`    - Information on the challenge and completion criteria.

## Strategy

The first step for each challenge is understand the completion criteria.
Sometimes they may be vague, e.g. "the client is able to load all blocks to
height N". In this case, it's good to go ahead and run the verifier with
logging on to understand what is stopping the client from processing each
block.

```console
$ go run main.go --loglevel=debug
...
########## BAD BLOCK #########
Chain config: {ChainID: 1, Homestead: 0, DAO: <nil>, DAOSupport: false, EIP150: 0, EIP155: 0, EIP158: 0, Byzantium: 0, Constantinople: 0, Petersburg: 0, Istanbul: 0, Muir Glacier: 0, Berlin: 0, London: 0, Arrow Glacier: 0, MergeFork: <nil>, ShanghaiFork: <nil>, CancunFork: <nil>, Terminal TD: <nil>, Engine: ethash}

Number: 1
Hash: 0x31553f1bb856b900a24d456f51ac4372fa57e08c5a16812db3ff87e63320bf26
         0: cumulative: 21011 gas: 21011 contract: 0x0000000000000000000000000000000000000000 status: 1 tx: 0x00c8d2ea97ca412f2c474b9dbcdf6e2447e2feb5c7c5cc1b230c5a654dcd2f69 logs: [] bloom: 00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000 state:


Error: invalid gas used (remote: 21009 local: 21011)
##############################
...
```

This presents the error `invalid gas used` while processing transaction 0 in
block 1. There are many reasons a block could fail validation, but this one
usually occurs when there is an issue in the EVM that causes execution to
deviate from the canonical trace.

To determine where / why the deviation occurs, there are few approaches.

## Test Harness

Challenges typically work by importing `chain.rlp` and verifying all blocks are
imported correctly. It's possible to simulate this behavior by using a
combination of client commands:

```
geth --datadir=ctf init genesis.json
geth --datadir=ctf import chain.rlp
geth --datadir=ctf console
```

This sequence of commands will start `geth` with the same initial state as it
would start via the harness. It's possible to now poke at it to better
understand its current state.

## Contributing

New challenges are not only welcome, but greatly appreciated. Please review the
[Challenge Structure](#challenge-structure) section and the format of existing
challenges for guidance.

## License

The content in this repository is licensed under the MIT license, with the
exception of the `go-ethereum` source code.

--

[wrong-price]: flags/wrong-price
[gobin]: https://go.dev/learn/
[geth]: https://github.com/ethereum/go-ethereum
[jsonrpc]: https://github.com/ethereum/execution-apis
