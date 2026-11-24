// Package frametx decodes EIP-8141 frame transactions (type 0x06), which go-ethereum cannot decode.
//
// The encoding and gas rules are bogota-devnet-2's: EIP-8141 at commit f1bcca1b with EIP-8250 keyed
// nonces at commit 1a086d5c, which replaces nonce with nonce_keys and nonce_seq.
// https://github.com/ethereum/EIPs/blob/f1bcca1b56ee91fe115d2a8d8fd731fd6ca38bb3/EIPS/eip-8141.md
// https://github.com/ethereum/EIPs/blob/1a086d5c67cdcf3a134cc571eb4bb4b9bed6fb02/EIPS/eip-8250.md
package frametx

import (
	"errors"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rlp"
)

// TxType is the EIP-2718 type byte of a frame transaction.
const TxType = 0x06

const (
	intrinsicCost          = 12000
	perFrameCost           = 475
	txValueCost            = 6000
	standardTokenCost      = 4
	totalCostFloorPerToken = 16
	gasPerBlob             = 131072
	keyedNonceAccessCost   = 12100 // COLD_STORAGE_ACCESS + STORAGE_WRITE
	maxNonceSeq            = ^uint64(0)

	schemeArbitrary = 0
	schemeSecp256k1 = 1
	schemeP256      = 2
)

// Transaction is a decoded frame transaction:
// [chain_id, nonce_keys, nonce_seq, sender, frames, signatures, fees, blob_versioned_hashes].
type Transaction struct {
	ChainID *big.Int
	// NonceKeys selects keyed nonce domains; empty selects the sender's account nonce.
	NonceKeys           []*big.Int
	NonceSeq            uint64
	Sender              common.Address
	Frames              []Frame
	Signatures          []Signature
	Fees                Fees
	BlobVersionedHashes []common.Hash

	// Hash is keccak256(0x06 || payload), the transaction hash.
	Hash common.Hash `rlp:"-"`
	// Size is the length of the typed envelope, without any blob sidecar.
	Size uint64 `rlp:"-"`
	// Blobs holds the sidecar blobs when the transaction came in the EIP-7594 network wrapper.
	Blobs [][]byte `rlp:"-"`
}

// Frame is [mode, flags, target, limits, value, data].
type Frame struct {
	Mode   uint64
	Flags  uint64
	Target []byte // empty means no target
	Limits Limits
	Value  *big.Int
	Data   []byte
}

// Limits is a frame's [execution, state] gas limits.
type Limits struct {
	Execution uint64
	State     uint64
}

// Signature is [scheme, signer, msg, signature].
type Signature struct {
	Scheme    uint64
	Signer    []byte
	Msg       []byte
	Signature []byte
}

// Fees is [max_fee, max_priority_fee_per_gas]; max_fee is a total wei budget, not a per-gas cap.
type Fees struct {
	MaxFee               *big.Int
	MaxPriorityFeePerGas *big.Int
}

// networkWrapper is the PooledTransactions form of a blob-carrying frame transaction:
// [tx_payload_body, wrapper_version, blobs, commitments, cell_proofs].
type networkWrapper struct {
	Payload     rlp.RawValue
	Version     uint64
	Blobs       [][]byte
	Commitments [][]byte
	CellProofs  [][]byte
}

// Decode decodes a typed frame transaction envelope, in its plain form or in the network wrapper.
func Decode(raw []byte) (*Transaction, error) {
	if len(raw) == 0 || raw[0] != TxType {
		return nil, errors.New("not a frame transaction")
	}

	// Trailing bytes are rejected by rlp.DecodeBytes below.
	elems, _, err := rlp.SplitList(raw[1:])
	if err != nil {
		return nil, fmt.Errorf("frame transaction is not an RLP list: %w", err)
	}

	payload := raw[1:]

	var blobs [][]byte

	// The plain payload starts with chain_id, a string; the wrapper starts with the payload, a list.
	if kind, _, _, err := rlp.Split(elems); err == nil && kind == rlp.List {
		var wrapper networkWrapper
		if err := rlp.DecodeBytes(payload, &wrapper); err != nil {
			return nil, fmt.Errorf("decoding frame transaction network wrapper: %w", err)
		}

		payload, blobs = wrapper.Payload, wrapper.Blobs
	}

	tx := new(Transaction)
	if err := rlp.DecodeBytes(payload, tx); err != nil {
		return nil, fmt.Errorf("decoding frame transaction: %w", err)
	}

	if len(tx.Frames) == 0 || tx.Fees.MaxFee == nil || tx.Fees.MaxPriorityFeePerGas == nil {
		return nil, errors.New("frame transaction has no frames or fees")
	}

	tx.Hash = crypto.Keccak256Hash([]byte{TxType}, payload)
	tx.Size = uint64(1 + len(payload))
	tx.Blobs = blobs

	return tx, nil
}

// BlobGas is the transaction's blob gas: GAS_PER_BLOB per versioned hash.
func (tx *Transaction) BlobGas() uint64 {
	return uint64(len(tx.BlobVersionedHashes)) * gasPerBlob
}

// MaxGas is the transaction's total gas limit, max_gas in the EIP, derived from its fields alone.
func (tx *Transaction) MaxGas() (uint64, error) {
	// EIP-8250 prices the nonce fields as data, except nonce_seq on single-use keys.
	nonceData, err := rlp.EncodeToBytes(tx.NonceKeys)
	if err != nil {
		return 0, err
	}

	if len(tx.NonceKeys) == 0 || tx.NonceSeq != maxNonceSeq {
		seq, err := rlp.EncodeToBytes(tx.NonceSeq)
		if err != nil {
			return 0, err
		}

		nonceData = append(nonceData, seq...)
	}

	var (
		frames       = uint64(len(tx.Frames))
		keyedNonces  = uint64(len(tx.NonceKeys)) * keyedNonceAccessCost
		verification uint64
		valueCost    uint64
		frameLimits  uint64
		stateLimits  uint64
		dataCost     = standardTokenCost * tokens(nonceData)
		floorCount   = floorTokens(nonceData)
	)

	for _, sig := range tx.Signatures {
		switch sig.Scheme {
		case schemeSecp256k1:
			verification += 2800
		case schemeP256:
			verification += 6700
		case schemeArbitrary:
			verification += 100
		default:
			return 0, fmt.Errorf("unknown signature scheme %d", sig.Scheme)
		}

		// Signature bytes elided from the signature hash (empty msg) are priced as all non-zero.
		sigTokens := tokens(sig.Signature)
		if len(sig.Msg) == 0 {
			sigTokens = floorTokens(sig.Signature)
		}

		dataCost += standardTokenCost * (tokens(sig.Signer) + tokens(sig.Msg) + sigTokens)
		floorCount += floorTokens(sig.Signer) + floorTokens(sig.Msg) + floorTokens(sig.Signature)
	}

	for _, frame := range tx.Frames {
		if frame.Value != nil && frame.Value.Sign() > 0 && len(frame.Target) != 0 &&
			common.BytesToAddress(frame.Target) != tx.Sender {
			valueCost += txValueCost
		}

		limits := frame.Limits.Execution + frame.Limits.State
		if limits < frame.Limits.Execution || frameLimits+limits < frameLimits {
			return 0, errors.New("frame gas limits overflow")
		}

		frameLimits += limits
		stateLimits += frame.Limits.State
		dataCost += standardTokenCost * tokens(frame.Data)
		floorCount += floorTokens(frame.Data)
	}

	base := intrinsicCost + frames*perFrameCost + verification + valueCost + keyedNonces
	standard := base + dataCost + frameLimits
	floor := base + totalCostFloorPerToken*floorCount + stateLimits

	if standard < frameLimits {
		return 0, errors.New("gas limit overflow")
	}

	return max(standard, floor), nil
}

// GasPrice is the per-gas price the transaction pays (the GASPRICE opcode value) at baseFee:
// base_fee + min(max_priority_fee_per_gas * max_gas, max_fee - max_gas * base_fee) // max_gas.
// It needs the blob base fee when the transaction carries blobs, so those are rejected.
func (tx *Transaction) GasPrice(baseFee *big.Int) (*big.Int, error) {
	if len(tx.BlobVersionedHashes) > 0 {
		return nil, errors.New("gas price of a blob-carrying frame transaction needs the blob base fee")
	}

	maxGas, err := tx.MaxGas()
	if err != nil {
		return nil, err
	}

	if maxGas == 0 {
		return nil, errors.New("frame transaction has zero gas")
	}

	gas := new(big.Int).SetUint64(maxGas)
	budget := new(big.Int).Sub(tx.Fees.MaxFee, new(big.Int).Mul(gas, baseFee))

	if budget.Sign() < 0 {
		return nil, errors.New("frame transaction max fee is below the required fee")
	}

	priority := new(big.Int).Mul(tx.Fees.MaxPriorityFeePerGas, gas)
	if priority.Cmp(budget) > 0 {
		priority = budget
	}

	return priority.Div(priority, gas).Add(priority, baseFee), nil
}

// tokens is EIP-7976's weighted token count: zero bytes count one, other bytes four.
func tokens(data []byte) uint64 {
	n := uint64(0)

	for _, b := range data {
		if b == 0 {
			n++
		} else {
			n += 4
		}
	}

	return n
}

// floorTokens counts every byte as four tokens.
func floorTokens(data []byte) uint64 {
	return uint64(len(data)) * 4
}
