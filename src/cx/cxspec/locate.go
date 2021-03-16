package cxspec

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/skycoin/skycoin/src/cipher"
	"github.com/skycoin/skycoin/src/util/logging"
)

// LocPrefix determines the location type of the location string.
type LocPrefix string

// Locations types.
const (
	FileLoc    = LocPrefix("file")
	TrackerLoc = LocPrefix("tracker")
)

// Constants.
const (
	// DefaultSpecFilepath is the default cx spec filepath.
	// This is for internal use.
	DefaultSpecFilepath = "skycoin.chain_spec.json"

	// DefaultSpecLocStr is the default cx spec location string.
	DefaultSpecLocStr = string(FileLoc + ":" + DefaultSpecFilepath)

	// DefaultTrackerURL is the default cx tracker URL.
	DefaultTrackerURL = "https://cxt.skycoin.com"
)

// Possible errors when executing 'Locate'.
var (
	ErrEmptySpec        = errors.New("empty chain spec provided")
	ErrEmptyTracker     = errors.New("tracker is not provided")
	ErrInvalidLocPrefix = errors.New("invalid spec location prefix")
)

// LocateConfig contains flag values for Locate.
type LocateConfig struct {
	CXChain   string // CX Chain spec location string.
	CXTracker string // CX Tracker URL.

	Logger     logrus.FieldLogger
	HTTPClient *http.Client
}

// FillDefaults fills LocateConfig with default values.
func (c *LocateConfig) FillDefaults() {
	c.CXChain = DefaultSpecLocStr
	c.CXTracker = DefaultTrackerURL
	c.Logger = logging.MustGetLogger("spec_loc")
}

// DefaultLocateConfig returns the default LocateConfig set.
func DefaultLocateConfig() LocateConfig {
	var lc LocateConfig
	lc.FillDefaults()
	return lc
}

// SoftParse parses the OS args for the 'chain' flag.
// It is called 'soft' parse because the existence of non-defined flags does not
// result in failure.
func (c *LocateConfig) SoftParse(args []string) {
	if v, ok := obtainFlagValue(args, "chain"); ok {
		c.CXChain = v
	}
	if v, ok := obtainFlagValue(args, "tracker"); ok {
		c.CXTracker = v
	}
}

// RegisterFlags ensures that the 'help' menu contains the locate flags and that
// the flags are recognized.
func (c *LocateConfig) RegisterFlags(fs *flag.FlagSet) {
	var temp string
	fs.StringVar(&temp, "chain", c.CXChain, fmt.Sprintf("cx chain location. Prepend with '%s:' or '%s:' for spec location type.", FileLoc, TrackerLoc))
	fs.StringVar(&temp, "tracker", c.CXTracker, "CX Tracker `URL`.")
}

// TrackerClient generates a CX Tracker client based on the defined config.
func (c *LocateConfig) TrackerClient() *CXTrackerClient {
	return NewCXTrackerClient(c.Logger, c.HTTPClient, c.CXTracker)
}

// LocateWithConfig locates a spec with a given locate config.
func LocateWithConfig(ctx context.Context, conf *LocateConfig) (ChainSpec, error) {
	return Locate(ctx, conf.Logger, conf.TrackerClient(), conf.CXChain)
}

// Locate locates the chain spec given a 'loc' string.
// The 'loc' string is to be of format '<location-prefix>:<location>'.
// * <location-prefix> is 'tracker' if undefined.
// * <location> either specifies the cx chain's genesis hash (if
// <location-prefix> is 'tracker') or filepath of the spec file (if
// <location-prefix> is 'file').
func Locate(ctx context.Context, log logrus.FieldLogger, tracker *CXTrackerClient, loc string) (ChainSpec, error) {
	// Ensure logger is existent.
	if log == nil {
		log = logging.MustGetLogger("cxspec").WithField("func", "Locate")
	}

	prefix, suffix, err := splitLocString(loc)
	if err != nil {
		return ChainSpec{}, err
	}

	// Check location prefix (LocPrefix).
	switch prefix {
	case FileLoc:
		if suffix == "" {
			suffix = DefaultSpecFilepath
		}

		return ReadSpecFile(suffix)

	case TrackerLoc:
		// Check that 'tracker' is not nil.
		if tracker == nil {
			return ChainSpec{}, ErrEmptyTracker
		}

		// Obtain genesis hash from hex string.
		hash, err := cipher.SHA256FromHex(suffix)
		if err != nil {
			return ChainSpec{}, fmt.Errorf("invalid genesis hash provided '%s': %w", loc, err)
		}

		// Obtain spec from tracker.
		signedChainSpec, err := tracker.SpecByGenesisHash(ctx, hash)
		if err != nil {
			return ChainSpec{}, fmt.Errorf("chain spec not of genesis hash not found in tracker: %w", err)
		}

		// Verify again (no harm in doing it twice).
		if err := signedChainSpec.Verify(); err != nil {
			return ChainSpec{}, err
		}

		return signedChainSpec.Spec, nil

	default:
		return ChainSpec{}, fmt.Errorf("%w '%s'", ErrInvalidLocPrefix, prefix)
	}
}

/*
	<< Helper functions >>
*/

func splitLocString(loc string) (prefix LocPrefix, suffix string, err error) {
	loc = strings.TrimSpace(loc)
	if loc == "" {
		return "", "", ErrEmptySpec
	}

	locParts := strings.SplitN(loc, ":", 2)

	switch len(locParts) {
	case 1:
		locParts = append([]string{string(TrackerLoc)}, locParts...)
	case 2:
		// continue
	default:
		panic("internal error: Locate() should never return >2 location parts")
	}

	return LocPrefix(locParts[0]), locParts[1], nil
}

func obtainFlagValue(args []string, key string) (string, bool) {
	var (
		keyPrefix1 = "-" + key
		keyPrefix2 = keyPrefix1 + "="
	)

	for i, a := range args {
		// Standardize flag prefix to single '-'.
		if strings.HasPrefix(a, "--") {
			a = a[1:]
		}

		// If there is no '=', the flag value is the next arg.
		if a == "-"+key && i+1 < len(args) {
			return args[i+1], true
		}

		if strings.HasPrefix(a, keyPrefix2) {
			return strings.TrimPrefix(a, keyPrefix2), true
		}
	}

	return "", false
}
