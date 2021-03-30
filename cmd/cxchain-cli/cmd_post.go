package main

import (
	"context"
	"flag"
	"os"

	"github.com/skycoin/skycoin/src/cipher"

	"github.com/skycoin/cx-chains/src/cx/cxspec"
	"github.com/skycoin/cx-chains/src/cx/cxutil"
)

type postFlags struct {
	cmd *flag.FlagSet

	specInput    string // chain spec input filename
	signedOutput string // signed chain spec output filename
	dryRun       bool   // if set, spec file will not be posted to cx-tracker
	tracker      string // cx tracker URL
}

func processPostFlags(args []string) (postFlags, cipher.SecKey) {
	// Specify default flag values.
	f := postFlags{
		cmd: flag.NewFlagSet("cxchain-cli post", flag.ExitOnError),

		specInput:    cxspec.DefaultSpecFilepath,
		signedOutput: "", // empty for no output
		dryRun:       false,
		tracker:      cxspec.DefaultTrackerURL,
	}

	f.cmd.Usage = func() {
		usage := cxutil.DefaultUsageFormat("flags")
		usage(f.cmd, nil)
		printChainSKENV(f.cmd)
	}

	f.cmd.StringVar(&f.specInput, "spec", f.specInput, "`FILENAME` of chain spec file input")
	f.cmd.StringVar(&f.specInput, "s", f.specInput, "shorthand for 'spec'")

	f.cmd.StringVar(&f.signedOutput, "output", f.signedOutput, "`FILENAME` for signed chain spec output (empty for no output)")
	f.cmd.StringVar(&f.signedOutput, "o", f.signedOutput, "shorthand for 'output'")

	f.cmd.BoolVar(&f.dryRun, "dry", f.dryRun, "whether to do a dry run (no actual post to cx-tracker)")

	f.cmd.StringVar(&f.tracker, "tracker", f.tracker, "`URL` for cx-tracker")
	f.cmd.StringVar(&f.tracker, "t", f.tracker, "shorthand for 'tracker'")

	// Parse flags.
	if err := f.cmd.Parse(args); err != nil {
		os.Exit(1)
	}

	// Parse ENVs.
	genSK, err := parseSKEnv(chainSKEnv)
	if err != nil {
		log.WithError(err).
			WithField(chainSKEnv, genSK.Hex()).
			Fatal("Failed to read secret key from ENV.")
	}

	return f, genSK
}

func cmdPost(args []string) {
	flags, genSK := processPostFlags(args)

	// Obtain chain spec.
	spec, err := cxspec.ReadSpecFile(flags.specInput)
	if err != nil {
		log.WithError(err).
			Fatal("Failed to read spec file.")
	}

	// Sign chain spec.
	signed, err := cxspec.MakeSignedChainSpec(spec, genSK)
	if err != nil {
		log.WithError(err).
			Fatal("Failed to make signed chain spec.")
	}

	if flags.signedOutput != "" {
		// TODO @evanlinjin: Write signed output.
		panic("flag 'output,o' is not implemented yet")
	}

	// tracker client.
	tC := cxspec.NewCXTrackerClient(log, nil, flags.tracker)

	if !flags.dryRun {
		if err := tC.PostSpec(context.Background(), signed); err != nil {
			log.WithError(err).Fatal("Failed to post spec to cx-tracker.")
		}
	} else {
		log.WithField("dry_run", flags.dryRun).
			Info("This is a dry run.")
	}

	log.WithField("spec_file", flags.specInput).
		WithField("cx_tracker", flags.tracker).
		Info("Chain spec file successfully posted!")
}
