package cxspec

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"time"

	"github.com/skycoin/cx-chains/src/api"
	"github.com/skycoin/cx-chains/src/kvstorage"
	"github.com/skycoin/cx-chains/src/params"
	"github.com/skycoin/cx-chains/src/readable"
	"github.com/skycoin/cx-chains/src/skycoin"
	"github.com/skycoin/cx-chains/src/wallet"
)

// ReadSpecFile reads chain spec from given filename.
func ReadSpecFile(filename string) (ChainSpec, error) {
	b, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read chain spec file '%s': %w", filename, err)
	}
	var spec ChainSpec
	if err := json.Unmarshal(b, &spec); err != nil {
		return nil, fmt.Errorf("chain spec file '%s' is ill-formed: %w", filename, err)
	}
	if _, err := spec.ObtainGenesisBlock(); err != nil {
		return nil, fmt.Errorf("chain spec file '%s' cannot generate genesis block: %w", filename, err)
	}
	return spec, nil
}

func ReadKeysFile(filename string) (KeySpec, error) {
	b, err := ioutil.ReadFile(filename)
	if err != nil {
		return KeySpec{}, fmt.Errorf("failed to read key file '%s': %w", filename, err)
	}
	var spec KeySpec
	if err := json.Unmarshal(b, &spec); err != nil {
		return KeySpec{}, fmt.Errorf("key file '%s' is ill-formed: %w", filename, err)
	}
	return spec, nil
}

// BaseNodeConfig returns the base node config.
// Fields which requires values from the cx chain spec are left blank.
func BaseNodeConfig() skycoin.NodeConfig {
	conf := skycoin.NodeConfig{
		CoinName:                          "", // populate with cx spec
		DisablePEX:                        false,
		DownloadPeerList:                  false,
		PeerListURL:                       "", // populate with cx spec
		DisableOutgoingConnections:        false,
		DisableIncomingConnections:        false,
		DisableNetworking:                 false,
		EnableGUI:                         false,
		DisableCSRF:                       false,
		DisableHeaderCheck:                false,
		DisableCSP:                        false,
		EnabledAPISets:                    "",
		DisabledAPISets:                   "",
		EnableAllAPISets:                  false,
		HostWhitelist:                     "",
		LocalhostOnly:                     true,
		Address:                           "",
		Port:                              0, // populate with cx spec
		MaxConnections:                    128,
		MaxOutgoingConnections:            8,
		MaxDefaultPeerOutgoingConnections: 1,
		OutgoingConnectionsRate:           time.Second * 5,
		MaxOutgoingMessageLength:          5243081 * 2, // TODO @evanlinjin: Find a way to regulate this with cx txns (originally 256 * 1024).
		MaxIncomingMessageLength:          5243081 * 4, // TODO @evanlinjin: Find a way to regulate this with cx txns (originally 1024 * 1024).
		PeerlistSize:                      65535,
		WebInterface:                      true,
		WebInterfacePort:                  0, // populate with cx spec
		WebInterfaceAddr:                  "127.0.0.1",
		WebInterfaceCert:                  "",
		WebInterfaceKey:                   "",
		WebInterfaceHTTPS:                 false,
		WebInterfaceUsername:              "",
		WebInterfacePassword:              "",
		WebInterfacePlaintextAuth:         false,
		LaunchBrowser:                     false,
		DataDirectory:                     "$HOME/.cxchain/{coin}", // populate with cx spec
		GUIDirectory:                      "./src/gui/static/",
		HTTPReadTimeout:                   time.Second * 10,
		HTTPWriteTimeout:                  time.Second * 60,
		HTTPIdleTimeout:                   time.Second * 120,
		UserAgentRemark:                   "",
		ColorLog:                          true,
		LogLevel:                          "INFO",
		DisablePingPong:                   false,
		VerifyDB:                          false,
		ResetCorruptDB:                    false,
		UnconfirmedVerifyTxn:              params.VerifyTxn{}, // populate with cx spec
		CreateBlockVerifyTxn:              params.VerifyTxn{}, // populate with cx spec
		MaxBlockTransactionsSize:          0,                  // populate with cx spec
		WalletDirectory:                   "",
		WalletCryptoType:                  string(wallet.CryptoTypeScryptChacha20poly1305),
		KVStorageDirectory:                "",
		EnabledStorageTypes: []kvstorage.Type{
			kvstorage.TypeTxIDNotes,
			kvstorage.TypeGeneral,
		},
		DisableDefaultPeers: false,
		CustomPeersFile:     "",
		RunBlockPublisher:   false,
		ProfileCPU:          false,
		ProfileCPUFile:      "cpu.prof",
		HTTPProf:            false,
		HTTPProfHost:        "localhost:6060",
		DBPath:              "",
		DBReadOnly:          false,
		LogToFile:           false,
		Version:             false,
		GenesisSignatureStr: "", // populate with cx spec
		GenesisAddressStr:   "", // populate with cx spec
		BlockchainPubkeyStr: "", // populate with cx spec
		BlockchainSeckeyStr: "",
		GenesisTimestamp:    0,                      // populate with cx spec
		GenesisCoinVolume:   0,                      // populate with cx spec
		DefaultConnections:  nil,                    // populate with cx spec
		Fiber:               readable.FiberConfig{}, // populate with cx spec
	}

	return conf
}

// ApplyStandaloneClientMode alters a node config for standalone node use-cases.
func ApplyStandaloneClientMode(conf *skycoin.NodeConfig) {
	conf.EnableAllAPISets = true
	conf.EnabledAPISets = api.EndpointsInsecureWalletSeed
	conf.EnableGUI = true
	conf.LaunchBrowser = true
	conf.DisableCSRF = false
	conf.DisableHeaderCheck = false
	conf.DisableCSP = false
	conf.DownloadPeerList = true
	conf.WebInterface = true
	conf.LogToFile = false
	conf.ResetCorruptDB = true
	conf.WebInterfacePort = 0 // randomize web interface port
}
