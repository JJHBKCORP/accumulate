package chain

import (
	"errors"
	"fmt"
	"reflect"

	"gitlab.com/accumulatenetwork/accumulate/internal/database"
	"gitlab.com/accumulatenetwork/accumulate/internal/logging"
	"gitlab.com/accumulatenetwork/accumulate/internal/url"
	"gitlab.com/accumulatenetwork/accumulate/protocol"
	"gitlab.com/accumulatenetwork/accumulate/smt/storage"
	"gitlab.com/accumulatenetwork/accumulate/types"
	"gitlab.com/accumulatenetwork/accumulate/types/state"
)

type stateCache struct {
	logger  logging.OptionalLogger
	nodeUrl *url.URL
	txType  protocol.TransactionType
	txHash  types.Bytes32

	batch      *database.Batch
	operations []stateOperation
	chains     map[[32]byte]protocol.Account
	indices    map[[32]byte]*writeIndex
}

func newStateCache(nodeUrl *url.URL, txtype protocol.TransactionType, txid [32]byte, batch *database.Batch) *stateCache {
	c := new(stateCache)
	c.nodeUrl = nodeUrl
	c.txType = txtype
	c.txHash = txid

	c.batch = batch
	c.Reset()
	return c
}

func (c *stateCache) Reset() {
	c.operations = c.operations[:0]
	c.chains = map[[32]byte]protocol.Account{}
	c.indices = map[[32]byte]*writeIndex{}
}

func (c *stateCache) Commit() ([]protocol.Account, error) {
	var create []protocol.Account
	for _, op := range c.operations {
		records, err := op.Execute(c)
		if err != nil {
			return nil, err
		}
		create = append(create, records...)
	}

	return create, nil
}

func (c *stateCache) load(id [32]byte, r *database.Account) (state.Chain, error) {
	st, ok := c.chains[id]
	if ok {
		return st, nil
	}

	st, err := r.GetState()
	if err != nil {
		return nil, err
	}

	if c.chains == nil {
		c.chains = map[[32]byte]protocol.Account{}
	}
	c.chains[id] = st
	return st, nil
}

func (c *stateCache) loadAs(id [32]byte, r *database.Account, v interface{}) (err error) {
	state, err := c.load(id, r)
	if err != nil {
		return err
	}

	rv := reflect.ValueOf(v)
	rr := reflect.ValueOf(state)
	if !rr.Type().AssignableTo(rv.Type()) {
		return fmt.Errorf("want %T, got %T", v, state)
	}

	// Catch reflection panic
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("failed to load chain: unable to write to %T", v)
		}
	}()
	rv.Elem().Set(rr.Elem())
	return nil
}

// LoadUrl loads a chain by URL and unmarshals it.
func (c *stateCache) LoadUrl(u *url.URL) (state.Chain, error) {
	return c.load(u.AccountID32(), c.batch.Account(u))
}

// LoadUrlAs loads a chain by URL and unmarshals it as a specific type.
func (c *stateCache) LoadUrlAs(u *url.URL, v interface{}) error {
	return c.loadAs(u.AccountID32(), c.batch.Account(u), v)
}

// ReadChain loads an account's chain by URL and name.
func (c *stateCache) ReadChain(u *url.URL, name string) (*database.Chain, error) {
	return c.batch.Account(u).ReadChain(name)
}

//GetHeight loads the height of the chain
func (c *stateCache) GetHeight(u *url.URL) (uint64, error) {
	chain, err := c.batch.Account(u).ReadChain(protocol.MainChain)
	if err != nil {
		return 0, err
	}

	return uint64(chain.Height()), nil
}

// LoadTxn loads and unmarshals a saved transaction
func (c *stateCache) LoadTxn(txid [32]byte) (*state.Transaction, *protocol.TransactionStatus, []protocol.Signature, error) {
	return c.batch.Transaction(txid[:]).Get()
}

func (c *stateCache) AddDirectoryEntry(u ...*url.URL) error {
	return AddDirectoryEntry(func(u *url.URL, key ...interface{}) Value {
		return c.RecordIndex(u, key...)
	}, u...)
}

type Value interface {
	Get() ([]byte, error)
	Put([]byte)
}

func AddDirectoryEntry(getIndex func(*url.URL, ...interface{}) Value, u ...*url.URL) error {
	identity := u[0].Identity()
	mdi := getIndex(identity, "Directory", "Metadata")
	md := new(protocol.DirectoryIndexMetadata)
	data, err := mdi.Get()
	if err == nil {
		err = md.UnmarshalBinary(data)
	}
	if err != nil && !errors.Is(err, storage.ErrNotFound) {
		return fmt.Errorf("failed to load metadata: %v", err)
	}

	for _, u := range u {
		getIndex(identity, "Directory", md.Count).Put([]byte(u.String()))
		md.Count++
	}

	data, err = md.MarshalBinary()
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %v", err)
	}

	mdi.Put(data)
	return nil
}
