package cxspec

import (
	"encoding/json"
	"fmt"

	"github.com/skycoin/cx-chains/src/cx/cxspec/alpha"
)

// Parse parses a chain spec from raw bytes.
func Parse(raw []byte) (ChainSpec, error) {
	era, err := ObtainSpecEra(raw)
	if err != nil {
		return nil, fmt.Errorf("failed to obtain spec era: %s", err)
	}

	var v ChainSpec

	switch era {
	case alpha.Era:
		// TODO @evanlinjin: Ensure this implements ChainSpec.
		v = new(alpha.ChainSpec)
	default:
		return nil, fmt.Errorf("unsupported spec era '%s'", era)
	}

	if err := json.Unmarshal(raw, &v); err != nil {
		return nil, fmt.Errorf("failed to parse spec of era '%s': %w", era, err)
	}

	return v, nil
}

// ObtainSpecEra obtains the spec era from a raw spec.
func ObtainSpecEra(raw []byte) (string, error) {
	var v struct {
		SpecEra string `json:"spec_era"`
	}

	if err := json.Unmarshal(raw, &v); err != nil {
		return "", err
	}

	return v.SpecEra, nil
}
