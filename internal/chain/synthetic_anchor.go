package chain

import (
	"bytes"
	"fmt"

	"gitlab.com/accumulatenetwork/accumulate/config"
	"gitlab.com/accumulatenetwork/accumulate/protocol"
	"gitlab.com/accumulatenetwork/accumulate/smt/managed"
)

type SyntheticAnchor struct {
	Network *config.Network
}

func (SyntheticAnchor) Type() protocol.TransactionType {
	return protocol.TransactionTypeSyntheticAnchor
}

func (x SyntheticAnchor) Validate(st *StateManager, tx *protocol.Envelope) (protocol.TransactionResult, error) {
	// Unpack the payload
	body, ok := tx.Transaction.Body.(*protocol.SyntheticAnchor)
	if !ok {
		return nil, fmt.Errorf("invalid payload: want %T, got %T", new(protocol.SyntheticAnchor), tx.Transaction.Body)
	}

	// Verify the origin
	if _, ok := st.Origin.(*protocol.Anchor); !ok {
		return nil, fmt.Errorf("invalid origin record: want %v, got %v", protocol.AccountTypeAnchor, st.Origin.GetType())
	}

	// Check the source URL
	name, ok := protocol.ParseBvnUrl(body.Source)
	var fromDirectory bool
	switch {
	case ok:
		name = "bvn-" + name
	case protocol.IsDnUrl(body.Source):
		name, fromDirectory = "dn", true
	default:
		return nil, fmt.Errorf("invalid source: not a BVN or the DN")
	}

	if body.Receipt.Start != nil {
		// If we got a receipt, verify it
		err := x.verifyReceipt(st, body)
		if err != nil {
			return nil, err
		}

		if fromDirectory {
			st.AddDirectoryAnchor(body)

			ledgerState := protocol.NewInternalLedger()
			err = st.LoadUrlAs(st.nodeUrl.JoinPath(protocol.Ledger), ledgerState)
			if err != nil {
				return nil, err
			}
			if body.AcmeOraclePrice == 0 {
				return nil, fmt.Errorf("attempting to set oracle price to 0")
			}
			ledgerState.PendingOracle = body.AcmeOraclePrice

			st.Update(ledgerState)
		}
	}

	// Add the anchor to the chain
	err := st.AddChainEntry(st.OriginUrl, name, protocol.ChainTypeAnchor, body.RootAnchor[:], body.RootIndex, body.Block)
	if err != nil {
		return nil, err
	}

	return nil, nil
}

func (SyntheticAnchor) verifyReceipt(st *StateManager, body *protocol.SyntheticAnchor) error {
	// Get the merkle state at the specified index
	chainName := protocol.MinorRootChain
	if body.Major {
		chainName = protocol.MajorRootChain
	}
	rootChain, err := st.ReadChain(st.nodeUrl.JoinPath(protocol.Ledger), chainName)
	if err != nil {
		return fmt.Errorf("failed to open ledger %s chain: %v", chainName, err)
	}
	ms, err := rootChain.State(int64(body.SourceIndex))
	if err != nil {
		return fmt.Errorf("failed to get state %d of ledger %s chain: %v", body.SourceIndex, chainName, err)
	}

	// Verify the start matches the root chain anchor
	if !bytes.Equal(ms.GetMDRoot(), body.Receipt.Start) {
		return fmt.Errorf("receipt start does match root anchor at %d", body.RootIndex)
	}

	// Calculate receipt end
	hash := managed.Hash(body.Receipt.Start)
	for _, entry := range body.Receipt.Entries {
		if entry.Right {
			hash = hash.Combine(managed.Sha256, entry.Hash)
		} else {
			hash = managed.Hash(entry.Hash).Combine(managed.Sha256, hash)
		}
	}

	// Verify the end matches what we received
	if !bytes.Equal(hash, body.RootAnchor[:]) {
		return fmt.Errorf("receipt end does match received root")
	}

	return nil
}
