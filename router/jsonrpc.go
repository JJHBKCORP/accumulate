package router

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"

	"github.com/AccumulateNetwork/accumulated/networks"
	"github.com/AccumulateNetwork/accumulated/types"
	anon "github.com/AccumulateNetwork/accumulated/types/anonaddress"
	acmeapi "github.com/AccumulateNetwork/accumulated/types/api"
	"github.com/AccumulateNetwork/accumulated/types/api/transactions"
	"github.com/AccumulateNetwork/accumulated/types/synthetic"

	//"github.com/AccumulateNetwork/accumulated/blockchain/validator"
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/AccumulateNetwork/jsonrpc2/v15"
	"github.com/go-playground/validator/v10"
	"github.com/gorilla/mux"
)

type API struct {
	port      int
	validate  *validator.Validate
	query     *Query
	txBouncer *networks.Bouncer
}

// StartAPI starts new JSON-RPC server
func StartAPI(port int, q *Query, txBouncer *networks.Bouncer) *API {

	// fmt.Printf("Starting JSON-RPC API at http://localhost:%d\n", port)

	api := &API{}
	api.port = port
	api.validate = validator.New()
	api.query = q
	api.txBouncer = txBouncer

	methods := jsonrpc2.MethodMap{
		// URL
		"get": api.getData,

		// ADI
		"adi":        api.getADI,
		"adi-create": api.createADI,

		// token
		"token":                api.getToken,
		"token-create":         api.createToken,
		"token-account":        api.getTokenAccount,
		"token-account-create": api.createTokenAccount,
		"token-tx":             api.getTokenTx,
		"token-tx-create":      api.createTokenTx,
		"faucet":               api.faucet,
	}

	apiHandler := jsonrpc2.HTTPRequestHandler(methods, log.New(os.Stdout, "", 0))

	apiRouter := mux.NewRouter().StrictSlash(true)
	apiRouter.HandleFunc("/v1", apiHandler)

	proxyRouter := mux.NewRouter().StrictSlash(true)
	proxyRouter.HandleFunc(`/url/{url:[a-zA-Z0-9=\.\-\_\~\!\$\&\'\(\)\*\+\,\;\=\:\@\/]+}`, proxyHandler)
	proxyRouter.HandleFunc(`/tx/{tx:[a-fA-F0-9]+}`, proxyHandler)

	// start JSON RPC API
	go func() {
		log.Fatal(http.ListenAndServe(":"+strconv.Itoa(port), apiRouter))
	}()

	// start REST proxy for JSON RPC API
	go func() {
		log.Fatal(http.ListenAndServe(":"+strconv.Itoa(port+1), proxyRouter))
	}()

	return api

}

// getData returns Accumulate Object by URL
func (api *API) getData(_ context.Context, params json.RawMessage) interface{} {

	var err error
	var resp interface{}
	req := &acmeapi.APIRequestGet{}

	if err = json.Unmarshal(params, &req); err != nil {
		return NewValidatorError(err)
	}

	// validate URL or Hash
	if req.URL != "" {
		if err = api.validate.StructPartial(req, "URL"); err != nil {
			return NewValidatorError(err)
		}
		resp, err = api.query.GetChainState(req.URL.AsString())
	} else {
		if err = api.validate.StructPartial(req, "Hash"); err != nil {
			return NewValidatorError(err)
		}
		// need to define if we need tokenaccount or not
		resp, err = api.query.GetTokenTx(req.Hash[:])
	}

	if err != nil {
		return NewAccumulateError(err)
	}

	return resp
}

// getADI returns ADI info
func (api *API) getADI(_ context.Context, params json.RawMessage) interface{} {

	var err error
	req := &acmeapi.APIRequestGet{}

	if err = json.Unmarshal(params, &req); err != nil {
		return NewValidatorError(err)
	}

	// validate URL
	if err = api.validate.Struct(req); err != nil {
		return NewValidatorError(err)
	}

	// Tendermint integration here
	resp, err := api.query.GetAdi(req.URL.AsString())

	if err != nil {
		return NewAccumulateError(err)
	}

	return resp
}

// createADI creates ADI
func (api *API) createADI(_ context.Context, params json.RawMessage) interface{} {

	var err error
	req := &acmeapi.APIRequestRaw{}
	data := &acmeapi.ADI{}

	// unmarshal req
	if err = json.Unmarshal(params, &req); err != nil {
		return NewValidatorError(err)
	}

	// validate request
	if err = api.validate.Struct(req); err != nil {
		return NewValidatorError(err)
	}

	// parse req.tx.data
	if err = json.Unmarshal(*req.Tx.Data, &data); err != nil {
		return NewValidatorError(err)
	}

	// validate request data
	if err = api.validate.Struct(data); err != nil {
		return NewValidatorError(err)
	}

	// Tendermint integration here
	var payload types.Bytes
	if payload, err = data.MarshalBinary(); err != nil {
		return NewValidatorError(err)
	}

	ret := api.sendTx(req, payload)
	ret.Type = "tokenTx"
	return ret
}

// getToken returns Token info
func (api *API) getToken(_ context.Context, params json.RawMessage) interface{} {

	var err error
	req := &acmeapi.APIRequestGet{}

	if err = json.Unmarshal(params, &req); err != nil {
		return NewValidatorError(err)
	}

	// validate URL
	if err = api.validate.Struct(req); err != nil {
		return NewValidatorError(err)
	}

	//query tendermint
	resp, err := api.query.GetToken(req.URL.AsString())

	if err != nil {
		return NewAccumulateError(err)
	}

	return resp

}

// createToken creates Token
func (api *API) createToken(_ context.Context, params json.RawMessage) interface{} {

	var err error
	req := &acmeapi.APIRequestRaw{}
	data := &acmeapi.Token{}

	// unmarshal req
	if err = json.Unmarshal(params, &req); err != nil {
		return NewValidatorError(err)
	}

	// validate request
	if err = api.validate.Struct(req); err != nil {
		return NewValidatorError(err)
	}

	// parse req.tx.data
	if err = json.Unmarshal(*req.Tx.Data, &data); err != nil {
		return NewValidatorError(err)
	}

	// validate request data
	if err = api.validate.Struct(data); err != nil {
		return NewValidatorError(err)
	}

	// Tendermint integration here
	var payload types.Bytes
	if payload, err = data.MarshalBinary(); err != nil {
		return NewValidatorError(err)
	}

	ret := api.sendTx(req, payload)
	ret.Type = "token"
	return ret
}

// getTokenAccount returns Token Account info
func (api *API) getTokenAccount(_ context.Context, params json.RawMessage) interface{} {

	var err error
	req := &acmeapi.APIRequestGet{}

	if err = json.Unmarshal(params, &req); err != nil {
		return NewValidatorError(err)
	}

	// validate URL
	if err = api.validate.Struct(req); err != nil {
		return NewValidatorError(err)
	}

	// Tendermint integration here
	taResp, err := api.query.GetTokenAccount(req.URL.AsString())
	if err != nil {
		return NewValidatorError(err)
	}

	return taResp

}

func (api *API) sendTx(req *acmeapi.APIRequestRaw, payload []byte) *acmeapi.APIDataResponse {
	genTx, err := acmeapi.NewAPIRequest(&req.Sig, req.Tx.Signer, req.Tx.Timestamp, payload)

	ret := &acmeapi.APIDataResponse{}
	var msg json.RawMessage

	if err != nil {
		msg = []byte(fmt.Sprintf("{\"error\":\"%v\"}", err))
		ret.Data = &msg
		return ret
	}
	resp, err := api.txBouncer.SendTx(genTx)
	if err != nil {
		msg = []byte(fmt.Sprintf("{\"error\":\"%v\"}", err))
		ret.Data = &msg
		return ret
	}

	msg = []byte(fmt.Sprintf("{\"txid\":\"%x\",\"log\":\"%s\"}", genTx.TransactionHash(), resp.Log))
	ret.Data = &msg
	return ret
}

// createTokenAccount creates Token Account
func (api *API) createTokenAccount(_ context.Context, params json.RawMessage) interface{} {

	var err error
	req := &acmeapi.APIRequestRaw{}
	data := &acmeapi.TokenAccount{}

	// unmarshal req
	if err = json.Unmarshal(params, &req); err != nil {
		return NewValidatorError(err)
	}

	// validate request
	if err = api.validate.Struct(req); err != nil {
		return NewValidatorError(err)
	}

	// parse req.tx.data
	if err = json.Unmarshal(*req.Tx.Data, &data); err != nil {
		return NewValidatorError(err)
	}

	// validate request data
	if err = api.validate.Struct(data); err != nil {
		return NewValidatorError(err)
	}

	// Tendermint integration here
	var payload types.Bytes
	if payload, err = data.MarshalBinary(); err != nil {
		return NewValidatorError(err)
	}

	ret := api.sendTx(req, payload)
	ret.Type = "tokenAccount"
	return ret
}

// getTokenTx returns Token Tx info
func (api *API) getTokenTx(_ context.Context, params json.RawMessage) interface{} {

	var err error
	req := &acmeapi.APIRequestGet{}

	if err = json.Unmarshal(params, &req); err != nil {
		return NewValidatorError(err)
	}

	// validate only TokenTx.Hash (Assuming the hash is the txid)
	if err = api.validate.StructPartial(req, "Hash"); err != nil {
		return NewValidatorError(err)
	}

	// Tendermint's integration here
	resp, err := api.query.GetTokenTx(req.Hash[:])
	if err != nil {
		return NewValidatorError(err)
	}

	return resp
}

// createTokenTx creates Token Tx
func (api *API) createTokenTx(_ context.Context, params json.RawMessage) interface{} {

	var err error
	req := &acmeapi.APIRequestRaw{}
	data := &acmeapi.TokenTx{}

	// unmarshal req
	if err = json.Unmarshal(params, &req); err != nil {
		return NewValidatorError(err)
	}

	// validate request
	if err = api.validate.Struct(req); err != nil {
		return NewValidatorError(err)
	}

	// parse req.tx.data
	if err = json.Unmarshal(*req.Tx.Data, &data); err != nil {
		return NewValidatorError(err)
	}

	// validate request data
	if err = api.validate.Struct(data); err != nil {
		return NewValidatorError(err)
	}

	// Tendermint integration here
	var payload types.Bytes
	if payload, err = data.MarshalBinary(); err != nil {
		return NewValidatorError(err)
	}

	ret := api.sendTx(req, payload)
	ret.Type = "tokenTx"
	return ret
}

// createTokenTx creates Token Tx
func (api *API) faucet(_ context.Context, params json.RawMessage) interface{} {

	var err error
	req := &acmeapi.APIRequestGet{}

	if err = json.Unmarshal(params, &req); err != nil {
		return NewValidatorError(err)
	}

	// validate URL
	if err = api.validate.Struct(req); err != nil {
		return NewValidatorError(err)
	}

	adi, _, _ := types.ParseIdentityChainPath(req.URL.AsString())

	if err = anon.IsAcmeAddress(adi); err != nil {
		return jsonrpc2.NewError(-32802, fmt.Sprintf("Invalid Anonymous ACME address %s: ", adi), err)
	}
	var destAccount types.String
	destAccount = types.String(adi)

	wallet := transactions.NewWalletEntry()
	fromAccount := types.String(wallet.Addr)

	//use the public key of the bvc to make a sponsor address (this doesn't really matter right now, but need something so Identity of the BVC is good)
	txid := sha256.Sum256([]byte("faucet"))

	tokenUrl := types.String("dc/ACME")

	//create a fake synthetic deposit for faucet.
	deposit := synthetic.NewTokenTransactionDeposit(txid[:], &fromAccount, &destAccount)
	amtToDeposit := int64(10)                                //deposit 50k tokens
	deposit.DepositAmount.SetInt64(amtToDeposit * 100000000) // assume 8 decimal places
	deposit.TokenUrl = tokenUrl

	depData, err := deposit.MarshalBinary()
	gtx := new(transactions.GenTransaction)
	gtx.SigInfo = new(transactions.SignatureInfo)
	gtx.SigInfo.URL = wallet.Addr
	gtx.SigInfo.Nonce = wallet.Nonce
	gtx.Transaction = depData
	if err := gtx.SetRoutingChainID(); err != nil {
		return jsonrpc2.NewError(-32802, fmt.Sprintf("bad url generated %s: ", adi), err)
	}
	dataToSign := gtx.TransactionHash()

	ed := new(transactions.ED25519Sig)
	err = ed.Sign(wallet.Nonce, wallet.PrivateKey, dataToSign)
	if err != nil {
		return NewSubmissionError(err)
	}

	gtx.Signature = append(gtx.Signature, ed)

	resp, err := api.txBouncer.SendTx(gtx)

	if err != nil {
		return NewAccumulateError(err)
	}

	ret := acmeapi.APIDataResponse{}
	ret.Type = "faucet"
	var msg json.RawMessage
	msg = []byte(fmt.Sprintf("{\"txid\":\"%x\",\"log\":\"%s\"}", gtx.TransactionHash(), resp.Log))
	ret.Data = &msg
	return &ret
}
