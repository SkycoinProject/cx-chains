package main

import (
	"context"
	"encoding/hex"
	"flag"
	"fmt"
	"os"

	"github.com/skycoin/skycoin/src/cipher"

	"github.com/skycoin/cx-chains/src/api"

	"github.com/skycoin/cx-chains/src/cx/cxutil"

	"github.com/skycoin/cx-chains/src/cx/cxflags"
)

type stateFlags struct {
	cmd *flag.FlagSet

	*cxflags.MemoryFlags

	nodeAddr string // Node address.
	appAddr  string // App address (unspecified == genesis address).
}

func processStateFlags(args []string) (stateFlags, cipher.Address) {
	cmd := flag.NewFlagSet("cxchain-cli state", flag.ExitOnError)
	spec := processSpecFlags(context.Background(), cmd, args)

	webPort, err := spec.ObtainWebInterfacePort()
	if err != nil {
		log.WithField("spec_era", spec.CXSpecEra()).
			Fatal("Failed to obtain web interface port.")
	}

	genesisAddr, err := spec.ObtainGenesisAddr()
	if err != nil {
		log.WithField("spec_era", spec.CXSpecEra()).
			Fatal("Failed to obtain genesis address.")
	}

	f := stateFlags{
		cmd:         cmd,
		MemoryFlags: cxflags.DefaultMemoryFlags(),
		nodeAddr:    fmt.Sprintf("http://127.0.0.1:%d", webPort),
		appAddr:     genesisAddr.String(),
	}

	f.cmd.Usage = func() {
		usage := cxutil.DefaultUsageFormat("flags")
		usage(f.cmd, nil)
		// TODO @evanlinjin: Print ENV help.
	}

	f.MemoryFlags.Register(f.cmd)

	f.cmd.StringVar(&f.nodeAddr, "node", f.nodeAddr, "HTTP API `ADDRESS` of cxchain node")
	f.cmd.StringVar(&f.nodeAddr, "n", f.nodeAddr, "shorthand for 'node'")

	f.cmd.StringVar(&f.appAddr, "app", f.appAddr, "`ADDRESS` of cx app")
	f.cmd.StringVar(&f.appAddr, "a", f.appAddr, "shorthand for 'app'")

	// Parse flags.
	if err := f.cmd.Parse(args); err != nil {
		os.Exit(1)
	}

	addr, err := cipher.DecodeBase58Address(f.appAddr)
	if err != nil {
		log.WithError(err).
			WithField("addr", f.appAddr).
			WithField("flag", "app,a").
			Fatal("Invalid flag value.")
	}

	return f, addr
}

func cmdState(args []string) {
	flags, addr := processStateFlags(args)

	c := api.NewClient(flags.nodeAddr)

	ux, err := RetryObtainProgramStateUxOut(c, addr)
	if err != nil {
		log.WithError(err).Fatal("Failed to obtain program state.")
	}

	fmt.Println(hex.EncodeToString(ux.Body.ProgramState))
}
