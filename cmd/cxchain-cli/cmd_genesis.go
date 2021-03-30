package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/skycoin/cx-chains/src/cx/cxspec"
	"github.com/skycoin/cx-chains/src/cx/cxutil"
)

type genesisFlags struct {
	cmd *flag.FlagSet

	in string
}

func processGenesisFlags(args []string) genesisFlags {
	// Specify default flag values.
	f := genesisFlags{
		cmd: flag.NewFlagSet("cxchain-cli genesis", flag.ExitOnError),
		in:  "skycoin.chain_spec.json",
	}

	f.cmd.Usage = func() {
		usage := cxutil.DefaultUsageFormat("flags")
		usage(f.cmd, nil)
	}

	f.cmd.StringVar(&f.in, "in", f.in, "`FILENAME` of file to read in")

	if err := f.cmd.Parse(args); err != nil {
		os.Exit(1)
	}

	return f
}

func cmdGenesis(args []string) {
	flags := processGenesisFlags(args)

	f, err := os.Open(flags.in)
	if err != nil {
		log.WithError(err).
			Fatal("Failed to read in file.")
	}
	defer func() {
		if err := f.Close(); err != nil {
			log.WithError(err).
				Fatal("Failed to close file.")
		}
	}()

	var cSpec cxspec.ChainSpec
	if err := json.NewDecoder(f).Decode(&cSpec); err != nil {
		log.WithError(err).
			Fatal("Failed to decode file")
	}

	block, err := cSpec.ObtainGenesisBlock()
	if err != nil {
		log.WithError(err).
			Fatal("Failed to generate genesis block.")
	}

	hash := block.HashHeader()
	fmt.Println(hash.Hex())
}
