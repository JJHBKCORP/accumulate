package cmd

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/AccumulateNetwork/jsonrpc2/v15"
	"github.com/spf13/cobra"
	url2 "gitlab.com/accumulatenetwork/accumulate/internal/url"
	"gitlab.com/accumulatenetwork/accumulate/protocol"
)

var tokenCmd = &cobra.Command{
	Use:   "token",
	Short: "Issue and get tokens",
}

var tokenCmdGet = &cobra.Command{
	Use:   "get [url]",
	Short: "get token by URL",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		out, err := GetToken(args[0])
		printOutput(cmd, out, err)
	},
}

var tokenCmdCreate = &cobra.Command{
	Use:   "create [origin adi or lite url] [adi signer key name (if applicable)] [token url] [symbol] [precision (0 - 18)] [properties URL (optional)]",
	Short: "Create new token",
	Args:  cobra.MinimumNArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		var out string
		var err error
		if len(args) > 1 {
			out, err = CreateToken(args[0], args[1:])
		} else {
			fmt.Println("Usage:")
			PrintTokenCreate()
		}
		printOutput(cmd, out, err)
	},
}

var tokenCmdIssue = &cobra.Command{
	Use:   "issue [adi token url] [signer key name] [recipient url] [amount]",
	Short: "send tokens from a token url to a recipient",
	Args:  cobra.MinimumNArgs(4),
	Run: func(cmd *cobra.Command, args []string) {
		out, err := IssueTokenToRecipient(args[0], args[1:])
		printOutput(cmd, out, err)
	},
}

var tokenCmdBurn = &cobra.Command{
	Use:   "burn [adi or lite token account] [adi signer key name (if applicable)] [amount]",
	Short: "burn tokens",
	Args:  cobra.MinimumNArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		out, err := BurnTokens(args[0], args[1:])
		printOutput(cmd, out, err)
	},
}

func init() {
	tokenCmd.AddCommand(
		tokenCmdGet,
		tokenCmdCreate,
		tokenCmdIssue,
		tokenCmdBurn)
}

func PrintTokenGet() {
	fmt.Println("  accumulate token get [url] Get token by URL")
}

func PrintTokenCreate() {
	fmt.Println("  accumulate token create [origin adi url] [signer key name] [token url] [symbol] [precision (0 - 18)] [properties URL (optional)] 	Create new token")
	fmt.Println("  accumulate token create [origin lite url] [token url] [symbol] [precision (0 - 18)] [properties URL (optional)] 	Create new token")
}

func GetToken(url string) (string, error) {
	res, err := GetUrl(url)
	if err != nil {
		return "", err
	}

	return PrintChainQueryResponseV2(res)
}

func CreateToken(origin string, args []string) (string, error) {
	originUrl, err := url2.Parse(origin)
	if err != nil {
		return "", err
	}

	args, si, privKey, err := prepareSigner(originUrl, args)
	if err != nil {
		return "", err
	}

	if len(args) < 3 {
		return "", fmt.Errorf("insufficient number of arguments")
	}

	url := args[0]
	symbol := args[1]
	precision := args[2]
	var properties *url2.URL
	if len(args) > 3 {
		properties, err = url2.Parse(args[3])
		if err != nil {
			return "", fmt.Errorf("invalid properties url, %v", err)
		}
		res, err := GetUrl(properties.String())
		if err != nil {
			return "", fmt.Errorf("cannot query properties url, %v", err)
		}
		//TODO: make a better test for properties to make sure contents are valid, for now we just see if it is at least a data account
		if res.Type != protocol.AccountTypeDataAccount.String() {
			return "", fmt.Errorf("properties url is not a valid properties data account")
		}
	}

	prcsn, err := strconv.Atoi(precision)
	if err != nil {
		return "", err
	}

	u, err := url2.Parse(url)
	if err != nil {
		return "", err
	}

	params := protocol.CreateToken{}
	params.Url = u
	params.Symbol = symbol
	params.Precision = uint64(prcsn)
	params.Properties = properties

	res, err := dispatchTxRequest("create-token", &params, nil, originUrl, si, privKey)
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

func IssueTokenToRecipient(origin string, args []string) (string, error) {
	originUrl, err := url2.Parse(origin)
	if err != nil {
		return "", err
	}

	args, si, privKey, err := prepareSigner(originUrl, args)
	if err != nil {
		return "", err
	}

	if len(args) < 2 {
		return "", fmt.Errorf("insufficient number of parameters provided")
	}
	recipient, err := url2.Parse(args[0])
	if err != nil {
		return "", err
	}

	//query the token precision and reformat amount argument into a bigInt.
	amt, err := amountToBigInt(originUrl.String(), args[1])
	if err != nil {
		return "", err
	}

	params := protocol.IssueTokens{}
	params.Recipient = recipient
	params.Amount.Set(amt)

	res, err := dispatchTxRequest("issue-tokens", &params, nil, originUrl, si, privKey)
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

func BurnTokens(origin string, args []string) (string, error) {
	originUrl, err := url2.Parse(origin)
	if err != nil {
		return "", err
	}

	args, si, privKey, err := prepareSigner(originUrl, args)
	if err != nil {
		return "", err
	}

	if len(args) < 1 {
		return "", fmt.Errorf("amount to burn is not specified")
	}

	tokenUrl, err := GetTokenUrlFromAccount(originUrl)
	if err != nil {
		return "", fmt.Errorf("invalid token url was obtained from %s, %v", originUrl.String(), err)
	}

	//query the token precision and reformat amount argument into a bigInt.
	amt, err := amountToBigInt(tokenUrl.String(), args[0])
	if err != nil {
		return "", err
	}

	params := protocol.BurnTokens{}
	params.Amount.Set(amt)

	res, err := dispatchTxRequest("burn-tokens", &params, nil, originUrl, si, privKey)
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
