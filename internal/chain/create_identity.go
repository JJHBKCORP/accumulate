package chain

import (
	"fmt"

	"gitlab.com/accumulatenetwork/accumulate/internal/url"
	"gitlab.com/accumulatenetwork/accumulate/protocol"
	"gitlab.com/accumulatenetwork/accumulate/types"
	"gitlab.com/accumulatenetwork/accumulate/types/api/transactions"
)

type CreateIdentity struct{}

func (CreateIdentity) Type() types.TxType { return types.TxTypeCreateIdentity }

func (CreateIdentity) Validate(st *StateManager, tx *transactions.Envelope) (protocol.TransactionResult, error) {
	// *protocol.IdentityCreate, *url.URL, state.Chain
	body, ok := tx.Transaction.Body.(*protocol.CreateIdentity)
	if !ok {
		return nil, fmt.Errorf("invalid payload: want %T, got %T", new(protocol.CreateIdentity), tx.Transaction.Body)
	}

	err := protocol.IsValidAdiUrl(body.Url)
	if err != nil {
		return nil, fmt.Errorf("invalid URL: %v", err)
	}

	switch st.Origin.(type) {
	case *protocol.LiteTokenAccount, *protocol.ADI:
		// OK
	default:
		return nil, fmt.Errorf("account type %d cannot be the origininator of ADIs", st.Origin.GetType())
	}

	var pageUrl, bookUrl *url.URL
	if body.KeyBookName == "" {
		return nil, fmt.Errorf("missing key book name")
	} else {
		bookUrl = body.Url.JoinPath(body.KeyBookName)
	}
	if body.KeyPageName == "" {
		return nil, fmt.Errorf("missing key page name")
	} else {
		pageUrl = body.Url.JoinPath(body.KeyPageName)
	}

	keySpec := new(protocol.KeySpec)
	keySpec.PublicKey = body.PublicKey

	page := protocol.NewKeyPage()
	page.Url = pageUrl // TODO Allow override
	page.Keys = append(page.Keys, keySpec)
	page.KeyBook = bookUrl
	page.Threshold = 1 // Require one signature from the Key Page

	book := protocol.NewKeyBook()
	book.Url = bookUrl // TODO Allow override
	book.Pages = append(book.Pages, pageUrl)

	identity := protocol.NewADI()

	identity.Url = body.Url
	identity.KeyBook = bookUrl
	identity.ManagerKeyBook = body.Manager

	st.Create(identity, book, page)
	return nil, nil
}
