package main

import (
	"context"
	"encoding/hex"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/skycoin/dmsg"
	cipher2 "github.com/skycoin/dmsg/cipher"
	"github.com/skycoin/skycoin/src/cipher"
	"github.com/skycoin/skycoin/src/util/logging"

	"github.com/skycoin/cx-chains/src/api"
	"github.com/skycoin/cx-chains/src/cx/cxdmsg"
	"github.com/skycoin/cx-chains/src/cx/cxspec"
	"github.com/skycoin/cx-chains/src/readable"
	"github.com/skycoin/cx-chains/src/skycoin"
)

const (
	// ENV for the chain secret key (in hex).
	secKeyEnv = "CXCHAIN_SK"

	// ENVs for config modes.
	standaloneClientConfMode = "STANDALONE_CLIENT"
)

// These values should be populated by -ldflags on compilation.
var (
	version  = "0.0.0"
	commit   = ""
	branch   = ""
	confMode = "" // valid values: "STANDALONE_CLIENT", ""
)

// log contains the main logger.
var log = logging.MustGetLogger("main")

// Additional flags.
var (
	dmsgDiscAddr = cxdmsg.DefaultDiscAddr     // dmsg discovery address
	dmsgPort     = uint64(cxdmsg.DefaultPort) // dmsg listening port
	forceClient  = false                      // more client mode (as opposed to publisher)

	// specFlags contains default cx spec discovery flags
	specFlags = cxspec.DefaultLocateConfig()
)

// locateSpec locates the spec location either from a local spec file or from a
// CX tracker instance.
func locateSpec() cxspec.ChainSpec {
	specFlags.RegisterFlags(flag.CommandLine)
	specFlags.SoftParse(os.Args)

	spec, err := cxspec.LocateWithConfig(context.Background(), &specFlags)
	if err != nil {
		log.WithError(err).
			WithField("chain", specFlags.CXChain).
			Fatal("Failed to find cx spec.")
	}

	return spec
}

// parseSecretKeyEnv parses secret key from CXCHAIN_SECRET_KEY env.
// The secret key can be null.
func parseSecretKeyEnv() cipher.SecKey {
	if skStr, ok := os.LookupEnv(secKeyEnv); ok {
		sk, err := cipher.SecKeyFromHex(skStr)
		if err != nil {
			log.WithError(err).
				WithField("ENV", secKeyEnv).
				Fatal("Provided secret key is invalid.")
		}
		return sk
	}
	return cipher.SecKey{} // return nil secret key
}

// ensureConfMode ensures 'confMode' settings are applied on the node config.
// 'confMode' is set on compile time.
func ensureConfMode(conf *skycoin.NodeConfig) {
	switch confMode {
	case "":
	case standaloneClientConfMode:
		cxspec.ApplyStandaloneClientMode(conf)
	default:
		log.Fatal("Invalid 'confMode' provided at build time. This cannot be fixed without recompiling the binary.")
	}
}

// trackerUpdateLoop updates the cx tracker of the current node state.
func trackerUpdateLoop(nodeSK cipher.SecKey, nodeTCPAddr string, spec cxspec.ChainSpec) {
	log := logging.MustGetLogger("cx_tracker_client")

	client := cxspec.NewCXTrackerClient(log, nil, specFlags.CXTracker)
	nodePK := cipher.MustPubKeyFromSecKey(nodeSK)

	block, err := spec.ObtainGenesisBlock()
	if err != nil {
		panic(err) // should not happen
	}
	hash := block.HashHeader()

	chainPK, err := spec.ObtainChainPubKey()
	if err != nil {
		panic(err) // TODO @evanlinjin: Proper fail and error reporting?
	}

	// If publisher, ensure spec is registered.
	if isPub := nodePK == chainPK; isPub {
		signedSpec, err := cxspec.MakeSignedChainSpec(spec, nodeSK)
		if err != nil {
			panic(err) // should not happen
		}
		for {
			if err := client.PostSpec(context.Background(), signedSpec); err != nil {
				// TODO @evanlinjin: This is a temporary solution for a bug in the cx tracker client.
				// TODO @evanlinjin: We should have a proper error type of conflicts which we can check here.
				if strings.Contains(err.Error(), "409 Conflict") {
					break
				}

				log.WithError(err).Error("Failed to post spec, retrying again...")
				time.Sleep(time.Second * 10)
				continue
			}
			break
		}
	}

	// Prepare ticker for cx tracker peer entry update loop.
	ticker := time.NewTicker(time.Second * 30)
	defer ticker.Stop()

	// All peers need to update entry.
	entry := cxspec.PeerEntry{
		PublicKey: cipher2.PubKey(nodePK),
		CXChains: map[string]cxspec.CXChainAddresses{
			hex.EncodeToString(hash[:]): {
				DmsgAddr: dmsg.Addr{PK: cipher2.PubKey(nodePK), Port: uint16(dmsgPort)},
				TCPAddr:  nodeTCPAddr,
			},
		},
	}
	updateEntry := func(now int64) {
		entry.LastSeen = now
		signedEntry, err := cxspec.MakeSignedPeerEntry(entry, cipher2.SecKey(nodeSK))
		if err != nil {
			panic(err) // should not happen
		}
		if err := client.UpdatePeerEntry(context.Background(), signedEntry); err != nil {
			log.WithError(err).Warn("Failed to update peer entry in cx tracker. Retrying...")
		}
	}
	updateEntry(time.Now().Unix())
	for now := range ticker.C {
		updateEntry(now.Unix())
	}
}

func main() {
	// Register and parse flags for cx chain spec.
	spec := locateSpec()
	if err := spec.PopulateParamsModule(); err != nil {
		log.WithError(err).Fatal("Failed to populate params module with spec.")
	}
	log.Info(spec.String())

	// Register additional CLI flags.
	cmd := flag.CommandLine
	cmd.StringVar(&dmsgDiscAddr, "dmsg-disc", dmsgDiscAddr, "HTTP `ADDRESS` of dmsg discovery")
	cmd.Uint64Var(&dmsgPort, "dmsg-port", dmsgPort, "dmsg `PORT` number to listen on")
	cmd.BoolVar(&forceClient, "client", forceClient, "force client mode (even with master sk set)")

	// Parse ENV for node secret key.
	nodeSK := parseSecretKeyEnv()

	var nodePK cipher.PubKey

	// Node config: Init.
	conf := cxspec.BaseNodeConfig()
	ensureConfMode(&conf)

	// Node config: Populate node config based on chain spec content.
	if err := spec.PopulateNodeConfig(&conf); err != nil {
		log.WithError(err).Fatal("Failed to populate node config from spec file.")
	}
	// TODO @evanlinjin: Do we need to do something about the cx tracker URL?

	// Node config: Ensure node keys are set.
	//	- If node secret key is null, randomly generate one.
	if nodeSK.Null() {
		nodePK, nodeSK = cipher.GenerateKeyPair()
		log.WithField("node_pk", nodePK.Hex()).
			Warn("Node secret key is not defined. Random key pair generated.")
	}
	if err := nodeSK.Verify(); err != nil {
		log.WithError(err).Fatal("Failed to verify provided node secret key.")
	}
	nodePK = cipher.MustPubKeyFromSecKey(nodeSK)

	chainPK, err := spec.ObtainChainPubKey()
	if err != nil {
		log.WithError(err).Fatal("Failed to obtain chain pk from spec.")
	}

	// Node config: Enable publisher mode if conditions are met.
	// - Skip if 'forceClient' is set.
	// - Skip if 'nodePK' is not equal to chain spec's PK.
	if !forceClient && nodePK == chainPK {
		conf.BlockchainSeckeyStr = nodeSK.Hex()
		conf.RunBlockPublisher = true
	}

	// Node config: Register node flags and parse entire flag set.
	conf.RegisterFlags(flag.CommandLine)
	flag.Parse()

	coin := skycoin.NewCoin(skycoin.Config{
		Node: conf,
		Build: readable.BuildInfo{
			Version: version,
			Commit:  commit,
			Branch:  branch,
		},
	}, log)

	// This is post-processing and not "parsing" (despite the name).
	// Do not get confused. I did not name this function. <3 @evanlinjin
	if err := coin.ParseConfig(flag.CommandLine); err != nil {
		log.Error(err)
		os.Exit(1)
	}

	gwCh := make(chan api.Gatewayer)
	defer close(gwCh)

	go func() {
		// await gateway to be loaded and ready
		gw, ok := <-gwCh
		if !ok {
			return
		}

		// Run cx tracker loop.
		// - Node params are uploaded to the tracker.
		// - Send keepalive requests on regular intervals.
		// 'addr' is the daemon IP address
		dConf := gw.DaemonConfig()
		addr := fmt.Sprintf("%s:%d", dConf.Address, dConf.Port)
		go trackerUpdateLoop(nodeSK, addr, spec)

		// Prepare API to be served via dmsg.
		dmsgAPI := &cxdmsg.API{
			Version:   version,
			NodeConf:  conf,
			ChainSpec: spec,
			Gateway:   gw,
		}

		// Prepare dmsg config.
		dmsgConf := &cxdmsg.Config{
			PK:       cipher2.PubKey(nodePK),
			SK:       cipher2.SecKey(nodeSK),
			DiscAddr: dmsgDiscAddr,
			DmsgPort: uint16(dmsgPort),
		}

		// Run dmsg loop.
		dmsgCtx := context.Background()
		dmsgLog := logging.MustGetLogger("dmsgC")
		cxdmsg.ServeDmsg(dmsgCtx, dmsgLog, dmsgConf, dmsgAPI)
	}()

	progState, err := spec.ObtainGenesisProgState()
	if err != nil {
		log.WithError(err).Fatal("Failed to obtain genesis program state from spec.")
	}

	// Run main daemon.
	if err := coin.Run(progState, gwCh); err != nil {
		os.Exit(1)
	}
}
