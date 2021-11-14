package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	url2 "github.com/AccumulateNetwork/accumulated/internal/url"
	"github.com/AccumulateNetwork/accumulated/protocol"
	"github.com/AccumulateNetwork/accumulated/types"
	acmeapi "github.com/AccumulateNetwork/accumulated/types/api"
	"github.com/AccumulateNetwork/accumulated/types/api/transactions"
	"github.com/boltdb/bolt"

	"github.com/spf13/cobra"
)

var adiCmd = &cobra.Command{
	Use:   "adi",
	Short: "Create and manage ADI",
	Run: func(cmd *cobra.Command, args []string) {

		if len(args) > 0 {
			switch arg := args[0]; arg {
			case "get":
				if len(args) > 1 {
					GetADI(args[1])
				} else {
					fmt.Println("Usage:")
					PrintADIGet()
				}
			case "list":
				ListADIs()

			case "directory":

				if len(args) > 1 {
					GetAdiDirectory(args[1])
				} else {
					PrintAdiDirectory()
				}
			case "create":
				if len(args) > 3 {
					NewADI(args[1], args[2:])
				} else {
					fmt.Println("Usage:")
					PrintADICreate()
				}
			default:
				fmt.Println("Usage:")
				PrintADI()
			}
		} else {
			fmt.Println("Usage:")
			PrintADI()
		}

	},
}

func init() {
	rootCmd.AddCommand(adiCmd)
}

func PrintADIGet() {
	fmt.Println("  accumulate adi get [URL]			Get existing ADI by URL")
}

func PrintADICreate() {
	fmt.Println("  accumulate adi create [actor-lite-account] [adi url to create] [public-key or key name] [key-book-name (optional)] [key-page-name (optional)]  Create new ADI from lite account")
	fmt.Println("  accumulate adi create [actor-adi-url] [wallet signing key name] [key index (optional)] [key height (optional)] [adi url to create] [public key or wallet key name] [key book url (optional)] [key page url (optional)] Create new ADI for another ADI")
}

func PrintADIImport() {
	fmt.Println("  accumulate adi import [adi-url] [private-key]	Import Existing ADI")
}

func PrintAdiDirectory() {
	fmt.Println("  accumulate adi directory [url] 		Get directory of URL's associated with an ADI")
}

func GetAdiDirectory(actor string) {

	u, err := url2.Parse(actor)
	if err != nil {
		PrintCredits()
		log.Fatal(err)
	}

	var res acmeapi.APIDataResponse

	params := acmeapi.APIRequestURL{}

	params.URL = types.String(u.String())

	if err := Client.Request(context.Background(), "get-directory", params, &res); err != nil {
		PrintJsonRpcError(err)
	}

	PrintQueryResponse(&res)
}

func PrintADI() {
	PrintADIGet()
	PrintAdiDirectory()
	PrintADICreate()
	PrintADIImport()
}

func GetADI(url string) {

	var res acmeapi.APIDataResponse

	params := acmeapi.APIRequestURL{}
	params.URL = types.String(url)

	if err := Client.Request(context.Background(), "adi", params, &res); err != nil {
		PrintJsonRpcError(err)
	}

	PrintQueryResponse(&res)
	//str, err := json.Marshal(res)
	//if err != nil {
	//	log.Fatal(err)
	//}
	//
	//fmt.Println(string(str))
}

func NewADIFromADISigner(actor *url2.URL, args []string) {
	var si *transactions.SignatureInfo
	var privKey []byte
	var err error

	args, si, privKey, err = prepareSigner(actor, args)
	if err != nil {
		log.Fatal(err)
	}

	var adiUrl string
	var book string
	var page string

	if len(args) == 0 {
		log.Fatal("insufficient number of command line arguments")
	}

	if len(args) > 1 {
		adiUrl = args[0]
	}
	if len(args) < 2 {
		log.Fatalf("invalid number of arguments")
	}

	pubKey, err := getPublicKey(args[1])
	if err != nil {
		pubKey, err = pubKeyFromString(args[1])
		if err != nil {
			log.Fatal(fmt.Errorf("key %s, does not exist in wallet, nor is it a valid public key", args[1]))
		}
	}

	if len(args) > 3 {
		book = args[3]
	}

	if len(args) > 4 {
		page = args[4]
	}

	u, err := url2.Parse(adiUrl)
	if err != nil {
		log.Fatalf("invalid adi url %s, %v", adiUrl, err)
	}

	idc := &protocol.IdentityCreate{}
	idc.Url = u.Authority
	idc.PublicKey = pubKey
	idc.KeyBookName = book
	idc.KeyPageName = page

	data, err := json.Marshal(idc)
	if err != nil {
		log.Fatal(err)
	}

	dataBinary, err := idc.MarshalBinary()
	if err != nil {
		log.Fatal(err)
	}

	nonce := uint64(time.Now().Unix())
	params, err := prepareGenTx(data, dataBinary, actor, si, privKey, nonce)
	if err != nil {
		log.Fatal(err)
	}

	var res acmeapi.APIDataResponse
	if err := Client.Request(context.Background(), "adi-create", params, &res); err != nil {
		PrintJsonRpcError(err)
	}

	ar := ActionResponse{}
	err = json.Unmarshal(*res.Data, &ar)
	if err != nil {
		log.Fatal("error unmarshalling create adi result")
	}
	ar.Print()

	//todo: turn around and query the ADI and store the results.
	err = Db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("adi"))
		if b != nil {
			b.Put([]byte(u.Authority), pubKey)
		} else {
			return fmt.Errorf("DB: %s", err)
		}
		return nil
	})

}

// NewADI create a new ADI from a sponsored account.
func NewADI(actor string, params []string) {

	u, err := url2.Parse(actor)
	if err != nil {
		log.Fatal(err)
	}

	NewADIFromADISigner(u, params[:])
}

func ListADIs() {

	err := Db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("adi"))
		c := b.Cursor()
		for k, v := c.First(); k != nil; k, _ = c.Next() {
			fmt.Printf("%s : %s \n", k, string(v))
		}
		return nil
	})
	if err != nil {
		log.Fatal(err)
	}

}