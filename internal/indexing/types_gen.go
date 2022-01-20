package indexing

// GENERATED BY go run ./tools/cmd/genmarshal. DO NOT EDIT.

import (
	"bytes"
	"fmt"

	"github.com/AccumulateNetwork/accumulate/internal/encoding"
	"github.com/AccumulateNetwork/accumulate/internal/url"
)

type TransactionChainEntry struct {
	Account     *url.URL `json:"account,omitempty" form:"account" query:"account" validate:"required"`
	Chain       string   `json:"chain,omitempty" form:"chain" query:"chain" validate:"required"`
	Block       uint64   `json:"block,omitempty" form:"block" query:"block" validate:"required"`
	ChainEntry  uint64   `json:"chainEntry,omitempty" form:"chainEntry" query:"chainEntry" validate:"required"`
	ChainAnchor uint64   `json:"chainAnchor,omitempty" form:"chainAnchor" query:"chainAnchor" validate:"required"`
	RootEntry   uint64   `json:"rootEntry,omitempty" form:"rootEntry" query:"rootEntry" validate:"required"`
	RootAnchor  uint64   `json:"rootAnchor,omitempty" form:"rootAnchor" query:"rootAnchor" validate:"required"`
}

type TransactionChainIndex struct {
	Entries []*TransactionChainEntry `json:"entries,omitempty" form:"entries" query:"entries" validate:"required"`
}

func (v *TransactionChainEntry) Equal(u *TransactionChainEntry) bool {
	if !(v.Account.Equal(u.Account)) {
		return false
	}

	if !(v.Chain == u.Chain) {
		return false
	}

	if !(v.Block == u.Block) {
		return false
	}

	if !(v.ChainEntry == u.ChainEntry) {
		return false
	}

	if !(v.ChainAnchor == u.ChainAnchor) {
		return false
	}

	if !(v.RootEntry == u.RootEntry) {
		return false
	}

	if !(v.RootAnchor == u.RootAnchor) {
		return false
	}

	return true
}

func (v *TransactionChainIndex) Equal(u *TransactionChainIndex) bool {
	if !(len(v.Entries) == len(u.Entries)) {
		return false
	}

	for i := range v.Entries {
		v, u := v.Entries[i], u.Entries[i]
		if !(v.Equal(u)) {
			return false
		}

	}

	return true
}

func (v *TransactionChainEntry) BinarySize() int {
	var n int

	n += v.Account.BinarySize()

	n += encoding.StringBinarySize(v.Chain)

	n += encoding.UvarintBinarySize(v.Block)

	n += encoding.UvarintBinarySize(v.ChainEntry)

	n += encoding.UvarintBinarySize(v.ChainAnchor)

	n += encoding.UvarintBinarySize(v.RootEntry)

	n += encoding.UvarintBinarySize(v.RootAnchor)

	return n
}

func (v *TransactionChainIndex) BinarySize() int {
	var n int

	n += encoding.UvarintBinarySize(uint64(len(v.Entries)))

	for _, v := range v.Entries {
		n += v.BinarySize()

	}

	return n
}

func (v *TransactionChainEntry) MarshalBinary() ([]byte, error) {
	var buffer bytes.Buffer

	if b, err := v.Account.MarshalBinary(); err != nil {
		return nil, fmt.Errorf("error encoding Account: %w", err)
	} else {
		buffer.Write(b)
	}

	buffer.Write(encoding.StringMarshalBinary(v.Chain))

	buffer.Write(encoding.UvarintMarshalBinary(v.Block))

	buffer.Write(encoding.UvarintMarshalBinary(v.ChainEntry))

	buffer.Write(encoding.UvarintMarshalBinary(v.ChainAnchor))

	buffer.Write(encoding.UvarintMarshalBinary(v.RootEntry))

	buffer.Write(encoding.UvarintMarshalBinary(v.RootAnchor))

	return buffer.Bytes(), nil
}

func (v *TransactionChainIndex) MarshalBinary() ([]byte, error) {
	var buffer bytes.Buffer

	buffer.Write(encoding.UvarintMarshalBinary(uint64(len(v.Entries))))
	for i, v := range v.Entries {
		_ = i
		if b, err := v.MarshalBinary(); err != nil {
			return nil, fmt.Errorf("error encoding Entries[%d]: %w", i, err)
		} else {
			buffer.Write(b)
		}

	}

	return buffer.Bytes(), nil
}

func (v *TransactionChainEntry) UnmarshalBinary(data []byte) error {
	v.Account = new(url.URL)
	if err := v.Account.UnmarshalBinary(data); err != nil {
		return fmt.Errorf("error decoding Account: %w", err)
	}
	data = data[v.Account.BinarySize():]

	if x, err := encoding.StringUnmarshalBinary(data); err != nil {
		return fmt.Errorf("error decoding Chain: %w", err)
	} else {
		v.Chain = x
	}
	data = data[encoding.StringBinarySize(v.Chain):]

	if x, err := encoding.UvarintUnmarshalBinary(data); err != nil {
		return fmt.Errorf("error decoding Block: %w", err)
	} else {
		v.Block = x
	}
	data = data[encoding.UvarintBinarySize(v.Block):]

	if x, err := encoding.UvarintUnmarshalBinary(data); err != nil {
		return fmt.Errorf("error decoding ChainEntry: %w", err)
	} else {
		v.ChainEntry = x
	}
	data = data[encoding.UvarintBinarySize(v.ChainEntry):]

	if x, err := encoding.UvarintUnmarshalBinary(data); err != nil {
		return fmt.Errorf("error decoding ChainAnchor: %w", err)
	} else {
		v.ChainAnchor = x
	}
	data = data[encoding.UvarintBinarySize(v.ChainAnchor):]

	if x, err := encoding.UvarintUnmarshalBinary(data); err != nil {
		return fmt.Errorf("error decoding RootEntry: %w", err)
	} else {
		v.RootEntry = x
	}
	data = data[encoding.UvarintBinarySize(v.RootEntry):]

	if x, err := encoding.UvarintUnmarshalBinary(data); err != nil {
		return fmt.Errorf("error decoding RootAnchor: %w", err)
	} else {
		v.RootAnchor = x
	}
	data = data[encoding.UvarintBinarySize(v.RootAnchor):]

	return nil
}

func (v *TransactionChainIndex) UnmarshalBinary(data []byte) error {
	var lenEntries uint64
	if x, err := encoding.UvarintUnmarshalBinary(data); err != nil {
		return fmt.Errorf("error decoding Entries: %w", err)
	} else {
		lenEntries = x
	}
	data = data[encoding.UvarintBinarySize(lenEntries):]

	v.Entries = make([]*TransactionChainEntry, lenEntries)
	for i := range v.Entries {
		var x *TransactionChainEntry
		x = new(TransactionChainEntry)
		if err := x.UnmarshalBinary(data); err != nil {
			return fmt.Errorf("error decoding Entries[%d]: %w", i, err)
		}
		data = data[x.BinarySize():]

		v.Entries[i] = x
	}

	return nil
}
