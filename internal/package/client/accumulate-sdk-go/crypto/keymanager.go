package crypto

import (
	"bytes"
	"crypto/ed25519"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/AccumulateNetwork/accumulate/internal/url"
	"github.com/AccumulateNetwork/accumulate/protocol"
	"github.com/AccumulateNetwork/accumulate/smt/common"
	"github.com/AccumulateNetwork/accumulate/types/api/transactions"
	"github.com/tyler-smith/go-bip39"
)

const (
	defaultBIP39Passphrase = ""
)

type KeyManager interface {
	Generate() (string, PrivKey)
	Sign(nonce uint64, privateKey []byte, hash []byte) error

	ExportPrivateKey(name string) (string, error)
	ImportPrivateKey(name string, privateKey string) (PrivKey, err error)

	ExportPublicKey() PubKey

	ImportMnemonic(mnemonic []string) (string, error)
	ExportMnemonic() (string, error)
}

type keyManager struct {
	addr *url.URL
	privateKey PrivKey
	mnemonic []string
}


func NewKeyManager() KeyManager {
	return &keyManager{}
}

func NewMnemonicManager(mnemonic []string) (KeyManager, error) {
	k := keyManager{}

	err := k.recoveryFromMnemonic(mnemonic)
	return &k, err
}


func (k *keyManager) Generate() (pubKey ed25519.PublicKey, privKey ed25519.PrivateKey) {
	//	return k.mnemonic, k.privateKey
	_, privKey, err := ed25519.GenerateKey(nil)
	if err != nil {
		panic(err)
	}
	return 
}

func (k *keyManager) GetPrivKey() PrivKey {
	return k.privateKey
}

func (k *keyManager) GetAddr() string {
	return k.addr.String()
}


func (k *keyManager) Sign(nonce uint64, privateKey []byte, hash []byte)  error {
	e := new(transactions.ED25519Sig)
	e.Nonce = nonce
	nonceHash := append(common.Uint64Bytes(nonce), hash...)
	sig := ed25519.Sign(privateKey, nonceHash)
	
	e.PublicKey = append([]byte{}, privateKey[32:]...)
	if !bytes.Equal(privateKey, e.PublicKey) {
		return fmt.Errorf("Signature does not match")
	}
	e.Signature = sig
	return nil
}


func (k *keyManager) recoveryFromMnemonic(mnemonic []string) error {
	m := strings.Join(mnemonic, " ")
	
	if !bip39.IsMnemonicValid(m) {
		return fmt.Errorf("Invalid mnemonic")
	}

	seed := bip39.NewSeed(m, "")

	k.mnemonic = mnemonic


	return nil
}

func (k *keyManager) ImportPrivateKey(name string, privateKey string) (PrivKey, error) {
	var p ed25519.PrivateKey

	token, err := hex.DecodeString(privateKey)
	if err != nil {
		return nil, err
	}

	if len(token) == 32 {
		p = ed25519.NewKeyFromSeed(token)
	} else {
		p = token
	}

	if name == "" {
		lt, err := protocol.LiteAddress(p[32:], protocol.AcmeUrl().String())
		if err != nil {
			return nil, err
		}
		name = lt.String()
}

	return k.privateKey, nil

}

func (k *keyManager) ExportPublicKey() PubKey {
	return k.privateKey.PubKey()
}