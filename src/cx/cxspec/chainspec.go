package cxspec

import (
	"fmt"

	"github.com/skycoin/skycoin/src/cipher"

	"github.com/skycoin/cx-chains/src/coin"
	"github.com/skycoin/cx-chains/src/cx/cxspec/alpha"
	"github.com/skycoin/cx-chains/src/skycoin"
)

const (
	specEraFieldName = "spec_era"
)

type ChainSpec interface {
	CXSpecHash() cipher.SHA256
	CXSpecEra() string
	String() string

	Finalize(genesisSK cipher.SecKey) error
	Check() error

	ObtainCoinName() string
	ObtainCoinTicker() string
	ObtainGenesisBlock() (*coin.Block, error)
	ObtainGenesisAddr() (cipher.Address, error)
	ObtainGenesisProgState() ([]byte, error)
	ObtainChainPubKey() (cipher.PubKey, error)
	ObtainWebInterfacePort() (int, error)

	PopulateParamsModule() error
	PopulateNodeConfig(conf *skycoin.NodeConfig) error
}

// TemplateSpecGenerator generates a template chain spec.
type TemplateSpecGenerator func() ChainSpec

// SpecFinalizer finalizes the chain spec.
type SpecFinalizer func(cs ChainSpec) error

// SignedChainSpec contains a chain spec alongside a valid signature.
type SignedChainSpec struct {
	Spec        ChainSpec `json:"spec"`
	GenesisHash string    `json:"genesis_hash,omitempty"`
	Sig         string    `json:"sig"` // hex representation of signature
}

// MakeSignedChainSpec generates a signed spec from a ChainSpec and secret key.
// Note that the secret key needs to be able to generate the ChainSpec's public
// key to be valid.
func MakeSignedChainSpec(spec ChainSpec, sk cipher.SecKey) (SignedChainSpec, error) {
	if err := spec.Check(); err != nil {
		return SignedChainSpec{}, fmt.Errorf("spec file failed to pass basic check: %w", err)
	}

	genesis, err := spec.ObtainGenesisBlock()
	if err != nil {
		return SignedChainSpec{}, fmt.Errorf("chain spec failed to generate genesis block: %w", err)
	}

	pk, err := cipher.PubKeyFromSecKey(sk)
	if err != nil {
		return SignedChainSpec{}, err
	}

	obtainedPK, err := spec.ObtainChainPubKey()
	if err != nil {
		return SignedChainSpec{}, fmt.Errorf("cannot obtain chain pk from spec: %w", err)
	}

	if pk != obtainedPK {
		return SignedChainSpec{}, fmt.Errorf("provided sk does not generate chain pk '%s'", obtainedPK)
	}

	sig, err := cipher.SignHash(spec.CXSpecHash(), sk)
	if err != nil {
		return SignedChainSpec{}, err
	}

	signedSpec := SignedChainSpec{
		Spec:        spec,
		GenesisHash: genesis.HashHeader().Hex(),
		Sig:         sig.Hex(),
	}

	return signedSpec, nil
}

// Verify checks the following:
// - Spec is of right era, has valid chain pk, and generates valid genesis block.
// - Signature is valid
func (ss *SignedChainSpec) Verify() error {
	const expectedEra = alpha.Era

	if era := ss.Spec.CXSpecEra(); era != expectedEra {
		return fmt.Errorf("unexpected chain spec era '%s' (expected '%s')",
			era, expectedEra)
	}

	if _, err := ss.Spec.ObtainGenesisBlock(); err != nil {
		return fmt.Errorf("chain spec failed to generate genesis block: %w", err)
	}

	sig, err := cipher.SigFromHex(ss.Sig)
	if err != nil {
		return fmt.Errorf("failed to decode spec signature: %w", err)
	}

	pk, err := ss.Spec.ObtainChainPubKey()
	if err != nil {
		return fmt.Errorf("failed to obtain chain pk: %w", err)
	}

	hash := ss.Spec.CXSpecHash()

	if err := cipher.VerifyPubKeySignedHash(pk, sig, hash); err != nil {
		return fmt.Errorf("failed to verify spec signature: %w", err)
	}

	return nil
}
