/*
Package testsuite is the cipher testdata testsuite
*/
package testsuite

import (
	"encoding/base64"
	"errors"
	"fmt"

	"github.com/amherag/skycoin/src/cipher"
	secp256k1 "github.com/amherag/skycoin/src/cipher/secp256k1-go"
)

// InputTestDataJSON contains hashes to be signed
type InputTestDataJSON struct {
	Hashes []string `json:"hashes"`
}

// KeysTestDataJSON contains address, public key,  secret key and list of signatures
type KeysTestDataJSON struct {
	Address        string   `json:"address"`
	BitcoinAddress string   `json:"bitcoin_address"`
	Secret         string   `json:"secret"`
	Public         string   `json:"public"`
	Signatures     []string `json:"signatures,omitempty"`
}

// SeedTestDataJSON contains data generated by Seed
type SeedTestDataJSON struct {
	Seed string             `json:"seed"`
	Keys []KeysTestDataJSON `json:"keys"`
}

// InputTestData contains hashes to be signed
type InputTestData struct {
	Hashes []cipher.SHA256
}

// ToJSON converts InputTestData to InputTestDataJSON
func (d *InputTestData) ToJSON() *InputTestDataJSON {
	hashes := make([]string, len(d.Hashes))
	for i, h := range d.Hashes {
		hashes[i] = h.Hex()
	}

	return &InputTestDataJSON{
		Hashes: hashes,
	}
}

// InputTestDataFromJSON converts InputTestDataJSON to InputTestData
func InputTestDataFromJSON(d *InputTestDataJSON) (*InputTestData, error) {
	hashes := make([]cipher.SHA256, len(d.Hashes))
	for i, h := range d.Hashes {
		var err error
		hashes[i], err = cipher.SHA256FromHex(h)
		if err != nil {
			return nil, err
		}
	}

	return &InputTestData{
		Hashes: hashes,
	}, nil
}

// KeysTestData contains address, public key,  secret key and list of signatures
type KeysTestData struct {
	Address        cipher.Address
	BitcoinAddress cipher.BitcoinAddress
	Secret         cipher.SecKey
	Public         cipher.PubKey
	Signatures     []cipher.Sig
}

// ToJSON converts KeysTestData to KeysTestDataJSON
func (k *KeysTestData) ToJSON() *KeysTestDataJSON {
	sigs := make([]string, len(k.Signatures))
	for i, s := range k.Signatures {
		sigs[i] = s.Hex()
	}

	return &KeysTestDataJSON{
		Address:        k.Address.String(),
		BitcoinAddress: k.BitcoinAddress.String(),
		Secret:         k.Secret.Hex(),
		Public:         k.Public.Hex(),
		Signatures:     sigs,
	}
}

// KeysTestDataFromJSON converts KeysTestDataJSON to KeysTestData
func KeysTestDataFromJSON(d *KeysTestDataJSON) (*KeysTestData, error) {
	addr, err := cipher.DecodeBase58Address(d.Address)
	if err != nil {
		return nil, err
	}

	btcAddr, err := cipher.DecodeBase58BitcoinAddress(d.BitcoinAddress)
	if err != nil {
		return nil, err
	}

	s, err := cipher.SecKeyFromHex(d.Secret)
	if err != nil {
		return nil, err
	}

	p, err := cipher.PubKeyFromHex(d.Public)
	if err != nil {
		return nil, err
	}

	var sigs []cipher.Sig
	if d.Signatures != nil {
		sigs = make([]cipher.Sig, len(d.Signatures))
		for i, s := range d.Signatures {
			var err error
			sigs[i], err = cipher.SigFromHex(s)
			if err != nil {
				return nil, err
			}
		}
	}

	return &KeysTestData{
		Address:        addr,
		BitcoinAddress: btcAddr,
		Secret:         s,
		Public:         p,
		Signatures:     sigs,
	}, nil
}

// SeedTestData contains data generated by Seed
type SeedTestData struct {
	Seed []byte
	Keys []KeysTestData
}

// ToJSON converts SeedTestData to SeedTestDataJSON
func (s *SeedTestData) ToJSON() *SeedTestDataJSON {
	keys := make([]KeysTestDataJSON, len(s.Keys))
	for i, k := range s.Keys {
		kj := k.ToJSON()
		keys[i] = *kj
	}

	return &SeedTestDataJSON{
		Seed: base64.StdEncoding.EncodeToString(s.Seed),
		Keys: keys,
	}
}

// SeedTestDataFromJSON converts SeedTestDataJSON to SeedTestData
func SeedTestDataFromJSON(d *SeedTestDataJSON) (*SeedTestData, error) {
	seed, err := base64.StdEncoding.DecodeString(d.Seed)
	if err != nil {
		return nil, err
	}

	keys := make([]KeysTestData, len(d.Keys))
	for i, kj := range d.Keys {
		k, err := KeysTestDataFromJSON(&kj)
		if err != nil {
			return nil, err
		}
		keys[i] = *k
	}

	return &SeedTestData{
		Seed: seed,
		Keys: keys,
	}, nil
}

// ValidateSeedData validates the provided SeedTestData against the current cipher library.
// inputData is required if SeedTestData contains signatures
func ValidateSeedData(seedData *SeedTestData, inputData *InputTestData) error {
	keys := cipher.MustGenerateDeterministicKeyPairs(seedData.Seed, len(seedData.Keys))
	if len(seedData.Keys) != len(keys) {
		return errors.New("cipher.GenerateDeterministicKeyPairs generated an unexpected number of keys")
	}

	for i, s := range keys {
		if s == (cipher.SecKey{}) {
			return errors.New("secret key is null")
		}
		if seedData.Keys[i].Secret != s {
			return errors.New("generated secret key does not match provided secret key")
		}

		p := cipher.MustPubKeyFromSecKey(s)
		if p == (cipher.PubKey{}) {
			return errors.New("public key is null")
		}
		if seedData.Keys[i].Public != p {
			return errors.New("derived public key does not match provided public key")
		}

		addr1 := cipher.AddressFromPubKey(p)
		if addr1 == (cipher.Address{}) {
			return errors.New("address is null")
		}
		if seedData.Keys[i].Address != addr1 {
			return errors.New("derived address does not match provided address")
		}

		addr2 := cipher.MustAddressFromSecKey(s)
		if addr1 != addr2 {
			return errors.New("cipher.AddressFromPubKey and cipher.AddressFromSecKey generated different addresses")
		}

		btcAddr1 := cipher.BitcoinAddressFromPubKey(p)
		if btcAddr1 == (cipher.BitcoinAddress{}) {
			return errors.New("bitcoin address is null")
		}
		if seedData.Keys[i].BitcoinAddress != btcAddr1 {
			return errors.New("derived bitcoin address does not match provided bitcoin address")
		}

		btcAddr2 := cipher.MustBitcoinAddressFromSecKey(s)
		if btcAddr1 != btcAddr2 {
			return errors.New("cipher.BitcoinAddressFromPubKey and cipher.BitcoinAddressFromSecKey generated different addresses")
		}

		validSec := secp256k1.VerifySeckey(s[:])
		if validSec != 1 {
			return errors.New("secp256k1.VerifySeckey failed")
		}

		validPub := secp256k1.VerifyPubkey(p[:])
		if validPub != 1 {
			return errors.New("secp256k1.VerifyPubkey failed")
		}

		if inputData == nil && len(seedData.Keys[i].Signatures) != 0 {
			return errors.New("seed data contains signatures but input data was not provided")
		}

		if inputData != nil {
			if len(seedData.Keys[i].Signatures) != len(inputData.Hashes) {
				return errors.New("Number of signatures in seed data does not match number of hashes in input data")
			}

			for j, h := range inputData.Hashes {
				sig := seedData.Keys[i].Signatures[j]
				if sig == (cipher.Sig{}) {
					return errors.New("provided signature is null")
				}

				err := cipher.VerifyPubKeySignedHash(p, sig, h)
				if err != nil {
					return fmt.Errorf("cipher.VerifyPubKeySignedHash failed: %v", err)
				}

				err = cipher.VerifyAddressSignedHash(addr1, sig, h)
				if err != nil {
					return fmt.Errorf("cipher.VerifyAddressSignedHash failed: %v", err)
				}

				err = cipher.VerifySignedHash(sig, h)
				if err != nil {
					return fmt.Errorf("cipher.VerifySignedHash failed: %v", err)
				}

				p2, err := cipher.PubKeyFromSig(sig, h)
				if err != nil {
					return fmt.Errorf("cipher.PubKeyFromSig failed: %v", err)
				}

				if p != p2 {
					return errors.New("public key derived from signature does not match public key derived from secret")
				}

				sig2 := cipher.MustSignHash(h, s)
				if sig2 == (cipher.Sig{}) {
					return errors.New("created signature is null")
				}

				// NOTE: signatures are not deterministic, they use a nonce,
				// so we don't compare the generated sig to the provided sig
			}
		}
	}

	return nil
}
