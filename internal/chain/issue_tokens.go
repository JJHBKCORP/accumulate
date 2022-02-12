package chain

import (
	"fmt"

	"gitlab.com/accumulatenetwork/accumulate/internal/url"
	"gitlab.com/accumulatenetwork/accumulate/protocol"
	"gitlab.com/accumulatenetwork/accumulate/types"
	"gitlab.com/accumulatenetwork/accumulate/types/api/transactions"
)

type IssueTokens struct{}

func (IssueTokens) Type() types.TxType { return types.TxTypeIssueTokens }

func (IssueTokens) Validate(st *StateManager, tx *transactions.Envelope) (protocol.TransactionResult, error) {
	body := new(protocol.IssueTokens)
	err := tx.As(body)
	if err != nil {
		return nil, fmt.Errorf("invalid payload: %v", err)
	}

	accountUrl, err := url.Parse(body.Recipient)
	if err != nil {
		return nil, fmt.Errorf("invalid recipient account URL: %v", err)
	}

	issuer, ok := st.Origin.(*protocol.TokenIssuer)
	if !ok {
		return nil, fmt.Errorf("invalid origin record: want chain type %v, got %v", protocol.AccountTypeTokenIssuer, st.Origin.Header().Type)
	}

	if issuer.Supply.Cmp(&body.Amount) < 0 && issuer.HasSupplyLimit {
		return nil, fmt.Errorf("can't issue more than the limited supply")
	}
	issuer.Supply.Sub(&issuer.Supply, &body.Amount)

	deposit := new(protocol.SyntheticDepositTokens)
	copy(deposit.Cause[:], tx.GetTxHash())
	deposit.Token = issuer.Header().GetChainUrl()
	deposit.Amount = body.Amount
	st.Submit(accountUrl, deposit)

	st.Update(issuer)

	return nil, nil
}
