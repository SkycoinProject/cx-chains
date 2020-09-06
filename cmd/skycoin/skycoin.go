/*
skycoin daemon
*/
package main

/*
CODE GENERATED AUTOMATICALLY WITH FIBER COIN CREATOR
AVOID EDITING THIS MANUALLY
*/

import (
	"flag"
	_ "net/http/pprof"
	"os"

	"github.com/SkycoinProject/cx-chains/src/fiber"
	"github.com/SkycoinProject/cx-chains/src/readable"
	"github.com/SkycoinProject/cx-chains/src/skycoin"
	"github.com/SkycoinProject/cx-chains/src/util/logging"
)

var (
	// Version of the node. Can be set by -ldflags
	Version = "0.26.0"
	// Commit ID. Can be set by -ldflags
	Commit = ""
	// Branch name. Can be set by -ldflags
	Branch = ""
	// ConfigMode (possible values are "", "STANDALONE_CLIENT").
	// This is used to change the default configuration.
	// Can be set by -ldflags
	ConfigMode = ""

	logger = logging.MustGetLogger("main")

	// CoinName name of coin
	CoinName = "skycoin"

	// GenesisSignatureStr hex string of genesis signature
	GenesisSignatureStr = "a214e0361ff99d80d2f9d646b25f93b8d1d2deb9f7bae0ff908d2302193d8cc31b8388b7bd38c019304b932bfd570444dbe8561aa9d47da021fd31a70146defd01"
	// GenesisAddressStr genesis address string
	GenesisAddressStr = "23v7mT1uLpViNKZHh9aww4VChxizqKsNq4E"
	// BlockchainPubkeyStr pubic key string
	BlockchainPubkeyStr = "02583e5ebbf85522474e0f17e681e62ca37910db6b8792763af4e97663c31a7984"
	// BlockchainSeckeyStr empty private key string
	BlockchainSeckeyStr = ""

	// GenesisTimestamp genesis block create unix time
	GenesisTimestamp uint64 = 1426562704
	// GenesisCoinVolume represents the coin capacity
	GenesisCoinVolume uint64 = 100000000000000

	// DefaultConnections the default trust node addresses
	DefaultConnections = []string{
		"118.178.135.93:6000",
		"47.88.33.156:6000",
		"104.237.142.206:6000",
		"176.58.126.224:6000",
		"172.104.85.6:6000",
		"139.162.7.132:6000",
		"139.162.39.186:6000",
		"45.33.111.142:6000",
		"109.237.27.172:6000",
		"172.104.41.14:6000",
		"172.104.114.58:6000",
		"172.104.71.211:6000",
		"172.105.217.244:6000",
		"139.162.98.190:6000",
	}

	nodeConfig = skycoin.NewNodeConfig(ConfigMode, fiber.NodeConfig{
		CoinName:            CoinName,
		GenesisSignatureStr: GenesisSignatureStr,
		GenesisAddressStr:   GenesisAddressStr,
		GenesisCoinVolume:   GenesisCoinVolume,
		GenesisTimestamp:    GenesisTimestamp,
		BlockchainPubkeyStr: BlockchainPubkeyStr,
		BlockchainSeckeyStr: BlockchainSeckeyStr,
		DefaultConnections:  DefaultConnections,
		PeerListURL:         "https://downloads.skycoin.net/blockchain/peers.txt",
		Port:                6000,
		WebInterfacePort:    6420,
		DataDirectory:       "$HOME/.skycoin",

		UnconfirmedBurnFactor:          10,
		UnconfirmedMaxTransactionSize:  65535,
		UnconfirmedMaxDropletPrecision: 3,
		CreateBlockBurnFactor:          10,
		CreateBlockMaxTransactionSize:  65535,
		CreateBlockMaxDropletPrecision: 3,
		MaxBlockTransactionsSize:       5242880,

		DisplayName:     "Skycoin",
		Ticker:          "SKY",
		CoinHoursName:   "Coin Hours",
		CoinHoursTicker: "SCH",
		ExplorerURL:     "https://explorer.skycoin.net",
	})

	parseFlags = true
)

func init() {
	nodeConfig.RegisterFlags(flag.CommandLine)
}

func main() {
	if parseFlags {
		flag.Parse()
	}

	// create a new fiber coin instance
	coin := skycoin.NewCoin(skycoin.Config{
		Node: nodeConfig,
		Build: readable.BuildInfo{
			Version: Version,
			Commit:  Commit,
			Branch:  Branch,
		},
	}, logger)

	// parse config values
	if err := coin.ParseConfig(flag.CommandLine); err != nil {
		logger.Error(err)
		os.Exit(1)
	}

	// run fiber coin node
	if err := coin.Run(); err != nil {
		os.Exit(1)
	}
}
