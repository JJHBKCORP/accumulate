package chain

import (
	"errors"
	"fmt"

	"gitlab.com/accumulatenetwork/accumulate/protocol"
	"gitlab.com/accumulatenetwork/accumulate/smt/storage"
	"gitlab.com/accumulatenetwork/accumulate/types"
	"gitlab.com/accumulatenetwork/accumulate/types/api/transactions"
)

type AddCredits struct{}

func (AddCredits) Type() types.TxType { return types.TxTypeAddCredits }

func (AddCredits) Validate(st *StateManager, tx *transactions.Envelope) (protocol.TransactionResult, error) {
	body, ok := tx.Transaction.Body.(*protocol.AddCredits)
	if !ok {
		return nil, fmt.Errorf("invalid payload: want %T, got %T", new(protocol.AddCredits), tx.Transaction.Body)
	}

	ledgerState := protocol.NewInternalLedger()
	err := st.LoadUrlAs(st.nodeUrl.JoinPath(protocol.Ledger), ledgerState)
	if err != nil {
		return nil, err
	}

	if ledgerState.ActiveOracle == 0 {
		return nil, fmt.Errorf("cannot purchase credits: acme oracle price has not been set")
	}

	// tokens = credits / (credits to dollars) / (dollars per token)
	amount := types.NewAmount(protocol.AcmePrecision) // Do everything with ACME precision

	// credits wanted
	amount.Mul(int64(body.Amount)) // Amount in credits
	amount.Div(protocol.CreditsPerFiatUnit)

	//dollars / token
	amount.Mul(protocol.AcmeOraclePrecision)
	amount.Div(int64(ledgerState.ActiveOracle)) // Amount in acme

	// TODO: convert credit purchase from buying exact number of credits to
	// specifying amount of acme to spend. body.Amount needs to be converted to bigint
	// If specifying amount of acme to spend, do this instead of amount calculation above:
	//credits := types.NewAmount(protocol.CreditsPerFiatUnit) // want to obtain credits
	//credits.Mul(int64(ledgerState.ActiveOracle))            // fiat units / acme
	//credits.Mul(body.Amount)                                // acme the user wants to spend
	//credits.Div(protocol.AcmeOraclePrecision)               // adjust the precision of oracle to real units
	//credits.Div(protocol.AcmePrecision)                     // adjust the precision of acme to spend to real units

	recv, err := st.LoadUrl(body.Recipient)
	if err == nil {
		// If the recipient happens to be on the same BVC, ensure it is a valid
		// recipient. Most credit transfers will be within the same ADI, so this
		// should catch most mistakes early.
		switch recv := recv.(type) {
		case *protocol.LiteTokenAccount, *protocol.KeyPage:
			// OK
		default:
			return nil, fmt.Errorf("invalid recipient: want account type %v or %v, got %v", protocol.AccountTypeLiteTokenAccount, protocol.AccountTypeKeyPage, recv.GetType())
		}
	} else if errors.Is(err, storage.ErrNotFound) {
		if body.Recipient.Routing() == tx.Transaction.Origin.Routing() {
			// If the recipient and the origin have the same routing number,
			// they must be on the same BVC. Thus in that case, failing to
			// locate the recipient chain means it doesn't exist.
			return nil, fmt.Errorf("invalid recipient: not found")
		}
	} else {
		return nil, fmt.Errorf("failed to load recipient: %v", err)
	}

	var account tokenChain
	switch origin := st.Origin.(type) {
	case *protocol.LiteTokenAccount:
		account = origin
	case *protocol.TokenAccount:
		account = origin
	default:
		return nil, fmt.Errorf("not an account: %q", tx.Transaction.Origin)
	}

	tokenUrl, err := account.ParseTokenUrl()
	if err != nil {
		return nil, fmt.Errorf("invalid token account: %v", err)
	}

	// Only ACME tokens can be converted into credits
	if !protocol.AcmeUrl().Equal(tokenUrl) {
		return nil, fmt.Errorf("%q tokens cannot be converted into credits", tokenUrl.String())
	}

	if !account.CanDebitTokens(&amount.Int) {
		return nil, fmt.Errorf("insufficient balance: have %v, want %v", account.TokenBalance(), &amount.Int)
	}

	if !account.DebitTokens(&amount.Int) {
		return nil, fmt.Errorf("failed to debit %v", tx.Transaction.Origin)
	}
	st.Update(account)

	// Create the synthetic transaction
	sdc := new(protocol.SyntheticDepositCredits)
	copy(sdc.Cause[:], tx.GetTxHash())
	sdc.Amount = body.Amount
	st.Submit(body.Recipient, sdc)

	//Create synthetic burn token
	burnAcme := new(protocol.SyntheticBurnTokens)
	copy(sdc.Cause[:], tx.GetTxHash())
	burnAcme.Amount = amount.Int
	st.Submit(tokenUrl, burnAcme)

	return nil, nil
}
