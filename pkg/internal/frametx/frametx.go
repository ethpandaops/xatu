// Package frametx decodes EIP-8141 frame transactions (type 0x06), which go-ethereum cannot decode.
//
// The encoding is daisugi-devnet-0's, set by Nethermind feature/eip8288-lean at af50ed19: EIP-8141 with
// EIP-8250 keyed nonces, where nonce_keys is left out entirely for a transaction using the account nonce.
// https://github.com/NethermindEth/nethermind/blob/af50ed19/src/Nethermind/Nethermind.Serialization.Rlp/TxDecoders/FrameTxDecoder.cs
package frametx

import (
	"bytes"
	"errors"
	"fmt"
	"math"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rlp"
)

// TxType is the EIP-2718 type byte of a frame transaction.
const TxType = 0x06

const gasPerBlob = 131072

// Transaction is a decoded frame transaction:
// [chain_id, nonce_keys (optional), nonce_seq, sender, frames, signatures, fees, blob_versioned_hashes].
type Transaction struct {
	ChainID *big.Int
	// NonceKeys selects keyed nonce domains; nil when the transaction leaves it out and uses the account nonce.
	NonceKeys           []*big.Int
	NonceSeq            uint64
	Sender              common.Address
	Frames              []Frame
	Signatures          []rlp.RawValue
	Fees                Fees
	BlobVersionedHashes []common.Hash

	// Hash is keccak256(0x06 || payload), the transaction hash.
	Hash common.Hash
	// Size is the length of the typed envelope, without any blob sidecar.
	Size uint64
	// Blobs holds the sidecar blobs when the transaction came in the EIP-7594 network wrapper.
	Blobs [][]byte
}

// Frame is [mode, flags, target, limits, value, data].
type Frame struct {
	Mode   uint64
	Flags  uint64
	Target []byte // empty means the sender
	Limits Limits
	Value  *big.Int
	Data   []byte
}

// Limits is a frame's [execution, state] gas limits.
type Limits struct {
	Execution uint64
	State     uint64
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

	elems, rest, err := rlp.SplitList(raw[1:])
	if err != nil {
		return nil, fmt.Errorf("frame transaction is not an RLP list: %w", err)
	}

	if len(rest) != 0 {
		return nil, errors.New("frame transaction has trailing bytes")
	}

	payload := raw[1:]

	var blobs [][]byte

	// The plain payload starts with chain_id, a string; the wrapper starts with the payload, a list.
	if kind, _, _, splitErr := rlp.Split(elems); splitErr == nil && kind == rlp.List {
		var wrapper networkWrapper
		if err = rlp.DecodeBytes(payload, &wrapper); err != nil {
			return nil, fmt.Errorf("decoding frame transaction network wrapper: %w", err)
		}

		payload, blobs = wrapper.Payload, wrapper.Blobs
	}

	tx, err := decodePayload(payload)
	if err != nil {
		return nil, fmt.Errorf("decoding frame transaction: %w", err)
	}

	if len(tx.Frames) == 0 {
		return nil, errors.New("frame transaction has no frames")
	}

	tx.Hash = crypto.Keccak256Hash([]byte{TxType}, payload)
	tx.Size = uint64(1 + len(payload))
	tx.Blobs = blobs

	return tx, nil
}

func decodePayload(payload []byte) (*Transaction, error) {
	tx := new(Transaction)
	s := rlp.NewStream(bytes.NewReader(payload), uint64(len(payload)))

	if _, err := s.List(); err != nil {
		return nil, err
	}

	if err := s.Decode(&tx.ChainID); err != nil {
		return nil, fmt.Errorf("chain_id: %w", err)
	}

	// nonce_keys is a list when present; nonce_seq, the next field, is an integer.
	kind, _, err := s.Kind()
	if err != nil {
		return nil, fmt.Errorf("nonce: %w", err)
	}

	if kind == rlp.List {
		if err := s.Decode(&tx.NonceKeys); err != nil {
			return nil, fmt.Errorf("nonce_keys: %w", err)
		}
	}

	fields := []struct {
		name string
		ptr  any
	}{
		{"nonce_seq", &tx.NonceSeq},
		{"sender", &tx.Sender},
		{"frames", &tx.Frames},
		{"signatures", &tx.Signatures},
		{"fees", &tx.Fees},
		{"blob_versioned_hashes", &tx.BlobVersionedHashes},
	}

	for _, field := range fields {
		if err := s.Decode(field.ptr); err != nil {
			return nil, fmt.Errorf("%s: %w", field.name, err)
		}
	}

	if err := s.ListEnd(); err != nil {
		return nil, err
	}

	return tx, nil
}

// Gas is the sum of the frames' execution and state gas limits, saturating at 2^64-1: the transaction's
// gas as Nethermind reports it. The gas the payer reserves, max_gas, adds intrinsic and data costs to it.
func (tx *Transaction) Gas() uint64 {
	var total uint64

	for _, frame := range tx.Frames {
		for _, limit := range []uint64{frame.Limits.Execution, frame.Limits.State} {
			if total > math.MaxUint64-limit {
				return math.MaxUint64
			}

			total += limit
		}
	}

	return total
}

// GasPrice is the gas price Nethermind reports for the transaction at baseFee:
// min(max_fee // gas, base_fee + max_priority_fee_per_gas), with gas from Gas.
func (tx *Transaction) GasPrice(baseFee *big.Int) *big.Int {
	maxFeePerGas := new(big.Int).Set(tx.Fees.MaxFee)
	if gas := tx.Gas(); gas != 0 {
		maxFeePerGas.Div(maxFeePerGas, new(big.Int).SetUint64(gas))
	}

	price := new(big.Int).Add(baseFee, tx.Fees.MaxPriorityFeePerGas)
	if price.Cmp(maxFeePerGas) > 0 {
		return maxFeePerGas
	}

	return price
}

// BlobGas is the transaction's blob gas: GAS_PER_BLOB per versioned hash.
func (tx *Transaction) BlobGas() uint64 {
	return uint64(len(tx.BlobVersionedHashes)) * gasPerBlob
}
