package cmd

import (
	"context"
	"crypto/ed25519"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	url2 "github.com/AccumulateNetwork/accumulate/internal/url"
	"github.com/AccumulateNetwork/accumulate/protocol"
	"github.com/AccumulateNetwork/accumulate/types/api/response"
	"github.com/mdp/qrterminal"

	"github.com/AccumulateNetwork/accumulate/types"
	acmeapi "github.com/AccumulateNetwork/accumulate/types/api"
	"github.com/spf13/cobra"
)

var accountCmd = &cobra.Command{
	Use:   "account",
	Short: "Create and get token accounts",
	Run: func(cmd *cobra.Command, args []string) {

		if len(args) > 0 {
			switch arg := args[0]; arg {
			case "get":
				if len(args) > 1 {
					GetAccount(args[1])
				} else {
					fmt.Println("Usage:")
					PrintAccountGet()
				}
			case "create":
				if len(args) > 3 {
					CreateAccount(args[1], args[2:])
				} else {
					fmt.Println("Usage:")
					PrintAccountCreate()
				}
			case "qr":
				if len(args) > 1 {
					QrAccount(args[1])
				} else {
					fmt.Println("Usage:")
					PrintAccountQr()
				}
			case "generate":
				GenerateAccount()
			case "list":
				ListAccounts()
			case "restore":
				RestoreAccounts()
			default:
				fmt.Println("Usage:")
				PrintAccount()
			}
		} else {
			fmt.Println("Usage:")
			PrintAccount()
		}

	},
}

func init() {
	rootCmd.AddCommand(accountCmd)
}

func PrintAccountGet() {
	fmt.Println("  accumulate account get [url]			Get anon token account by URL")
}

func PrintAccountQr() {
	fmt.Println("  accumulate account qr [url]			Display QR code for lite account URL")
}

func PrintAccountGenerate() {
	fmt.Println("  accumulate account generate			Generate random lite token account")
}

func PrintAccountList() {
	fmt.Println("  accumulate account list			Display all anon token accounts")
}

func PrintAccountRestore() {
	fmt.Println("  accumulate account restore			Restore old anon token accounts")
}

func PrintAccountCreate() {
	fmt.Println("  accumulate account create [actor adi] [signing key name] [key index (optional)] [key height (optional)] [token account url] [tokenUrl] [keyBook]	Create a token account for an ADI")
}

func PrintAccountImport() {
	fmt.Println("  accumulate account import [private-key]	Import anon token account from private key hex")
}

func PrintAccountExport() {
	fmt.Println("  accumulate account export [url]		Export private key hex of anon token account")
}

func PrintAccount() {
	PrintAccountGet()
	PrintAccountQr()
	PrintAccountGenerate()
	PrintAccountList()
	PrintAccountRestore()
	PrintAccountCreate()
	PrintAccountImport()
	PrintAccountExport()
}

func GetAccount(url string) {
	var res acmeapi.APIDataResponse

	params := acmeapi.APIRequestURL{}
	params.URL = types.String(url)

	if err := Client.Request(context.Background(), "token-account", params, &res); err != nil {
		PrintJsonRpcError(err)
	}

	PrintQueryResponse(&res)
}

func QrAccount(s string) {
	u, err := url2.Parse(s)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%q is not a valid Accumulate URL: %v\n", s, err)
	}

	qrterminal.GenerateWithConfig(u.String(), qrterminal.Config{
		Level:          qrterminal.M,
		Writer:         os.Stdout,
		HalfBlocks:     true,
		BlackChar:      qrterminal.BLACK_BLACK,
		BlackWhiteChar: qrterminal.BLACK_WHITE,
		WhiteChar:      qrterminal.WHITE_WHITE,
		WhiteBlackChar: qrterminal.WHITE_BLACK,
		QuietZone:      2,
	})
}

//account create adiActor labelOrPubKeyHex height index tokenUrl keyBookUrl
func CreateAccount(url string, args []string) {

	actor, err := url2.Parse(url)
	if err != nil {
		PrintAccountCreate()
		log.Fatal(err)
	}

	args, si, privKey, err := prepareSigner(actor, args)
	if len(args) < 3 {
		PrintAccountCreate()
		log.Fatal("insufficient number of command line arguments")
	}

	accountUrl, err := url2.Parse(args[0])
	if err != nil {
		PrintAccountCreate()
		log.Fatalf("invalid account url %s", args[0])
	}
	if actor.Authority != accountUrl.Authority {
		log.Fatalf("account url to create (%s) doesn't match the authority adi (%s)", accountUrl.Authority, actor.Authority)
	}
	tok, err := url2.Parse(args[1])
	if err != nil {
		log.Fatal("invalid token url")
	}

	var keybook string
	if len(args) > 2 {
		kbu, err := url2.Parse(args[2])
		if err != nil {
			log.Fatal("invalid key book url")
		}
		keybook = kbu.String()
	}

	//make sure this is a valid token account
	tokenJson := Get(tok.String())
	token := response.Token{}
	err = json.Unmarshal([]byte(tokenJson), &token)
	if err != nil {
		PrintAccountCreate()
		log.Fatal(fmt.Errorf("invalid token type %v", err))
	}

	tac := &protocol.TokenAccountCreate{}
	tac.Url = accountUrl.String()
	tac.TokenUrl = tok.String()
	tac.KeyBookUrl = keybook

	binaryData, err := tac.MarshalBinary()
	if err != nil {
		log.Fatal(err)
	}

	jsonData, err := json.Marshal(&tac)
	if err != nil {
		log.Fatal(err)
	}

	nonce := uint64(time.Now().Unix())

	params, err := prepareGenTx(jsonData, binaryData, actor, si, privKey, nonce)
	if err != nil {
		log.Fatal(err)
	}

	var res acmeapi.APIDataResponse
	if err := Client.Request(context.Background(), "token-account-create", params, &res); err != nil {
		//todo: if we fail, then we need to remove the adi from storage or keep it and try again later...
		log.Fatal(err)
	}

	ar := ActionResponse{}
	err = json.Unmarshal(*res.Data, &ar)
	if err != nil {
		log.Fatal("error unmarshalling account create result")
	}
	ar.Print()
}

func GenerateAccount() {
	GenerateKey("")
}

func ListAccounts() {

	b, err := Db.GetBucket(BucketLabel)
	if err != nil {
		//no accounts so nothing to do...
		return
	}
	for _, v := range b.KeyValueList {
		lt, err := protocol.AnonymousAddress(v.Value, protocol.AcmeUrl().String())
		if err != nil {
			continue
		}
		if lt.String() == string(v.Key) {
			fmt.Printf("%s\n", v.Key)
		}
	}
	//TODO: this probably should also list out adi accounts as well
}

func RestoreAccounts() {
	anon, err := Db.GetBucket(BucketAnon)
	if err != nil {
		//no anon accounts so nothing to do...
		return
	}
	for _, v := range anon.KeyValueList {
		u, err := url2.Parse(string(v.Key))
		if err != nil {
			fmt.Fprintf(os.Stderr, "%q is not a valid URL\n", v.Key)
		}
		key, _, err := protocol.ParseAnonymousAddress(u)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%q is not a valid lite account: %v\n", v.Key, err)
		} else if key == nil {
			fmt.Fprintf(os.Stderr, "%q is not a lite account\n", v.Key)
		}

		privKey := ed25519.PrivateKey(v.Value)
		pubKey := privKey.Public().(ed25519.PublicKey)
		fmt.Printf("Converting %s : %x\n", v.Key, pubKey)

		err = Db.Put(BucketLabel, v.Key, pubKey)
		if err != nil {
			log.Fatal(err)
		}
		err = Db.Put(BucketKeys, pubKey, privKey)
		if err != nil {
			log.Fatal(err)
		}
		err = Db.DeleteBucket(BucketAnon)
		if err != nil {
			log.Fatal(err)
		}
	}
}
