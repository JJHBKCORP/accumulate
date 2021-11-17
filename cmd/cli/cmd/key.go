package cmd

import (
	"bytes"
	"crypto/ed25519"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/AccumulateNetwork/accumulate/protocol"
	"github.com/AccumulateNetwork/accumulate/types"
	"github.com/spf13/cobra"
	"github.com/tyler-smith/go-bip32"
	"github.com/tyler-smith/go-bip39"
)

var keyCmd = &cobra.Command{
	Use:   "key",
	Short: "Create and manage Keys for ADI Key Books, and Pages",
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) > 0 {
			switch arg := args[0]; arg {
			case "import":
				if len(args) == 3 {
					if args[1] == "lite" {
						ImportKey(args[2], "")
					} else {
						PrintKeyImport()
					}

				} else if len(args) > 3 {
					switch args[1] {
					case "mnemonic":
						ImportMnemonic(args[2:])
					case "private":
						ImportKey(args[2], args[3])
					case "public":
						//reserved for future use.
						fallthrough
					default:
						PrintKeyImport()
					}
				} else {
					PrintKeyImport()
				}
			case "export":
				if len(args) > 1 {
					switch args[1] {
					case "all":
						ExportKeys()
					case "seed":
						ExportSeed()
					case "private":
						if len(args) > 2 {
							ExportKey(args[2])
						} else {
							PrintKeyExport()
						}
					case "mnemonic":
						ExportMnemonic()
					default:
						PrintKeyExport()
					}
				} else {
					PrintKeyExport()
				}
			case "list":
				ListKeyPublic()
			case "generate":
				if len(args) > 1 {
					GenerateKey(args[1])
				} else {
					PrintKeyGenerate()
				}
			default:
				fmt.Println("Usage:")
				PrintKey()
			}
		} else {
			fmt.Println("Usage:")
			PrintKey()
		}

	},
}

type KeyResponse struct {
	Label      string `json:"name"`
	PrivateKey []byte `json:"privateKey"`
	PublicKey  []byte `json:"publicKey"`
	Seed       []byte `json:"seed"`
	Mnemonic   []byte `json:"mnemonic"`
}

func init() {
	rootCmd.AddCommand(keyCmd)
}

func PrintKeyPublic() {
	fmt.Println("  accumulate key list			List generated keys associated with the wallet")
}

func PrintKeyExport() {
	fmt.Println("  accumulate key export all			            export all keys in wallet")
	fmt.Println("  accumulate key export private [key name]			export the private key by key name")
	fmt.Println("  accumulate key export mnemonic		            export the mnemonic phrase if one was entered")
	fmt.Println("  accumulate key export seed                       export the seed generated from the mnemonic phrase")
}

func PrintKeyGenerate() {
	fmt.Println("  accumulate key generate [key name]     Generate a new key and give it a name in the wallet")
}

func PrintKeyImport() {
	fmt.Println("  accumulate key import mnemonic [mnemonic phrase...]     Import the mneumonic phrase used to generate keys in the wallet")
	fmt.Println("  accumulate key import private [private key hex] [key name]      Import a key and give it a name in the wallet")
	fmt.Println("  accumulate key import lite [private key hex]       Import a key as an anonymous address")
}

func PrintKey() {
	PrintKeyGenerate()
	PrintKeyPublic()
	PrintKeyImport()

	PrintKeyExport()
}

func pubKeyFromString(s string) ([]byte, error) {
	var pubKey types.Bytes32
	if len(s) != 64 {
		return nil, fmt.Errorf("invalid public key or wallet key name")
	}
	i, err := hex.Decode(pubKey[:], []byte(s))

	if err != nil {
		return nil, err
	}

	if i != 32 {
		return nil, fmt.Errorf("invalid public key")
	}

	return pubKey[:], nil
}

func getPublicKey(s string) ([]byte, error) {
	var pubKey types.Bytes32
	privKey, err := LookupByLabel(s)

	if err != nil {
		b, err := pubKeyFromString(s)
		if err != nil {
			return nil, fmt.Errorf("unable to resolve public key %s,%v", s, err)
		}
		pubKey.FromBytes(b)
	} else {
		pubKey.FromBytes(privKey[32:])
	}

	return pubKey[:], nil
}

func LookupByLabel(label string) ([]byte, error) {
	pubKey, err := Db.Get(BucketLabel, []byte(label))
	if err != nil {
		return nil, fmt.Errorf("valid key not found for %s", label)
	}
	return LookupByPubKey(pubKey)
}

func LookupByPubKey(pubKey []byte) ([]byte, error) {
	return Db.Get(BucketKeys, pubKey)
}

func GenerateKey(label string) {
	if _, err := strconv.ParseInt(label, 10, 64); err == nil {
		log.Fatal("key name cannot be a number")
	}

	privKey, err := GeneratePrivateKey()

	if err != nil {
		log.Fatal(err)
	}

	pubKey := privKey[32:]

	if label == "" {
		ltu, err := protocol.AnonymousAddress(pubKey, protocol.AcmeUrl().String())
		if err != nil {
			log.Fatal("unable to create lite account")
		}
		label = ltu.String()
	}

	_, err = LookupByLabel(label)
	if err == nil {
		log.Fatal(fmt.Errorf("key already exists for key name %s", label))
	}

	err = Db.Put(BucketKeys, pubKey, privKey)
	if err != nil {
		log.Fatal(err)
	}

	err = Db.Put(BucketLabel, []byte(label), pubKey)
	if err != nil {
		log.Fatal(err)
	}

	if WantJsonOutput {
		a := KeyResponse{}
		a.Label = label
		a.PublicKey = pubKey
		dump, err := json.Marshal(a)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Fprintf(os.Stderr, "%s\n", string(dump))
	} else {
		fmt.Fprintf(os.Stderr, "%s :\t%x", label, pubKey)
	}
}

func ListKeyPublic() {
	fmt.Fprintf(os.Stderr, "%s\t\t\t\t\t\t\t\tKey name\n", "Public Key")
	b, err := Db.GetBucket(BucketLabel)
	if err != nil {
		log.Fatal(err)
	}

	for _, v := range b.KeyValueList {
		fmt.Fprintf(os.Stderr, "%x\t%s\n", v.Value, v.Key)
	}
}

func FindLabelFromPubKey(pubKey []byte) (lab string, err error) {
	b, err := Db.GetBucket(BucketLabel)
	if err != nil {
		return lab, err
	}

	for _, v := range b.KeyValueList {
		if bytes.Equal(v.Value, pubKey) {
			lab = string(v.Key)
			break
		}
	}

	if lab == "" {
		err = fmt.Errorf("key name not found for %x", pubKey)
	}
	return lab, err
}

// ImportKey will import the private key and assign it to the label
func ImportKey(pkhex string, label string) {

	var pk ed25519.PrivateKey

	token, err := hex.DecodeString(pkhex)
	if err != nil {
		log.Fatal(err)
	}

	if len(token) == 32 {
		pk = ed25519.NewKeyFromSeed(token)
	} else {
		pk = token
	}

	if label == "" {
		lt, err := protocol.AnonymousAddress(pk[32:], protocol.AcmeUrl().String())
		if err != nil {
			log.Fatalf("no label specified and cannot import as lite account")
		}
		label = lt.String()
	}

	_, err = LookupByLabel(label)
	if err == nil {
		log.Fatal("key name is already being used")
	}

	_, err = LookupByPubKey(pk[32:])
	lab := "not found"
	if err == nil {
		b, _ := Db.GetBucket(BucketLabel)
		if b != nil {
			for k, v := range b.KeyValueList {
				if bytes.Equal(v.Value, pk[32:]) {
					lab = string(k)
					break
				}
			}
			log.Fatalf("private key already exists in wallet by key name of %s", lab)
		}
	}

	err = Db.Put(BucketKeys, pk[32:], pk)
	if err != nil {
		log.Fatal(err)
	}

	err = Db.Put(BucketLabel, []byte(label), pk[32:])
	if err != nil {
		log.Fatal(err)
	}

	if WantJsonOutput {
		a := KeyResponse{}
		a.Label = label
		a.PublicKey = pk[32:]
		dump, err := json.Marshal(a)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Fprintf(os.Stderr, "%s\n", string(dump))
	} else {
		fmt.Fprintf(os.Stderr, "\tname\t\t:%s\n\tpublic key\t:%x\n", label, pk[32:])
	}
}

func ExportKey(label string) {
	pk, err := LookupByLabel(label)
	if err != nil {
		pubk, err := pubKeyFromString(label)
		if err != nil {
			log.Fatalf("no private key found for key name %s", label)
		}
		pk, err = LookupByPubKey(pubk)
		if err != nil {
			log.Fatalf("no private key found for key name %s", label)
		}
		label, err = FindLabelFromPubKey(pubk)
		if err != nil {
			log.Fatalf("no private key found for key name %s", label)
		}
	}

	if WantJsonOutput {
		a := KeyResponse{}
		a.Label = label
		a.PrivateKey = pk[:32]
		a.PublicKey = pk[32:]
		dump, err := json.Marshal(a)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Fprintf(os.Stderr, "%s\n", string(dump))
	} else {
		fmt.Fprintf(os.Stderr, "\tname\t\t:\t%s\n\tprivate key\t:\t%x\n\tpublic key\t:\t%x\n", label, pk[:32], pk[32:])
	}
}

func GeneratePrivateKey() (privKey []byte, err error) {
	seed, err := lookupSeed()

	if err != nil {
		//if private key seed doesn't exist, just create a key
		_, privKey, err = ed25519.GenerateKey(nil)
		if err != nil {
			return nil, err
		}
	} else {
		//if we do have a seed, then create a new key
		masterKey, _ := bip32.NewMasterKey(seed)

		newKey, err := masterKey.NewChildKey(uint32(getKeyCountAndIncrement()))
		if err != nil {
			return nil, err
		}
		privKey = ed25519.NewKeyFromSeed(newKey.Key)
	}
	return
}

func getKeyCountAndIncrement() (count uint32) {

	ct, err := Db.Get(BucketMnemonic, []byte("count"))
	if ct != nil {
		count = binary.LittleEndian.Uint32(ct)
	}

	ct = make([]byte, 8)
	binary.LittleEndian.PutUint32(ct, count+1)
	err = Db.Put(BucketMnemonic, []byte("count"), ct)
	if err != nil {
		log.Fatal(err)
	}

	return count
}

func lookupSeed() (seed []byte, err error) {
	seed, err = Db.Get(BucketMnemonic, []byte("seed"))
	if err != nil {
		log.Fatal("mnemonic seed doesn't exist")
	}

	return seed, nil
}

func ImportMnemonic(mnemonic []string) {
	mns := strings.Join(mnemonic, " ")

	if !bip39.IsMnemonicValid(mns) {
		log.Fatal("invalid mnemonic provided")
	}

	// Generate a Bip32 HD wallet for the mnemonic and a user supplied password
	seed := bip39.NewSeed(mns, "")

	root, err := Db.Get(BucketMnemonic, []byte("seed"))
	if len(root) != 0 {
		log.Fatal("mnemonic seed phrase already exists within wallet")
	}

	err = Db.Put(BucketMnemonic, []byte("seed"), seed)
	if err != nil {
		log.Fatalf("DB: seed write error, %v", err)
	}

	err = Db.Put(BucketMnemonic, []byte("phrase"), []byte(mns))
	if err != nil {
		log.Fatalf("DB: phrase write error %s", err)
	}

	println("mnemonic import successful")
}

func ExportKeys() {
	b, err := Db.GetBucket(BucketKeys)
	if err != nil {
		log.Fatal(err)
	}

	for _, v := range b.KeyValueList {
		label, err := FindLabelFromPubKey(v.Key)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: Cannot find label for public key %x\n", v.Key)
		} else {
			ExportKey(label)
		}
	}
}

func ExportSeed() {
	seed, err := Db.Get(BucketMnemonic, []byte("seed"))
	if err != nil {
		log.Fatal("mnemonic seed not found")
	}
	if WantJsonOutput {
		a := KeyResponse{}
		a.Seed = seed
		dump, err := json.Marshal(a)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Fprintf(os.Stderr, "%s\n", string(dump))
	} else {
		fmt.Fprintf(os.Stderr, " seed: %x\n", seed)
	}
}

func ExportMnemonic() {
	phrase, err := Db.Get(BucketMnemonic, []byte("phrase"))
	if err != nil {
		log.Fatal(err)
	}
	if WantJsonOutput {
		a := KeyResponse{}
		a.Mnemonic = phrase
		dump, err := json.Marshal(a)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Fprintf(os.Stderr, "%s\n", string(dump))
	} else {
		fmt.Fprintf(os.Stderr, "mnemonic phrase: %s\n", string(phrase))
	}
}
