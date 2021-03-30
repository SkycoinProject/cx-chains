package alpha

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"
	"unicode"

	"github.com/skycoin/skycoin/src/cipher"

	"github.com/skycoin/cx-chains/src/coin"
	"github.com/skycoin/cx-chains/src/params"
	"github.com/skycoin/cx-chains/src/readable"
	"github.com/skycoin/cx-chains/src/skycoin"
)

const Era = "cx_alpha"

var progSEnc = base64.StdEncoding

// ProtocolParams defines the coin's consensus parameters.
type ProtocolParams struct {
	UnconfirmedBurnFactor          uint32 `json:"unconfirmed_burn_factor"`           // Burn factor for an unconfirmed transaction.
	UnconfirmedMaxTransactionSize  uint32 `json:"unconfirmed_max_transaction_size"`  // Maximum size for an unconfirmed transaction.
	UnconfirmedMaxDropletPrecision uint8  `json:"unconfirmed_max_droplet_precision"` // Maximum number of decimals allowed for an unconfirmed transaction.

	CreateBlockBurnFactor          uint32 `json:"create_block_burn_factor"`           // Burn factor to transactions when publishing blocks.
	CreateBlockMaxTransactionSize  uint32 `json:"create_block_max_transaction_size"`  // Maximum size of a transaction when publishing blocks.
	CreateBlockMaxDropletPrecision uint8  `json:"create_block_max_droplet_precision"` // Maximum number of decimals allowed for a transaction when publishing blocks.

	MaxBlockTransactionSize uint32 `json:"max_block_transaction_size"` // Maximum total size of transactions when publishing a block.
}

// DefaultProtocolParams returns default values for ProtocolParams.
func DefaultProtocolParams() ProtocolParams {
	return ProtocolParams{
		UnconfirmedBurnFactor:          10,
		UnconfirmedMaxTransactionSize:  5 * 1024 * 1024,
		UnconfirmedMaxDropletPrecision: 3,
		CreateBlockBurnFactor:          10,
		CreateBlockMaxTransactionSize:  5 * 1024 * 1024,
		CreateBlockMaxDropletPrecision: 3,
		MaxBlockTransactionSize:        5 * 1024 * 1024,
	}
}

// NodeParams defines the coin's default node parameters.
// TODO @evanlinjin: In the future, we may use the same network for different cx-chains.
// TODO: If that ever comes to light, we can remove these.
type NodeParams struct {
	Port               int      `json:"port"`                // Default port for wire protocol.
	WebInterfacePort   int      `json:"web_interface_port"`  // Default port for web interface.
	DefaultConnections []string `json:"default_connections"` // Default bootstrapping nodes (trusted).

	/* Parameters for user-generated transactions. */
	UserBurnFactor          uint64 `json:"user_burn_factor"`           // Inverse fraction of coin hours that must be burned (used when creating transactions).
	UserMaxTransactionSize  uint32 `json:"user_max_transaction_size"`  // Maximum size of user-created transactions (typically equal to the max size of a block).
	UserMaxDropletPrecision uint64 `json:"user_max_droplet_precision"` // Decimal precision of droplets (smallest coin unit).
}

// DefaultNodeParams returns the default values for NodeParams.
func DefaultNodeParams() NodeParams {
	return NodeParams{
		Port:             6001,
		WebInterfacePort: 6421,
		DefaultConnections: []string{
			"127.0.0.1:6001",
		},
		UserBurnFactor:          10,
		UserMaxTransactionSize:  32 * 1024,
		UserMaxDropletPrecision: 3,
	}
}

// ChainSpec represents the chain spec structure of version alpha.
type ChainSpec struct {
	SpecEra string `json:"spec_era"`

	ChainPubKey string `json:"chain_pubkey"` // Blockchain public key.
	TrackerAddr string `json:"tracker_addr"` // CX Tracker address.

	Protocol ProtocolParams `json:"protocol"` // Params that define the transaction protocol.
	Node     NodeParams     `json:"node"`     // Default params for a node of given coin (this may be removed in future eras).

	/* Identity Params */
	CoinName        string `json:"coin_name"`         // Coin display name (e.g. skycoin).
	CoinTicker      string `json:"coin_ticker"`       // Coin price ticker (e.g. SKY).

	/* Genesis Params */
	GenesisAddr       string `json:"genesis_address"`       // Genesis address (base58 representation).
	GenesisSig        string `json:"genesis_signature"`     // Genesis signature (hex representation).
	GenesisCoinVolume uint64 `json:"genesis_coin_volume"`   // Genesis coin volume.
	GenesisProgState  string `json:"genesis_program_state"` // Initial program state on genesis addr (hex representation).
	GenesisTimestamp  uint64 `json:"genesis_timestamp"`     // Timestamp of genesis block (in seconds, UTC time).

	/* Distribution Params */
	// TODO @evanlinjin: Figure out if these are needed for the time being.
	MaxCoinSupply uint64 `json:"max_coin_supply"` // Maximum coin supply.
	// InitialUnlockedCount      uint64   `json:"initial_unlocked_count"`       // Initial number of unlocked addresses.
	// UnlockAddressRate         uint64   `json:"unlock_address_rate"`          // Number of addresses to unlock per time interval.
	// UnlockAddressTimeInterval uint64   `json:"unlock_address_time_interval"` // Time interval (in seconds) in which addresses are unlocked. Once the InitialUnlockedCount is exhausted, UnlockAddressRate addresses will be unlocked per UnlockTimeInterval.
	// DistributionAddresses     []string `json:"distribution_addresses"`       // Addresses that receive coins.

	/* post-processed params */
	chainPK      cipher.PubKey
	genAddr      cipher.Address
	genSig       cipher.Sig
	genProgState []byte
	// distAddresses []cipher.Address // TODO @evanlinjin: May not be needed.
}

// New generates a new chain spec.
func New(coin, ticker string, chainSK cipher.SecKey, trackerAddr string, genesisAddr cipher.Address, genesisProgState []byte) (*ChainSpec, error) {
	coin = strings.ToLower(strings.Replace(coin, " ", "", -1))
	ticker = strings.ToUpper(strings.Replace(ticker, " ", "", -1))

	spec := &ChainSpec{
		SpecEra:     Era,
		ChainPubKey: "", // ChainPubKey is generated at a later step via generateAndSignGenesisBlock
		TrackerAddr: trackerAddr,
		Protocol:    DefaultProtocolParams(),
		Node:        DefaultNodeParams(),

		CoinName:        coin,
		CoinTicker:      ticker,

		GenesisAddr:       genesisAddr.String(),
		GenesisSig:        "", // GenesisSig is generated at a later step via generateAndSignGenesisBlock
		GenesisCoinVolume: 100e12,
		GenesisProgState:  progSEnc.EncodeToString(genesisProgState),
		GenesisTimestamp:  uint64(time.Now().UTC().Unix()),

		MaxCoinSupply: 1e8,
	}

	// Fill post-processed fields.
	if err := postProcess(spec, true); err != nil {
		return nil, err
	}

	// Generate genesis signature.
	block, err := generateGenesisBlock(spec)
	if err != nil {
		return nil, err
	}
	if err := signAndFill(spec, block, chainSK); err != nil {
		return nil, err
	}

	return spec, nil
}

// SpecHash returns the hashed spec object.
func (cs *ChainSpec) CXSpecHash() cipher.SHA256 {
	b, err := json.Marshal(cs)
	if err != nil {
		panic(err)
	}
	return cipher.SumSHA256(b)
}

// CXSpecEra returns the spec era string.
func (*ChainSpec) CXSpecEra() string { return Era }

// String prints an indented json representation of the chain spec.
func (cs *ChainSpec) String() string {
	b, err := json.MarshalIndent(cs, "", "\t")
	if err != nil {
		panic(err)
	}

	return string(b)
}

// Finalize finalizes the spec, providing the genesis public key and genesis
// signature.
func (cs *ChainSpec) Finalize(genesisSK cipher.SecKey) error {
	if err := postProcess(cs, true); err != nil {
		return err
	}

	block, err := generateGenesisBlock(cs)
	if err != nil {
		return err
	}

	if err := signAndFill(cs, block, genesisSK); err != nil {
		return err
	}

	return nil
}

// Check checks whether the spec is valid.
func (cs *ChainSpec) Check() error {
	if _, err := cs.ObtainGenesisBlock(); err != nil {
		return err
	}

	// TODO @evanlinjin: Implement more checks.

	return nil
}

// ObtainCoinName obtains the coin name of the spec.
func (cs *ChainSpec) ObtainCoinName() string { return cs.CoinName }

// ObtainCoinTicker obtains the coin ticker of the spec.
func (cs *ChainSpec) ObtainCoinTicker() string { return cs.CoinTicker }

// ObtainGenesisBlock generates a genesis block from the chain spec and verifies it.
// It returns an error if anything fails.
func (cs *ChainSpec) ObtainGenesisBlock() (*coin.Block, error) {
	if err := postProcess(cs, false); err != nil {
		return nil, err
	}

	block, err := generateGenesisBlock(cs)
	if err != nil {
		return nil, err
	}

	if err := cipher.VerifyPubKeySignedHash(cs.chainPK, cs.genSig, block.HashHeader()); err != nil {
		return nil, err
	}

	return block, nil
}

// ObtainGenesisAddr obtains the genesis address.
func (cs *ChainSpec) ObtainGenesisAddr() (cipher.Address, error) {
	return cipher.DecodeBase58Address(cs.GenesisAddr)
}

// ObtainGenesisProgState obtains the genesis program state.
func (cs *ChainSpec) ObtainGenesisProgState() ([]byte, error) {
	return progSEnc.DecodeString(cs.GenesisProgState)
}


// ObtainChainPubKey returns the processed chain public key.
func (cs *ChainSpec) ObtainChainPubKey() (cipher.PubKey, error) {
	return cipher.PubKeyFromHex(cs.ChainPubKey)
}

// ObtainWebInterfacePort returns the web interface port.
func (cs *ChainSpec) ObtainWebInterfacePort() (int, error) {
	return cs.Node.WebInterfacePort, nil
}

// PopulateParamsModule populates the params module within cx chain.
func (cs *ChainSpec) PopulateParamsModule() error {
	// TODO @evanlinjin: Figure out distribution.
	params.MainNetDistribution = params.Distribution{
		MaxCoinSupply:        cs.MaxCoinSupply,
		InitialUnlockedCount: 1,
		UnlockAddressRate:    0,
		UnlockTimeInterval:   0,
		Addresses:            []string{cs.GenesisAddr},
	}
	params.UserVerifyTxn = params.VerifyTxn{
		BurnFactor:          uint32(cs.Node.UserBurnFactor),
		MaxTransactionSize:  cs.Node.UserMaxTransactionSize,
		MaxDropletPrecision: uint8(cs.Node.UserMaxDropletPrecision),
	}
	params.InitFromEnv()

	return nil
}

// PopulateNodeConfig populates the node config with values from cx chain spec.
func (cs *ChainSpec) PopulateNodeConfig(conf *skycoin.NodeConfig) error {
	genesis, err := cs.ObtainGenesisBlock()
	if err != nil {
		return err
	}
	peerListURL := fmt.Sprintf("%s/peerlists/%s.txt", cs.TrackerAddr, genesis.HashHeader())

	conf.CoinName = cs.CoinName
	conf.PeerListURL = peerListURL
	conf.Port = cs.Node.Port
	conf.WebInterfacePort = cs.Node.WebInterfacePort
	conf.UnconfirmedVerifyTxn = params.VerifyTxn{
		BurnFactor:          cs.Protocol.UnconfirmedBurnFactor,
		MaxTransactionSize:  cs.Protocol.UnconfirmedMaxTransactionSize,
		MaxDropletPrecision: cs.Protocol.UnconfirmedMaxDropletPrecision,
	}
	conf.CreateBlockVerifyTxn = params.VerifyTxn{
		BurnFactor:          cs.Protocol.CreateBlockBurnFactor,
		MaxTransactionSize:  cs.Protocol.CreateBlockMaxTransactionSize,
		MaxDropletPrecision: cs.Protocol.CreateBlockMaxDropletPrecision,
	}
	conf.MaxBlockTransactionsSize = cs.Protocol.MaxBlockTransactionSize
	conf.GenesisSignatureStr = cs.GenesisSig
	conf.GenesisAddressStr = cs.GenesisAddr
	conf.BlockchainPubkeyStr = cs.ChainPubKey
	conf.GenesisTimestamp = cs.GenesisTimestamp
	conf.GenesisCoinVolume = cs.GenesisCoinVolume
	conf.DefaultConnections = cs.Node.DefaultConnections

	conf.Fiber = readable.FiberConfig{
		Name:            cs.CoinName,
		DisplayName:     cs.CoinName,
		Ticker:          cs.CoinTicker,
		CoinHoursName:   coinHoursName(cs.CoinName),
		CoinHoursTicker: coinHoursTicker(cs.CoinTicker),
		ExplorerURL:     "", // TODO @evanlinjin: CX Chain explorer?
	}

	if conf.DataDirectory == "" {
		conf.DataDirectory = "$HOME/.cxchain/" + cs.CoinName
	} else {
		conf.DataDirectory = strings.ReplaceAll(conf.DataDirectory, "{coin}", cs.CoinName)
	}

	return nil
}

/*
	<<< Helper functions >>>
*/

// postProcess fills post-process fields of chain spec.
// The 'allowEmpty'
func postProcess(cs *ChainSpec, allowEmpty bool) error {
	wrapErr := func(name string, err error) error {
		return fmt.Errorf("chain spec: failed to post-process '%s': %w", name, err)
	}
	var err error

	if !allowEmpty || cs.ChainPubKey != "" {
		if cs.chainPK, err = cipher.PubKeyFromHex(cs.ChainPubKey); err != nil {
			return wrapErr("chain_pubkey", err)
		}
	}
	if cs.genAddr, err = cipher.DecodeBase58Address(cs.GenesisAddr); err != nil {
		return wrapErr("genesis_address", err)
	}
	if !allowEmpty || cs.GenesisSig != "" {
		if cs.genSig, err = cipher.SigFromHex(cs.GenesisSig); err != nil {
			return wrapErr("genesis_signature", err)
		}
	}
	if cs.genProgState, err = progSEnc.DecodeString(cs.GenesisProgState); err != nil {
		return wrapErr("genesis_prog_state", err)
	}
	return nil
}

// generateGenesisBlock generates a genesis block from the chain spec with no checks.
func generateGenesisBlock(cs *ChainSpec) (*coin.Block, error) {
	block, err := coin.NewGenesisBlock(cs.genAddr, cs.GenesisCoinVolume, cs.GenesisTimestamp, cs.genProgState)
	if err != nil {
		return nil, fmt.Errorf("chain spec: %w", err)
	}
	return block, nil
}

// signAndFill fills the chain spec with block and block signature.
func signAndFill(cs *ChainSpec, block *coin.Block, sk cipher.SecKey) error {
	pk, err := cipher.PubKeyFromSecKey(sk)
	if err != nil {
		return err
	}

	blockSig, err := cipher.SignHash(block.HashHeader(), sk)
	if err != nil {
		return fmt.Errorf("failed to sign genesis block: %w", err)
	}

	cs.chainPK = pk
	cs.ChainPubKey = pk.Hex()
	cs.genSig = blockSig
	cs.GenesisSig = blockSig.Hex()

	return nil
}

// coinHoursName generates the coin hours name from given coin name.
func coinHoursName(coinName string) string {
	return fmt.Sprintf("%s coin hours", strings.ToLower(stripWhitespaces(coinName)))
}

// coinHoursTicker generates the coin hours ticker symbol from given coin ticker.
func coinHoursTicker(coinTicker string) string {
	return fmt.Sprintf("%s_CH", strings.ToUpper(stripWhitespaces(coinTicker)))
}

func stripWhitespaces(s string) string {
	out := make([]int32, 0, len(s))
	for _, c := range s {
		if unicode.IsSpace(c) {
			continue
		}
		out = append(out, c)
	}

	return string(out)
}