package protocol

// GENERATED BY go run ./tools/cmd/gen-types. DO NOT EDIT.

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"

	"gitlab.com/accumulatenetwork/accumulate/internal/encoding"
)

// NewAccount creates a new Account for the specified AccountType.
func NewAccount(typ AccountType) (Account, error) {
	switch typ {
	case AccountTypeIdentity:
		return new(ADI), nil
	case AccountTypeAnchor:
		return new(Anchor), nil
	case AccountTypeDataAccount:
		return new(DataAccount), nil
	case AccountTypeInternalLedger:
		return new(InternalLedger), nil
	case AccountTypeKeyBook:
		return new(KeyBook), nil
	case AccountTypeKeyPage:
		return new(KeyPage), nil
	case AccountTypeLiteDataAccount:
		return new(LiteDataAccount), nil
	case AccountTypeLiteIdentity:
		return new(LiteIdentity), nil
	case AccountTypeLiteTokenAccount:
		return new(LiteTokenAccount), nil
	case AccountTypePendingTransaction:
		return new(PendingTransactionState), nil
	case AccountTypeTokenAccount:
		return new(TokenAccount), nil
	case AccountTypeTokenIssuer:
		return new(TokenIssuer), nil
	case AccountTypeTransaction:
		return new(TransactionState), nil
	default:
		return nil, fmt.Errorf("unknown account type %v", typ)
	}
}

// UnmarshalAccountType unmarshals the AccountType from the start of a Account.
func UnmarshalAccountType(r io.Reader) (AccountType, error) {
	var typ AccountType
	err := encoding.UnmarshalEnumType(r, &typ)
	return typ, err
}

// UnmarshalAccount unmarshals a Account.
func UnmarshalAccount(data []byte) (Account, error) {
	typ, err := UnmarshalAccountType(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}

	v, err := NewAccount(typ)
	if err != nil {
		return nil, err
	}

	err = v.UnmarshalBinary(data)
	if err != nil {
		return nil, err
	}

	return v, nil
}

// UnmarshalAccountFrom unmarshals a Account.
func UnmarshalAccountFrom(rd io.ReadSeeker) (Account, error) {
	// Get the reader's current position
	pos, err := rd.Seek(0, io.SeekCurrent)
	if err != nil {
		return nil, err
	}

	// Read the type code
	typ, err := UnmarshalAccountType(rd)
	if err != nil {
		return nil, err
	}

	// Reset the reader's position
	_, err = rd.Seek(pos, io.SeekStart)
	if err != nil {
		return nil, err
	}

	// Create a new transaction result
	v, err := NewAccount(AccountType(typ))
	if err != nil {
		return nil, err
	}

	// Unmarshal the result
	err = v.UnmarshalBinaryFrom(rd)
	if err != nil {
		return nil, err
	}

	return v, nil
}

// UnmarshalAccountJson unmarshals a Account.
func UnmarshalAccountJSON(data []byte) (Account, error) {
	var typ struct{ Type AccountType }
	err := json.Unmarshal(data, &typ)
	if err != nil {
		return nil, err
	}

	acnt, err := NewAccount(typ.Type)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(data, acnt)
	if err != nil {
		return nil, err
	}

	return acnt, nil
}

// NewTransactionBody creates a new TransactionBody for the specified TransactionType.
func NewTransactionBody(typ TransactionType) (TransactionBody, error) {
	switch typ {
	case TransactionTypeAcmeFaucet:
		return new(AcmeFaucet), nil
	case TransactionTypeAddCredits:
		return new(AddCredits), nil
	case TransactionTypeBurnTokens:
		return new(BurnTokens), nil
	case TransactionTypeCreateDataAccount:
		return new(CreateDataAccount), nil
	case TransactionTypeCreateIdentity:
		return new(CreateIdentity), nil
	case TransactionTypeCreateKeyBook:
		return new(CreateKeyBook), nil
	case TransactionTypeCreateKeyPage:
		return new(CreateKeyPage), nil
	case TransactionTypeCreateToken:
		return new(CreateToken), nil
	case TransactionTypeCreateTokenAccount:
		return new(CreateTokenAccount), nil
	case TransactionTypeInternalGenesis:
		return new(InternalGenesis), nil
	case TransactionTypeInternalSendTransactions:
		return new(InternalSendTransactions), nil
	case TransactionTypeInternalTransactionsSent:
		return new(InternalTransactionsSent), nil
	case TransactionTypeInternalTransactionsSigned:
		return new(InternalTransactionsSigned), nil
	case TransactionTypeIssueTokens:
		return new(IssueTokens), nil
	case TransactionTypeRemoveManager:
		return new(RemoveManager), nil
	case TransactionTypeSegWitDataEntry:
		return new(SegWitDataEntry), nil
	case TransactionTypeSendTokens:
		return new(SendTokens), nil
	case TransactionTypeSignPending:
		return new(SignPending), nil
	case TransactionTypeSyntheticAnchor:
		return new(SyntheticAnchor), nil
	case TransactionTypeSyntheticBurnTokens:
		return new(SyntheticBurnTokens), nil
	case TransactionTypeSyntheticCreateChain:
		return new(SyntheticCreateChain), nil
	case TransactionTypeSyntheticDepositCredits:
		return new(SyntheticDepositCredits), nil
	case TransactionTypeSyntheticDepositTokens:
		return new(SyntheticDepositTokens), nil
	case TransactionTypeSyntheticMirror:
		return new(SyntheticMirror), nil
	case TransactionTypeSyntheticWriteData:
		return new(SyntheticWriteData), nil
	case TransactionTypeUpdateKeyPage:
		return new(UpdateKeyPage), nil
	case TransactionTypeUpdateManager:
		return new(UpdateManager), nil
	case TransactionTypeWriteData:
		return new(WriteData), nil
	case TransactionTypeWriteDataTo:
		return new(WriteDataTo), nil
	default:
		return nil, fmt.Errorf("unknown transaction type %v", typ)
	}
}

// UnmarshalTransactionType unmarshals the TransactionType from the start of a TransactionBody.
func UnmarshalTransactionType(r io.Reader) (TransactionType, error) {
	var typ TransactionType
	err := encoding.UnmarshalEnumType(r, &typ)
	return typ, err
}

// UnmarshalTransactionBody unmarshals a TransactionBody.
func UnmarshalTransactionBody(data []byte) (TransactionBody, error) {
	typ, err := UnmarshalTransactionType(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}

	v, err := NewTransactionBody(typ)
	if err != nil {
		return nil, err
	}

	err = v.UnmarshalBinary(data)
	if err != nil {
		return nil, err
	}

	return v, nil
}

// UnmarshalTransactionBodyFrom unmarshals a TransactionBody.
func UnmarshalTransactionBodyFrom(rd io.ReadSeeker) (TransactionBody, error) {
	// Get the reader's current position
	pos, err := rd.Seek(0, io.SeekCurrent)
	if err != nil {
		return nil, err
	}

	// Read the type code
	typ, err := UnmarshalTransactionType(rd)
	if err != nil {
		return nil, err
	}

	// Reset the reader's position
	_, err = rd.Seek(pos, io.SeekStart)
	if err != nil {
		return nil, err
	}

	// Create a new transaction result
	v, err := NewTransactionBody(TransactionType(typ))
	if err != nil {
		return nil, err
	}

	// Unmarshal the result
	err = v.UnmarshalBinaryFrom(rd)
	if err != nil {
		return nil, err
	}

	return v, nil
}

// UnmarshalTransactionBodyJson unmarshals a TransactionBody.
func UnmarshalTransactionBodyJSON(data []byte) (TransactionBody, error) {
	var typ struct{ Type TransactionType }
	err := json.Unmarshal(data, &typ)
	if err != nil {
		return nil, err
	}

	acnt, err := NewTransactionBody(typ.Type)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(data, acnt)
	if err != nil {
		return nil, err
	}

	return acnt, nil
}

// NewSignature creates a new Signature for the specified SignatureType.
func NewSignature(typ SignatureType) (Signature, error) {
	switch typ {
	case SignatureTypeED25519:
		return new(ED25519Signature), nil
	case SignatureTypeLegacyED25519:
		return new(LegacyED25519Signature), nil
	default:
		return nil, fmt.Errorf("unknown signature type %v", typ)
	}
}

// UnmarshalSignatureType unmarshals the SignatureType from the start of a Signature.
func UnmarshalSignatureType(r io.Reader) (SignatureType, error) {
	var typ SignatureType
	err := encoding.UnmarshalEnumType(r, &typ)
	return typ, err
}

// UnmarshalSignature unmarshals a Signature.
func UnmarshalSignature(data []byte) (Signature, error) {
	typ, err := UnmarshalSignatureType(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}

	v, err := NewSignature(typ)
	if err != nil {
		return nil, err
	}

	err = v.UnmarshalBinary(data)
	if err != nil {
		return nil, err
	}

	return v, nil
}

// UnmarshalSignatureFrom unmarshals a Signature.
func UnmarshalSignatureFrom(rd io.ReadSeeker) (Signature, error) {
	// Get the reader's current position
	pos, err := rd.Seek(0, io.SeekCurrent)
	if err != nil {
		return nil, err
	}

	// Read the type code
	typ, err := UnmarshalSignatureType(rd)
	if err != nil {
		return nil, err
	}

	// Reset the reader's position
	_, err = rd.Seek(pos, io.SeekStart)
	if err != nil {
		return nil, err
	}

	// Create a new transaction result
	v, err := NewSignature(SignatureType(typ))
	if err != nil {
		return nil, err
	}

	// Unmarshal the result
	err = v.UnmarshalBinaryFrom(rd)
	if err != nil {
		return nil, err
	}

	return v, nil
}

// UnmarshalSignatureJson unmarshals a Signature.
func UnmarshalSignatureJSON(data []byte) (Signature, error) {
	var typ struct{ Type SignatureType }
	err := json.Unmarshal(data, &typ)
	if err != nil {
		return nil, err
	}

	acnt, err := NewSignature(typ.Type)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(data, acnt)
	if err != nil {
		return nil, err
	}

	return acnt, nil
}
