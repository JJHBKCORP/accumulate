package chain

import (
	"fmt"

	"gitlab.com/accumulatenetwork/accumulate/protocol"
	"gitlab.com/accumulatenetwork/accumulate/types"
	"gitlab.com/accumulatenetwork/accumulate/types/api/transactions"
)

type CreateKeyPage struct{}

func (CreateKeyPage) Type() types.TxType { return types.TxTypeCreateKeyPage }

func (CreateKeyPage) Validate(st *StateManager, tx *transactions.Envelope) (protocol.TransactionResult, error) {
	var group *protocol.KeyBook
	switch origin := st.Origin.(type) {
	case *protocol.ADI:
		// Create an unbound sig spec
	case *protocol.KeyBook:
		group = origin
	default:
		return nil, fmt.Errorf("invalid origin record: want account type %v or %v, got %v", protocol.AccountTypeIdentity, protocol.AccountTypeKeyBook, origin.GetType())
	}

	body, ok := tx.Transaction.Body.(*protocol.CreateKeyPage)
	if !ok {
		return nil, fmt.Errorf("invalid payload: want %T, got %T", new(protocol.CreateKeyPage), tx.Transaction.Body)
	}

	if len(body.Keys) == 0 {
		return nil, fmt.Errorf("cannot create empty sig spec")
	}

	if !body.Url.Identity().Equal(st.OriginUrl.Identity()) {
		return nil, fmt.Errorf("%q does not belong to %q", body.Url, st.OriginUrl)
	}

	scc := new(protocol.SyntheticCreateChain)
	scc.Cause = types.Bytes(tx.GetTxHash()).AsBytes32()
	st.Submit(st.OriginUrl, scc)

	page := protocol.NewKeyPage()
	page.Url = body.Url
	page.Threshold = 1 // Require one signature from the Key Page
	page.ManagerKeyBook = body.Manager

	if group != nil {
		groupUrl, err := group.ParseUrl()
		if err != nil {
			// Failing here would require writing an invalid URL to the state.
			// But stuff happens, so don't panic.
			return nil, fmt.Errorf("invalid origin record URL: %v", err)
		}

		group.Pages = append(group.Pages, body.Url)
		page.KeyBook = groupUrl

		err = scc.Update(group)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal state: %v", err)
		}
	}

	for _, sig := range body.Keys {
		ss := new(protocol.KeySpec)
		ss.PublicKey = sig.PublicKey
		page.Keys = append(page.Keys, ss)
	}

	err := scc.Create(page)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal state: %v", err)
	}

	return nil, nil
}
