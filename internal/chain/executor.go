package chain

import (
	"bytes"
	"crypto/ed25519"
	"crypto/sha256"
	"errors"
	"fmt"
	"github.com/AccumulateNetwork/accumulate/networks/connections"
	"strings"
	"sync"
	"time"

	"github.com/AccumulateNetwork/accumulate/config"
	"github.com/AccumulateNetwork/accumulate/internal/abci"
	"github.com/AccumulateNetwork/accumulate/internal/database"
	"github.com/AccumulateNetwork/accumulate/internal/indexing"
	"github.com/AccumulateNetwork/accumulate/internal/logging"
	"github.com/AccumulateNetwork/accumulate/protocol"
	"github.com/AccumulateNetwork/accumulate/smt/pmt"
	"github.com/AccumulateNetwork/accumulate/smt/storage"
	"github.com/AccumulateNetwork/accumulate/smt/storage/memory"
	"github.com/AccumulateNetwork/accumulate/types"
	"github.com/AccumulateNetwork/accumulate/types/api/transactions"
	"github.com/AccumulateNetwork/accumulate/types/state"
	"github.com/tendermint/tendermint/libs/log"
)

const chainWGSize = 4

type Executor struct {
	ExecutorOptions

	executors map[types.TxType]TxExecutor
	governor  *governor
	logger    log.Logger

	wg      *sync.WaitGroup
	mu      *sync.Mutex
	chainWG map[uint64]*sync.WaitGroup

	blockLeader bool
	blockIndex  int64
	blockTime   time.Time
	blockBatch  *database.Batch
	blockMeta   blockMetadata
}

var _ abci.Chain = (*Executor)(nil)

type ExecutorOptions struct {
	DB               *database.Database
	Logger           log.Logger
	Key              ed25519.PrivateKey
	ConnectionMgr    connections.ConnectionManager
	ConnectionRouter connections.ConnectionRouter
	Local            connections.ABCIBroadcastClient
	Network          config.Network

	isGenesis bool
	// TODO Remove once tests support running the DN
	IsTest bool
}

func newExecutor(opts ExecutorOptions, executors ...TxExecutor) (*Executor, error) {
	m := new(Executor)
	m.ExecutorOptions = opts
	m.executors = map[types.TxType]TxExecutor{}
	m.wg = new(sync.WaitGroup)
	m.mu = new(sync.Mutex)

	if opts.Logger != nil {
		m.logger = opts.Logger.With("module", "executor")
	}

	if !m.isGenesis {
		m.governor = newGovernor(opts)
	}

	for _, x := range executors {
		if _, ok := m.executors[x.Type()]; ok {
			panic(fmt.Errorf("duplicate executor for %d", x.Type()))
		}
		m.executors[x.Type()] = x
	}

	batch := m.DB.Begin()
	defer batch.Discard()

	var height int64
	ledger := protocol.NewInternalLedger()
	err := batch.Account(m.Network.NodeUrl(protocol.Ledger)).GetStateAs(ledger)
	switch {
	case err == nil:
		height = ledger.Index
	case errors.Is(err, storage.ErrNotFound):
		height = 0
	default:
		return nil, err
	}

	m.logInfo("Loaded", "height", height, "hash", logging.AsHex(batch.RootHash()))
	return m, nil
}

func (m *Executor) logDebug(msg string, keyVals ...interface{}) {
	if m.logger != nil {
		m.logger.Debug(msg, keyVals...)
	}
}

func (m *Executor) logInfo(msg string, keyVals ...interface{}) {
	if m.logger != nil {
		m.logger.Info(msg, keyVals...)
	}
}

func (m *Executor) logError(msg string, keyVals ...interface{}) {
	if m.logger != nil {
		m.logger.Error(msg, keyVals...)
	}
}

func (m *Executor) Start() error {
	return m.governor.Start()
}

func (m *Executor) Stop() error {
	return m.governor.Stop()
}

func (m *Executor) Genesis(time time.Time, callback func(st *StateManager) error) ([]byte, error) {
	var err error

	if !m.isGenesis {
		panic("Cannot call Genesis on a node txn executor")
	}

	m.blockIndex = 1
	m.blockTime = time
	m.blockBatch = m.DB.Begin()

	env := new(transactions.Envelope)
	env.Transaction = new(transactions.Transaction)
	env.Transaction.Origin = protocol.AcmeUrl()
	env.Transaction.Body, err = new(protocol.InternalGenesis).MarshalBinary()
	if err != nil {
		return nil, err
	}

	st, err := NewStateManager(m.blockBatch, m.Network.NodeUrl(), env)
	if err == nil {
		return nil, errors.New("already initialized")
	} else if !errors.Is(err, storage.ErrNotFound) {
		return nil, err
	}
	st.logger.L = m.logger

	txPending := state.NewPendingTransaction(env)
	txAccepted, txPending := state.NewTransaction(txPending)

	status := &protocol.TransactionStatus{Delivered: true}
	err = m.blockBatch.Transaction(env.Transaction.Hash()).Put(txAccepted, status, nil)
	if err != nil {
		return nil, err
	}

	err = indexing.BlockState(m.blockBatch, m.Network.NodeUrl(protocol.Ledger)).Clear()
	if err != nil {
		return nil, err
	}

	err = callback(st)
	if err != nil {
		return nil, err
	}

	submitted, err := st.Commit()
	if err != nil {
		return nil, err
	}

	// Process synthetic transactions generated by the validator
	st.Reset()
	err = m.addSynthTxns(&st.stateCache, submitted)
	if err != nil {
		return nil, err
	}
	_, err = st.Commit()
	if err != nil {
		return nil, err
	}

	return m.Commit()
}

func (m *Executor) InitChain(data []byte, time time.Time, blockIndex int64) error {
	if m.isGenesis {
		panic("Cannot call InitChain on a genesis txn executor")
	}

	// Load the genesis state (JSON) into an in-memory key-value store
	src := new(memory.DB)
	_ = src.InitDB("", nil)
	err := src.UnmarshalJSON(data)
	if err != nil {
		return fmt.Errorf("failed to unmarshal app state: %v", err)
	}

	// Load the BPT root hash so we can verify the system state
	var hash [32]byte
	data, err = src.Begin().Get(storage.MakeKey("BPT", "Root"))
	switch {
	case err == nil:
		bpt := pmt.NewBPT()
		bpt.UnMarshal(data)
		hash = bpt.Root.Hash
	case errors.Is(err, storage.ErrNotFound):
		// OK
	default:
		return fmt.Errorf("failed to load BPT root hash from app state: %v", err)
	}

	// Dump the genesis state into the key-value store
	batch := m.DB.Begin()
	defer batch.Discard()
	batch.Import(src)

	// Commit the database batch
	err = batch.Commit()
	if err != nil {
		return fmt.Errorf("failed to load app state into database: %v", err)
	}

	// Recreate the batch to reload the BPT
	batch = m.DB.Begin()
	defer batch.Discard()

	// Make sure the database BPT root hash matches what we found in the genesis state
	if !bytes.Equal(hash[:], batch.RootHash()) {
		panic(fmt.Errorf("BPT root hash from state DB does not match the app state\nWant: %X\nGot:  %X", hash[:], batch.RootHash()))
	}

	return m.governor.DidCommit(batch, true, true, blockIndex, time)
}

// BeginBlock implements ./abci.Chain
func (m *Executor) BeginBlock(req abci.BeginBlockRequest) (abci.BeginBlockResponse, error) {
	m.logDebug("Begin block", "height", req.Height, "leader", req.IsLeader, "time", req.Time)

	m.chainWG = make(map[uint64]*sync.WaitGroup, chainWGSize)
	m.blockLeader = req.IsLeader
	m.blockIndex = req.Height
	m.blockTime = req.Time
	m.blockBatch = m.DB.Begin()
	m.blockMeta = blockMetadata{}

	m.governor.DidBeginBlock(req.IsLeader, req.Height, req.Time)

	// Reset the block state
	err := indexing.BlockState(m.blockBatch, m.Network.NodeUrl(protocol.Ledger)).Clear()
	if err != nil {
		return abci.BeginBlockResponse{}, nil
	}

	// Load the ledger state
	ledger := m.blockBatch.Account(m.Network.NodeUrl(protocol.Ledger))
	ledgerState := protocol.NewInternalLedger()
	err = ledger.GetStateAs(ledgerState)
	switch {
	case err == nil:
		// Make sure the block index is increasing
		if ledgerState.Index >= m.blockIndex {
			panic(fmt.Errorf("Current height is %d but the next block height is %d!", ledgerState.Index, m.blockIndex))
		}

	case m.isGenesis && errors.Is(err, storage.ErrNotFound):
		// OK

	default:
		return abci.BeginBlockResponse{}, err
	}

	// Reset transient values
	ledgerState.Index = m.blockIndex
	ledgerState.Timestamp = m.blockTime
	ledgerState.Updates = nil
	ledgerState.Synthetic.Produced = nil

	err = ledger.PutState(ledgerState)
	if err != nil {
		return abci.BeginBlockResponse{}, err
	}

	return abci.BeginBlockResponse{}, nil
}

// EndBlock implements ./abci.Chain
func (m *Executor) EndBlock(req abci.EndBlockRequest) {}

// Commit implements ./abci.Chain
func (m *Executor) Commit() ([]byte, error) {
	m.wg.Wait()

	// Discard changes if commit fails
	defer m.blockBatch.Discard()

	// Load the ledger
	ledger := m.blockBatch.Account(m.Network.NodeUrl(protocol.Ledger))
	ledgerState := protocol.NewInternalLedger()
	err := ledger.GetStateAs(ledgerState)
	if err != nil {
		return nil, err
	}

	// Deduplicate the update list
	updatedMap := make(map[string]bool, len(ledgerState.Updates))
	updatedSlice := make([]protocol.AnchorMetadata, 0, len(ledgerState.Updates))
	for _, u := range ledgerState.Updates {
		s := strings.ToLower(fmt.Sprintf("%s#chain/%s", u.Account, u.Name))
		if updatedMap[s] {
			continue
		}

		updatedSlice = append(updatedSlice, u)
		updatedMap[s] = true
	}
	ledgerState.Updates = updatedSlice

	if m.blockMeta.Empty() && len(updatedSlice) == 0 && len(ledgerState.Synthetic.Produced) == 0 {
		m.logInfo("Committed empty transaction")
	} else {
		m.logInfo("Committing", "height", m.blockIndex, "delivered", m.blockMeta.Delivered, "signed", m.blockMeta.SynthSigned, "sent", m.blockMeta.SynthSent, "updated", len(updatedSlice), "submitted", len(ledgerState.Synthetic.Produced))
		t := time.Now()

		err := m.doCommit(ledgerState)
		if err != nil {
			return nil, err
		}

		// Write the updated ledger
		err = ledger.PutState(ledgerState)
		if err != nil {
			return nil, err
		}

		err = m.blockBatch.Commit()
		if err != nil {
			return nil, err
		}

		m.logInfo("Committed", "height", m.blockIndex, "duration", time.Since(t))
	}

	if !m.isGenesis {
		err := m.governor.DidCommit(m.blockBatch, m.blockLeader, false, m.blockIndex, m.blockTime)
		if err != nil {
			return nil, err
		}
	}

	// Get BPT root from a clean batch
	batch := m.DB.Begin()
	defer batch.Discard()
	return batch.RootHash(), nil
}

func (m *Executor) doCommit(ledgerState *protocol.InternalLedger) error {
	// Load the main chain of the minor root
	ledgerUrl := m.Network.NodeUrl(protocol.Ledger)
	ledger := m.blockBatch.Account(ledgerUrl)
	rootChain, err := ledger.Chain(protocol.MinorRootChain, protocol.ChainTypeAnchor)
	if err != nil {
		return err
	}

	// Pending transaction-chain index entries
	type txChainIndexEntry struct {
		indexing.TransactionChainEntry
		Txid []byte
	}
	txChainEntries := make([]*txChainIndexEntry, 0, len(ledgerState.Updates))

	// Add an anchor to the root chain for every updated chain
	accountSeen := map[string]bool{}
	updates := ledgerState.Updates
	ledgerState.Updates = make([]protocol.AnchorMetadata, 0, len(updates))
	for _, u := range updates {
		// Do not create root chain or BPT entries for the ledger
		if ledgerUrl.Equal(u.Account) {
			continue
		}

		ledgerState.Updates = append(ledgerState.Updates, u)
		m.logDebug("Updated a chain", "url", fmt.Sprintf("%s#chain/%s", u.Account, u.Name))

		// Load the chain
		record := m.blockBatch.Account(u.Account)
		recordChain, err := record.ReadChain(u.Name)
		if err != nil {
			return err
		}

		// Add its anchor to the root chain
		rootIndex := rootChain.Height()
		err = rootChain.AddEntry(recordChain.Anchor())
		if err != nil {
			return err
		}

		// Add a pending transaction-chain index update
		if u.Type == protocol.ChainTypeTransaction {
			e := new(txChainIndexEntry)
			e.Txid = u.Entry
			e.Account = u.Account
			e.Chain = u.Name
			e.Block = uint64(m.blockIndex)
			e.ChainEntry = u.Index
			e.ChainAnchor = uint64(recordChain.Height()) - 1
			e.RootEntry = uint64(rootIndex)
			txChainEntries = append(txChainEntries, e)
		}

		// Once for each account
		s := strings.ToLower(u.Account.String())
		if accountSeen[s] {
			continue
		}
		accountSeen[s] = true

		// Load the state
		state, err := record.GetState()
		if err != nil {
			return err
		}

		// Marshal it
		data, err := state.MarshalBinary()
		if err != nil {
			return err
		}

		// Hash it
		var hashes []byte
		h := sha256.Sum256(data)
		hashes = append(hashes, h[:]...)

		// Load the object metadata
		objMeta, err := record.GetObject()
		if err != nil {
			return err
		}

		// For each chain
		for _, chainMeta := range objMeta.Chains {
			// Load the chain
			recordChain, err := record.ReadChain(chainMeta.Name)
			if err != nil {
				return err
			}

			// Get the anchor
			anchor := recordChain.Anchor()
			h := sha256.Sum256(anchor)
			hashes = append(hashes, h[:]...)
		}

		// Write the hash of the hashes to the BPT
		record.PutBpt(sha256.Sum256(hashes))
	}

	// Add the synthetic transaction chain to the root chain
	var synthRootIndex, synthAnchorIndex uint64
	if len(ledgerState.Synthetic.Produced) > 0 {
		synthChain, err := ledger.ReadChain(protocol.SyntheticChain)
		if err != nil {
			return err
		}

		ledgerState.Updates = append(ledgerState.Updates, protocol.AnchorMetadata{
			ChainMetadata: protocol.ChainMetadata{
				Name: protocol.SyntheticChain,
				Type: protocol.ChainTypeTransaction,
			},
			Account: ledgerUrl,
			Index:   uint64(synthChain.Height() - 1),
		})

		synthAnchorIndex = uint64(synthChain.Height() - 1)
		synthRootIndex = uint64(rootChain.Height())
		err = rootChain.AddEntry(synthChain.Anchor())
		if err != nil {
			return err
		}
	}

	// Add the BPT to the root chain
	m.blockBatch.UpdateBpt()
	ledgerState.Updates = append(ledgerState.Updates, protocol.AnchorMetadata{
		ChainMetadata: protocol.ChainMetadata{
			Name: "bpt",
		},
		Account: m.Network.NodeUrl(),
		Index:   uint64(m.blockIndex - 1),
	})

	err = rootChain.AddEntry(m.blockBatch.RootHash())
	if err != nil {
		return err
	}

	// Update the transaction-chain index
	for _, e := range txChainEntries {
		e.RootAnchor = uint64(rootChain.Height()) - 1
		err = indexing.TransactionChain(m.blockBatch, e.Txid).Add(&e.TransactionChainEntry)
		if err != nil {
			return err
		}
	}

	// Add transaction-chain index entries for synthetic transactions
	blockState, err := indexing.BlockState(m.blockBatch, ledgerUrl).Get()
	if err != nil {
		return err
	}

	for _, e := range blockState.ProducedSynthTxns {
		err = indexing.TransactionChain(m.blockBatch, e.Transaction).Add(&indexing.TransactionChainEntry{
			Account:     ledgerUrl,
			Chain:       protocol.SyntheticChain,
			Block:       uint64(m.blockIndex),
			ChainEntry:  e.ChainEntry,
			ChainAnchor: synthAnchorIndex,
			RootEntry:   synthRootIndex,
			RootAnchor:  uint64(rootChain.Height()) - 1,
		})
		if err != nil {
			return err
		}
	}

	return nil
}
