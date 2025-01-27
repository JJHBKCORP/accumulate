package api

import (
	"fmt"

	"gitlab.com/accumulatenetwork/accumulate/protocol"
	"gitlab.com/accumulatenetwork/accumulate/types/state"
)

func packStateResponse(obj *state.Object, chain state.Chain) (*ChainQueryResponse, error) {
	res := new(ChainQueryResponse)
	res.Type = chain.GetType().String()
	res.MainChain = new(MerkleState)
	res.MainChain.Height = obj.Height
	res.MainChain.Roots = obj.Roots
	res.Data = chain

	u, err := chain.Header().ParseUrl()
	if err == nil {
		res.ChainId = u.AccountID()
	}
	return res, nil
}

func packTxResponse(txid [32]byte, synth []byte, ms *MerkleState, main *state.Transaction, pend *state.PendingTransaction, payload protocol.TransactionPayload) (*TransactionQueryResponse, error) {
	var tx *state.TxState
	if main != nil {
		tx = &main.TxState
	} else {
		tx = pend.TransactionState
	}

	res := new(TransactionQueryResponse)
	res.Type = payload.GetType().String()
	res.Data = payload
	res.TransactionHash = txid[:]
	res.MainChain = ms
	res.KeyPage = new(KeyPage)
	res.KeyPage.Height = tx.SigInfo.KeyPageHeight
	res.KeyPage.Index = tx.SigInfo.KeyPageIndex

	if len(synth)%32 != 0 {
		return nil, fmt.Errorf("invalid synthetic transaction information, not divisible by 32")
	}

	if synth != nil {
		res.SyntheticTxids = make([][32]byte, len(synth)/32)
		for i := range res.SyntheticTxids {
			copy(res.SyntheticTxids[i][:], synth[i*32:(i+1)*32])
		}
	}

	switch payload := payload.(type) {
	case *protocol.SendTokens:
		if synth != nil && len(res.SyntheticTxids) != len(payload.To) {
			return nil, fmt.Errorf("not enough synthetic TXs: want %d, got %d", len(payload.To), len(res.SyntheticTxids))
		}

		res.Origin = tx.SigInfo.Origin
		data := new(TokenSend)
		data.From = main.SigInfo.Origin
		data.To = make([]TokenDeposit, len(payload.To))
		for i, to := range payload.To {
			data.To[i].Url = to.Url
			data.To[i].Amount = to.Amount
			if synth != nil {
				data.To[i].Txid = synth[i*32 : (i+1)*32]
			}
		}

		res.Origin = main.SigInfo.Origin
		res.Data = data

	case *protocol.SyntheticDepositTokens:
		res.Origin = main.SigInfo.Origin
		res.Data = payload

	default:
		res.Origin = tx.SigInfo.Origin
		res.Data = payload
	}

	if pend == nil {
		return res, nil
	}

	res.Signatures = pend.Signature

	if len(pend.Status) > 0 {
		status := new(protocol.TransactionStatus)
		err := status.UnmarshalBinary(pend.Status)
		if err != nil {
			return nil, fmt.Errorf("invalid transaction status: %v", err)
		}

		res.Status = status
	}

	return res, nil
}
