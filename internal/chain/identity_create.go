package chain

import (
	"fmt"
	"strings"

	"github.com/AccumulateNetwork/accumulated/internal/url"
	"github.com/AccumulateNetwork/accumulated/protocol"
	"github.com/AccumulateNetwork/accumulated/types"
	"github.com/AccumulateNetwork/accumulated/types/api"
	"github.com/AccumulateNetwork/accumulated/types/api/transactions"
	"github.com/AccumulateNetwork/accumulated/types/state"
)

type IdentityCreate struct{}

func (IdentityCreate) Type() types.TxType { return types.TxTypeIdentityCreate }

func (IdentityCreate) CheckTx(st *state.StateEntry, tx *transactions.GenTransaction) error {
	if st == nil {
		return fmt.Errorf("current state not defined")
	}
	return nil
}

func (IdentityCreate) DeliverTx(st *state.StateEntry, tx *transactions.GenTransaction) (*DeliverTxResult, error) {
	if st == nil {
		return nil, fmt.Errorf("current state not defined")
	}
	if st.AdiHeader == nil {
		return nil, fmt.Errorf("missing sponsor identity")
	}
	if tx.Signature == nil {
		return nil, fmt.Errorf("no signatures available")
	}

	switch st.AdiHeader.Type {
	case types.ChainTypeAnonTokenAccount:
		account := new(state.TokenAccount)
		err := st.AdiState.As(account)
		if err != nil {
			return nil, fmt.Errorf("error unmarshaling anon state account object, %v", err)
		}

	case types.ChainTypeAdi:
		adiState := state.AdiState{}
		err := adiState.UnmarshalBinary(st.AdiState.Entry)
		if err != nil {
			return nil, fmt.Errorf("unable to unmarshal adi state entry, %v", err)
		}

		//this should be done at a higher level...
		if !adiState.VerifyKey(tx.Signature[0].PublicKey) {
			return nil, fmt.Errorf("key is not supported by current ADI state")
		}

		if !adiState.VerifyAndUpdateNonce(tx.SigInfo.Nonce) {
			return nil, fmt.Errorf("invalid nonce, adi state %d but provided %d", adiState.Nonce, tx.SigInfo.Nonce)
		}

	default:
		return nil, fmt.Errorf("chain type %d cannot sponsor ADIs", st.AdiHeader.Type)
	}

	ic := api.ADI{}
	err := ic.UnmarshalBinary(tx.Transaction)
	if err != nil {
		return nil, fmt.Errorf("transaction is not a valid identity create message")
	}

	u, err := url.Parse(*ic.URL.AsString())
	if err != nil {
		return nil, fmt.Errorf("invalid URL: %v", err)
	}
	if u.Path != "" {
		return nil, fmt.Errorf("creating sub-ADIs is not supported")
	}
	if strings.ContainsRune(u.Hostname(), '.') {
		return nil, fmt.Errorf("ADI URLs cannot contain dots")
	}

	scc := new(protocol.SyntheticCreateChain)
	scc.Cause = types.Bytes(tx.TransactionHash()).AsBytes32()
	err = scc.Add(state.NewADI(ic.URL, state.KeyTypeSha256, ic.PublicKeyHash[:]))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal synthetic TX: %v", err)
	}

	syn := new(transactions.GenTransaction)
	syn.SigInfo = &transactions.SignatureInfo{}
	syn.SigInfo.URL = *ic.URL.AsString()
	syn.Transaction, err = scc.MarshalBinary()
	if err != nil {
		return nil, fmt.Errorf("failed to marshal synthetic TX: %v", err)
	}

	res := new(DeliverTxResult)
	res.AddSyntheticTransaction(syn)
	return res, nil
}