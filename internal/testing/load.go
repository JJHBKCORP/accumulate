package testing

import (
	"crypto/ed25519"
	"crypto/sha256"
	"fmt"
	"math/rand"
	"time"

	"github.com/AccumulateNetwork/accumulated/internal/relay"
	"github.com/AccumulateNetwork/accumulated/types"
	anon "github.com/AccumulateNetwork/accumulated/types/anonaddress"
	"github.com/AccumulateNetwork/accumulated/types/api/transactions"
	"github.com/AccumulateNetwork/accumulated/types/synthetic"
)

// Load
// Generate load in our test.  Create a bunch of transactions, and submit them.
func Load(txBouncer *relay.Relay, Origin ed25519.PrivateKey, walletCount, txCount int) (addrList []string, err error) {

	var wallet []*transactions.WalletEntry

	wallet = append(wallet, transactions.NewWalletEntry()) // wallet[0] is where we put 5000 ACME tokens
	wallet[0].Nonce = 1                                    // start the nonce at 1
	wallet[0].PrivateKey = Origin                          // Put the private key for the origin
	wallet[0].Addr = anon.GenerateAcmeAddress(Origin[32:]) // Generate the origin address

	for i := 0; i < walletCount; i++ { //                            create a 1000 addresses for anonymous token chains
		wallet = append(wallet, transactions.NewWalletEntry()) // create a new wallet entry
	}

	addrCountMap := make(map[string]int)
	for i := 0; i < txCount; i++ { // Make a bunch of transactions
		if i%200 == 0 {
			txBouncer.BatchSend()
			time.Sleep(200 * time.Millisecond)
		}
		const origin = 0
		randDest := rand.Int()%(len(wallet)-1) + 1                            // pick a destination address
		out := transactions.Output{Dest: wallet[randDest].Addr, Amount: 1000} // create the transaction output
		addrCountMap[wallet[randDest].Addr]++                                 // count the number of deposits to output
		send := transactions.NewTokenSend(wallet[origin].Addr, out)           // Create a send token transaction
		gtx := new(transactions.GenTransaction)                               // wrap in a GenTransaction
		gtx.SigInfo = new(transactions.SignatureInfo)                         // Get a Signature Info block
		gtx.Transaction = send.Marshal()                                      // add  send transaction
		gtx.SigInfo.URL = wallet[origin].Addr                                 // URL of source
		if err := gtx.SetRoutingChainID(); err != nil {                       // Routing ChainID is the tx source
			return nil, fmt.Errorf("failed to set routing chain ID: %v", err)
		}

		binaryGtx := gtx.TransactionHash() // Must sign the GenTransaction

		gtx.Signature = append(gtx.Signature, wallet[origin].Sign(binaryGtx))

		if resp, err := txBouncer.BatchTx(gtx); err != nil {
			return nil, fmt.Errorf("failed to send TX: %v", err)
		} else {
			if len(resp.Log) > 0 {
				fmt.Printf("<%d>%v<<\n", i, resp.Log)
			}
		}
	}
	txBouncer.BatchSend()
	for addr, ct := range addrCountMap {
		addrList = append(addrList, addr)
		_ = ct
		fmt.Printf("%s : %d\n", addr, ct*1000)
	}

	return addrList, nil
}

func RunLoadTest(txBouncer *relay.Relay, origin *ed25519.PrivateKey, walletCount, txCount int) (addrList []string, err error) {
	//use the public key of the bvc to make a sponsor address (this doesn't really matter right now, but need something so Identity of the BVC is good)
	adiSponsor := types.String(anon.GenerateAcmeAddress(origin.Public().(ed25519.PublicKey)))

	_, privateKey, _ := ed25519.GenerateKey(nil)
	//set destination url address
	destAddress := types.String(anon.GenerateAcmeAddress(privateKey.Public().(ed25519.PublicKey)))

	txid := sha256.Sum256([]byte("fake txid"))

	tokenUrl := types.String("dc/ACME")

	//create a fake synthetic deposit for faucet.
	deposit := synthetic.NewTokenTransactionDeposit(txid[:], &adiSponsor, &destAddress)
	amtToDeposit := int64(50000)                             //deposit 50k tokens
	deposit.DepositAmount.SetInt64(amtToDeposit * 100000000) // assume 8 decimal places
	deposit.TokenUrl = tokenUrl

	depData, err := deposit.MarshalBinary()
	if err != nil {
		return nil, fmt.Errorf("failed to marshal deposit: %v", err)
	}

	gtx := new(transactions.GenTransaction)
	gtx.SigInfo = new(transactions.SignatureInfo)
	gtx.Transaction = depData
	gtx.SigInfo.URL = *destAddress.AsString()
	gtx.ChainID = types.GetChainIdFromChainPath(destAddress.AsString())[:]
	gtx.Routing = types.GetAddressFromIdentity(destAddress.AsString())

	ed := new(transactions.ED25519Sig)
	gtx.SigInfo.Nonce = 1
	ed.PublicKey = privateKey[32:]
	err = ed.Sign(gtx.SigInfo.Nonce, privateKey, gtx.TransactionHash())
	if err != nil {
		return nil, fmt.Errorf("failed to sign TX: %v", err)
	}

	gtx.Signature = append(gtx.Signature, ed)

	_, err = txBouncer.SendTx(gtx)
	if err != nil {
		return nil, fmt.Errorf("failed to send TX: %v", err)
	}

	addresses, err := Load(txBouncer, privateKey, walletCount, txCount)
	if err != nil {
		return nil, err
	}

	addrList = append(addrList, *adiSponsor.AsString())
	addrList = append(addrList, *destAddress.AsString())
	addrList = append(addrList, addresses...)
	return addrList, nil
}