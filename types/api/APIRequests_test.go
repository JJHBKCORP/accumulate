package api_test

import (
	"crypto/sha256"
	"encoding/json"
	"testing"
	"time"

	"github.com/AccumulateNetwork/accumulate/internal/url"
	"github.com/AccumulateNetwork/accumulate/protocol"
	"github.com/AccumulateNetwork/accumulate/types"
	. "github.com/AccumulateNetwork/accumulate/types/api"
	"github.com/mitchellh/mapstructure"
	"github.com/stretchr/testify/require"
	"github.com/tendermint/tendermint/crypto/ed25519"
)

func createAdiTx(adiUrl string, pubkey []byte) (string, error) {
	data := &protocol.CreateIdentity{}

	data.Url = adiUrl
	keyhash := sha256.Sum256(pubkey)
	data.PublicKey = keyhash[:]

	ret, err := json.Marshal(data)
	if err != nil {
		return "", err
	}

	return string(ret), nil
}

func createToken(tokenUrl string) (string, error) {
	data := &protocol.CreateToken{}

	data.Url = tokenUrl
	data.Precision = 8
	data.Symbol = "ACME"
	ret, err := json.Marshal(data)
	if err != nil {
		return "", err
	}

	return string(ret), nil
}

func createVal(operator string, pubkey []byte, moniker string, identity string, website string, details string, rate uint64, maxR uint64, maxCr uint64) (string, error) {
	data := &protocol.CreateValidator{}

	data.PubKey = pubkey
	data.ValidatorAddress = operator
	data.Moniker = moniker
	data.Identity = identity
	data.Website = website 
	data.Details = details
	data.Commission.CommissionRates.Rate = rate
	data.Commission.CommissionRates.MaxRate = maxR
	data.Commission.CommissionRates.MaxChangeRate = maxCr

	ret, err := json.Marshal(data)
	if err != nil {
		return "", err
	}

	return string(ret), nil
}


func createTokenTx() (string, error) {
	tx := &protocol.SendTokens{}
	amt := uint64(1234)
	tx.AddRecipient(&url.URL{Authority: "redwagon", Path: "/AcmeAccount"}, amt)
	ret, err := json.Marshal(&tx)
	return string(ret), err
}

func createRequest(t *testing.T, adiUrl string, kp *ed25519.PrivKey, message string) []byte {
	req := &APIRequestRaw{}

	req.Tx = &APIRequestRawTx{}
	// make a raw json message.
	raw := json.RawMessage(message)

	//Set the message data. Making it a json.RawMessage will prevent go from unmarshalling it which
	//allows us to verify the signature against it.
	req.Tx.Data = &raw
	req.Tx.Signer = &Signer{}
	req.Tx.Signer.Nonce = uint64(time.Now().Unix())
	req.Tx.Origin = types.String(adiUrl)
	req.Tx.KeyPage = &APIRequestKeyPage{}
	req.Tx.KeyPage.Height = 1
	copy(req.Tx.Signer.PublicKey[:], kp.PubKey().Bytes())

	//form the ledger for signing
	ledger := types.MarshalBinaryLedgerAdiChainPath(adiUrl, []byte(message), int64(req.Tx.Signer.Nonce))

	//sign it...
	sig, err := kp.Sign(ledger)

	//store the signature
	copy(req.Tx.Sig[:], sig)

	//make the json for submission to the jsonrpc
	params, err := json.Marshal(&req)
	if err != nil {
		t.Fatal(err)
	}

	return params
}

func TestAPIRequest_Adi(t *testing.T) {
	kp := types.CreateKeyPair()

	adiUrl := "redwagon"

	message, err := createAdiTx(adiUrl, kp.PubKey().Bytes())
	if err != nil {
		t.Fatal(err)

	}
	params := createRequest(t, adiUrl, &kp, message)

	validate, err := protocol.NewValidator()
	require.NoError(t, err)

	req := &APIRequestRaw{}
	// unmarshal req
	if err = json.Unmarshal(params, &req); err != nil {
		t.Fatal(err)
	}

	// validate request
	if err = validate.Struct(req); err != nil {
		t.Fatal(err)
	}

	data := &protocol.CreateIdentity{}

	// parse req.tx.data
	err = mapstructure.Decode(req.Tx.Data, data)
	if err == nil {
		//in this case we are EXPECTING failure because the mapstructure doesn't decode the hex encoded strings from data
		t.Fatal(err)
	}

	rawreq := APIRequestRaw{}
	err = json.Unmarshal(params, &rawreq)
	if err != nil {
		t.Fatal(err)
	}

	err = json.Unmarshal(*rawreq.Tx.Data, data)
	if err != nil {
		t.Fatal(err)
	}

	// validate request data
	if err = validate.Struct(data); err != nil {
		//the data should have been unmarshalled correctly and the data is should be valid
		t.Fatal(err)
	}
}

func TestAPIRequest_CreateValidator(t *testing.T) {
	kp := types.CreateKeyPair()

	adiUrl := "redwagon"
	pubKey := kp.PubKey().Bytes()
	op := kp.PubKey().Address().String()
	mon := "mo"
	id := "lo"
	web := "www"
	det := "lol"
	r := uint64(100)
	rr := uint64(10000)
	rrr := uint64(100000)

	message, err := createVal(op , pubKey, mon, id, web, det, r, rr, rrr)
	if err != nil {
		t.Fatal(err)
	}
	params := createRequest(t, adiUrl, &kp, message)

	validate, err := protocol.NewValidator()
	require.NoError(t, err)

	req := &APIRequestRaw{}
	// unmarshal req
	if err = json.Unmarshal(params, &req); err != nil {
		t.Fatal(err)
	}

	// validate request
	if err = validate.Struct(req); err != nil {
		t.Fatal(err)
	}

	data := &protocol.CreateValidator{}

	// parse req.tx.data
	err = mapstructure.Decode(req.Tx.Data, data)
	if err == nil {
		//in this case we are EXPECTING failure because the mapstructure doesn't decode the hex encoded strings from data
		t.Fatal(err)
	}

	rawreq := APIRequestRaw{}
	err = json.Unmarshal(params, &rawreq)
	if err != nil {
		t.Fatal(err)
	}

	err = json.Unmarshal(*rawreq.Tx.Data, data)
	if err != nil {
		t.Fatal(err)
	}

	// validate request data
	if err = validate.Struct(data); err != nil {
		//the data should have been unmarshalled correctly and the data is should be valid
		t.Fatal(err)
	}
	t.Log(rawreq)
	t.Log(rawreq.Tx.Data)
}

func TestAPIRequest_TokenTx(t *testing.T) {
	kp := types.CreateKeyPair()

	adiUrl := "greentractor"

	message, err := createTokenTx()
	require.NoError(t, err)
	params := createRequest(t, adiUrl, &kp, message)

	validate, err := protocol.NewValidator()
	require.NoError(t, err)

	req := &APIRequestRaw{}
	// unmarshal req
	err = json.Unmarshal(params, &req)
	require.NoError(t, err)

	// validate request
	err = validate.Struct(req)
	require.NoError(t, err)

	tx, err := json.Marshal(&req)
	require.NoError(t, err)

	t.Logf("%s", tx)
}
