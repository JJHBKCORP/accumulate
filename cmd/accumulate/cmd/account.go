package cmd

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"errors"
	"fmt"
	"io/ioutil"
	"log"

	"github.com/AccumulateNetwork/jsonrpc2/v15"
	"github.com/mdp/qrterminal"
	"github.com/spf13/cobra"
	"gitlab.com/accumulatenetwork/accumulate/internal/api/v2"
	url2 "gitlab.com/accumulatenetwork/accumulate/internal/url"
	"gitlab.com/accumulatenetwork/accumulate/protocol"
)

func init() {
	accountCmd.AddCommand(
		accountGetCmd,
		accountCreateCmd,
		accountQrCmd,
		accountGenerateCmd,
		accountListCmd,
		accountRestoreCmd)

	accountCreateCmd.AddCommand(
		accountCreateTokenCmd,
		accountCreateDataCmd)

	accountCreateDataCmd.AddCommand(
		accountCreateDataLiteCmd)

	accountCreateTokenCmd.Flags().BoolVar(&flagAccount.Scratch, "scratch", false, "Create a scratch token account")
	accountCreateDataCmd.Flags().BoolVar(&flagAccount.Scratch, "scratch", false, "Create a scratch data account")
	accountCreateDataCmd.Flags().BoolVar(&flagAccount.Lite, "lite", false, "Create a lite data account")
}

var flagAccount = struct {
	Lite    bool
	Scratch bool
}{}

var accountCmd = &cobra.Command{
	Use:   "account",
	Short: "Create and get token accounts",
}

var accountGetCmd = &cobra.Command{
	Use:   "get [url]",
	Short: "Get an account by URL",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		out, err := GetAccount(args[0])
		printOutput(cmd, out, err)
	},
}

var accountCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create an account",
	Args:  cobra.MinimumNArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("Deprecation Warning!\nTo create a token account, in future please specify either \"token\" or \"data\"\n\n")
		//this will be removed in future release and replaced with usage: PrintAccountCreate()
		out, err := CreateAccount(cmd, args[0], args[1:])
		printOutput(cmd, out, err)
	},
}

var accountCreateTokenCmd = &cobra.Command{
	Use:   "token [actor adi] [signing key name] [key index (optional)] [key height (optional)] [new token account url] [tokenUrl] [keyBook (optional)]",
	Short: "Create an ADI token account",
	Args:  cobra.MinimumNArgs(4),
	Run: func(cmd *cobra.Command, args []string) {
		out, err := CreateAccount(cmd, args[0], args[1:])
		printOutput(cmd, out, err)
	},
}

var accountCreateDataCmd = &cobra.Command{
	Use:   "data",
	Short: "Create a data account",
	Run: func(cmd *cobra.Command, args []string) {
		var out string
		var err error
		if flagAccount.Lite {
			if len(args) < 2 {
				PrintDataLiteAccountCreate()
				return
			}
			out, err = CreateLiteDataAccount(args[0], args[1:])
		} else {
			if len(args) < 3 {
				PrintDataAccountCreate()
				PrintDataLiteAccountCreate()
				return
			}
			out, err = CreateDataAccount(args[0], args[1:])
		}
		printOutput(cmd, out, err)
	},
}

var accountCreateDataLiteCmd = &cobra.Command{
	Use:   "lite",
	Short: "Create a lite data account",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("Deprecation Warning!\nTo create a lite data account, use `accumulate account create data --lite ...`\n\n")
		if len(args) < 2 {
			PrintDataLiteAccountCreate()
			return
		}
		out, err := CreateLiteDataAccount(args[0], args[1:])
		printOutput(cmd, out, err)
	},
}

var accountQrCmd = &cobra.Command{
	Use:   "qr [url]",
	Short: "Display QR code for lite token account URL",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		out, err := QrAccount(args[0])
		printOutput(cmd, out, err)
	},
}

var accountGenerateCmd = &cobra.Command{
	Use:   "generate",
	Short: "Generate random lite token account",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, _ []string) {
		out, err := GenerateAccount()
		printOutput(cmd, out, err)
	},
}

var accountListCmd = &cobra.Command{
	Use:   "list",
	Short: "Display all lite token accounts",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, _ []string) {
		out, err := ListAccounts()
		printOutput(cmd, out, err)
	},
}

var accountRestoreCmd = &cobra.Command{
	Use:   "restore",
	Short: "Restore old lite token accounts",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, _ []string) {
		out, err := RestoreAccounts()
		printOutput(cmd, out, err)
	},
}

func GetAccount(url string) (string, error) {
	res, err := GetUrl(url)
	if err != nil {
		return "", err
	}

	if res.Type != protocol.AccountTypeTokenAccount.String() && res.Type != protocol.AccountTypeLiteTokenAccount.String() &&
		res.Type != protocol.AccountTypeDataAccount.String() && res.Type != protocol.AccountTypeLiteDataAccount.String() {
		return "", fmt.Errorf("expecting token account or data account but received %v", res.Type)
	}

	return PrintChainQueryResponseV2(res)
}

func QrAccount(s string) (string, error) {
	u, err := url2.Parse(s)
	if err != nil {
		return "", fmt.Errorf("%q is not a valid Accumulate URL: %v\n", s, err)
	}

	b := bytes.NewBufferString("")
	qrterminal.GenerateWithConfig(u.String(), qrterminal.Config{
		Level:          qrterminal.M,
		Writer:         b,
		HalfBlocks:     true,
		BlackChar:      qrterminal.BLACK_BLACK,
		BlackWhiteChar: qrterminal.BLACK_WHITE,
		WhiteChar:      qrterminal.WHITE_WHITE,
		WhiteBlackChar: qrterminal.WHITE_BLACK,
		QuietZone:      2,
	})

	r, err := ioutil.ReadAll(b)
	return string(r), err
}

//CreateAccount account create url labelOrPubKeyHex height index tokenUrl keyBookUrl
func CreateAccount(cmd *cobra.Command, origin string, args []string) (string, error) {
	u, err := url2.Parse(origin)
	if err != nil {
		_ = cmd.Usage()
		return "", err
	}

	args, si, privKey, err := prepareSigner(u, args)
	if err != nil {
		return "", err
	}
	if len(args) < 2 {
		return "", fmt.Errorf("not enough arguments")
	}

	accountUrl, err := url2.Parse(args[0])
	if err != nil {
		_ = cmd.Usage()
		return "", fmt.Errorf("invalid account url %s", args[0])
	}
	if u.Authority != accountUrl.Authority {
		return "", fmt.Errorf("account url to create (%s) doesn't match the authority adi (%s)", accountUrl.Authority, u.Authority)
	}
	tok, err := url2.Parse(args[1])
	if err != nil {
		return "", fmt.Errorf("invalid token url")
	}

	var keybook *url2.URL
	if len(args) > 2 {
		keybook, err = url2.Parse(args[2])
		if err != nil {
			return "", fmt.Errorf("invalid key book url")
		}
	}

	//make sure this is a valid token account
	req := new(api.GeneralQuery)
	req.Url = tok
	resp := new(api.ChainQueryResponse)
	token := protocol.TokenIssuer{}
	resp.Data = &token
	err = Client.RequestAPIv2(context.Background(), "query", req, resp)
	if err != nil || resp.Type != protocol.AccountTypeTokenIssuer.String() {
		return "", fmt.Errorf("invalid token type %v", err)
	}

	tac := protocol.CreateTokenAccount{}
	tac.Url = accountUrl
	tac.TokenUrl = tok
	tac.KeyBookUrl = keybook
	tac.Scratch = flagAccount.Scratch

	res, err := dispatchTxRequest("create-token-account", &tac, nil, u, si, privKey)
	if err != nil {
		return "", err
	}

	if !TxNoWait && TxWait > 0 {
		_, err := waitForTxn(res.TransactionHash, TxWait)
		if err != nil {
			var rpcErr jsonrpc2.Error
			if errors.As(err, &rpcErr) {
				return PrintJsonRpcError(err)
			}
			return "", err
		}
	}

	return ActionResponseFrom(res).Print()
}

func GenerateAccount() (string, error) {
	return GenerateKey("")
}

func ListAccounts() (string, error) {
	b, err := Db.GetBucket(BucketLabel)
	if err != nil {
		//no accounts so nothing to do...
		return "", err
	}
	var out string
	for _, v := range b.KeyValueList {
		lt, err := protocol.LiteTokenAddress(v.Value, protocol.AcmeUrl().String())
		if err != nil {
			continue
		}
		l, _ := LabelForLiteTokenAccount(lt.String())
		if l == string(v.Key) {
			out += fmt.Sprintf("%s\n", lt)
		}
	}
	//TODO: this probably should also list out adi accounts as well
	return out, nil
}

func RestoreAccounts() (out string, err error) {
	anon, err := Db.GetBucket(BucketAnon)
	if err == nil {
		for _, v := range anon.KeyValueList {
			u, err := url2.Parse(string(v.Key))
			if err != nil {
				out += fmt.Sprintf("%q is not a valid URL\n", v.Key)
			}
			if u != nil {
				key, _, err := protocol.ParseLiteTokenAddress(u)
				if err != nil {
					out += fmt.Sprintf("%q is not a valid lite account: %v\n", v.Key, err)
				} else if key == nil {
					out += fmt.Sprintf("%q is not a lite account\n", v.Key)
				}
			}

			label, _ := LabelForLiteTokenAccount(string(v.Key))
			v.Key = []byte(label)

			privKey := ed25519.PrivateKey(v.Value)
			pubKey := privKey.Public().(ed25519.PublicKey)
			out += fmt.Sprintf("Converting %s : %x\n", v.Key, pubKey)

			err = Db.Put(BucketLabel, v.Key, pubKey)
			if err != nil {
				log.Fatal(err)
			}
			err = Db.Put(BucketKeys, pubKey, privKey)
			if err != nil {
				return "", err
			}
			err = Db.DeleteBucket(BucketAnon)
			if err != nil {
				return "", err
			}
		}
	}

	//fix the labels... there can be only one key one label.
	//should not have multiple labels to the same public key
	labelz, err := Db.GetBucket(BucketLabel)
	if err != nil {
		//nothing to do...
		return
	}
	for _, v := range labelz.KeyValueList {
		label, isLite := LabelForLiteTokenAccount(string(v.Key))
		if isLite {
			//if we get here, then that means we have a bogus label.
			bogusLiteLabel := string(v.Key)
			//so check to see if it is in our regular key bucket
			otherPubKey, err := Db.Get(BucketLabel, []byte(label))
			if err != nil {
				//key isn't found, so let's add it
				out += fmt.Sprintf("Converting %s to %s : %x\n", v.Key, label, v.Value)
				//so it doesn't exist, map the good label to the public key
				err = Db.Put(BucketLabel, []byte(label), v.Value)
				if err != nil {
					return "", err
				}

				//now delete the bogus label
				err = Db.Delete(BucketLabel, []byte(bogusLiteLabel))
				if err != nil {
					return "", err
				}
			} else {
				//ok so it does exist, now need to know if public key is the same, it is
				//an error if they don't match so warn user
				if !bytes.Equal(v.Value, otherPubKey) {
					out += fmt.Sprintf("public key stored for %v, doesn't match what is expected for a lite account: %s (%x != %x)\n",
						bogusLiteLabel, label, v.Value, otherPubKey)
				} else {
					//key isn't found, so let's add it
					out += fmt.Sprintf("Removing duplicate %s / %s : %x\n", v.Key, label, v.Value)
					//now delete the bogus label
					err = Db.Delete(BucketLabel, []byte(bogusLiteLabel))
					if err != nil {
						return "", err
					}
				}
			}
		}
	}
	return out, nil
}
