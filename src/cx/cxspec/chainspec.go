package cxspec

import (
	"fmt"
	"strings"
	"unicode"

	"github.com/skycoin/skycoin/src/cipher"

	"github.com/skycoin/cx-chains/src/coin"
)

// ObtainSpecEra obtains the spec era from a json-parsed result.
func ObtainSpecEra(t map[string]interface{}) string {
	s, ok := t["spec_era"].(string)
	if !ok {
		return ""
	}

	return s
}

// CoinHoursName generates the coin hours name from given coin name.
func CoinHoursName(coinName string) string {
	return fmt.Sprintf("%s coin hours", strings.ToLower(stripWhitespaces(coinName)))
}

// CoinHoursTicker generates the coin hours ticker symbol from given coin ticker.
func CoinHoursTicker(coinTicker string) string {
	return fmt.Sprintf("%s_CH", strings.ToUpper(stripWhitespaces(coinTicker)))
}

type TemplatePreparer func() map[string]interface{}

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

type ChainSpec interface {
	GenesisProgramState() []byte
	GenerateGenesisBlock() (*coin.Block, error)
	SpecHash() cipher.SHA256
	String() string
}
