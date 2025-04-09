// Copyright (c) 2025 Fantom Foundation
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at fantom.foundation/bsl11.
//
// Change Date: 2028-4-16
//
// On the date above, in accordance with the Business Source License, use of
// this software will be governed by the GNU Lesser General Public License v3.

package consensus

import (
	"fmt"
	"math/big"
	"reflect"
	"strings"

	"github.com/ethereum/go-ethereum/common/hexutil"
)

const (
	// HashLength is the expected length of the hash
	HashLength = 32
)

var (
	// Zero is an empty hash.
	Zero  = Hash{}
	hashT = reflect.TypeOf(Hash{})
)

// Hash represents the 32 byte hash of arbitrary data.
type Hash [HashLength]byte

type Hashes []Hash

type HashesSet map[Hash]struct{}

// BytesToHash sets b to hash.
// If b is larger than len(h), b will be cropped from the left.
func BytesToHash(b []byte) Hash {
	var h Hash
	h.SetBytes(b)
	return h
}

// BigToHash sets byte representation of b to hash.
// If b is larger than len(h), b will be cropped from the left.
func BigToHash(b *big.Int) Hash { return BytesToHash(b.Bytes()) }

// HexToHash sets byte representation of s to hash.
// If b is larger than len(h), b will be cropped from the left.
func HexToHash(s string) Hash { return BytesToHash(hexutil.MustDecode(s)) }

// Bytes gets the byte representation of the underlying hash.
func (h Hash) Bytes() []byte { return h[:] }

// Big converts a hash to a big integer.
func (h Hash) Big() *big.Int { return new(big.Int).SetBytes(h[:]) }

// Hex converts a hash to a hex string.
func (h Hash) Hex() string { return hexutil.Encode(h[:]) }

// TerminalString implements log.TerminalStringer, formatting a string for console
// output during logging.
func (h Hash) TerminalString() string {
	return fmt.Sprintf("%x…%x", h[:3], h[29:])
}

// String implements the stringer interface and is used also by the logger when
// doing full logging into a file.
func (h Hash) String() string {
	return h.Hex()
}

// UnmarshalText parses a hash in hex syntax.
func (h *Hash) UnmarshalText(input []byte) error {
	return hexutil.UnmarshalFixedText("Hash", input, h[:])
}

// UnmarshalJSON parses a hash in hex syntax.
func (h *Hash) UnmarshalJSON(input []byte) error {
	return hexutil.UnmarshalFixedJSON(hashT, input, h[:])
}

// MarshalText returns the hex representation of h.
func (h Hash) MarshalText() ([]byte, error) {
	return hexutil.Bytes(h[:]).MarshalText()
}

// setBytes sets the hash to the value of b.
// If b is larger than len(h), b will be cropped from the left.
func (h *Hash) SetBytes(b []byte) {
	if len(b) > len(h) {
		b = b[len(b)-HashLength:]
	}

	copy(h[HashLength-len(b):], b)
}

/*
 * HashesSet methods:
 */

// NewHashesSet makes hash index.
func NewHashesSet(h ...Hash) HashesSet {
	hh := HashesSet{}
	hh.Add(h...)
	return hh
}

// Copy copies hashes to a new structure.
func (hh HashesSet) Copy() HashesSet {
	ee := make(HashesSet, len(hh))
	for k, v := range hh {
		ee[k] = v
	}

	return ee
}

// String returns human readable string representation.
func (hh HashesSet) String() string {
	ss := make([]string, 0, len(hh))
	for h := range hh {
		ss = append(ss, h.String())
	}
	return "[" + strings.Join(ss, ", ") + "]"
}

// Slice returns whole index as slice.
func (hh HashesSet) Slice() Hashes {
	arr := make(Hashes, len(hh))
	i := 0
	for h := range hh {
		arr[i] = h
		i++
	}
	return arr
}

// Add appends hash to the index.
func (hh HashesSet) Add(hash ...Hash) {
	for _, h := range hash {
		hh[h] = struct{}{}
	}
}

// Erase erase hash from the index.
func (hh HashesSet) Erase(hash ...Hash) {
	for _, h := range hash {
		delete(hh, h)
	}
}

// Contains returns true if hash is in.
func (hh HashesSet) Contains(hash Hash) bool {
	_, ok := hh[hash]
	return ok
}

/*
 * Hashes methods:
 */

// NewHashes makes hash slice.
func NewHashes(h ...Hash) Hashes {
	hh := Hashes{}
	hh.Add(h...)
	return hh
}

// Copy copies hashes to a new structure.
func (hh Hashes) Copy() Hashes {
	return append(Hashes(nil), hh...)
}

// String returns human readable string representation.
func (hh Hashes) String() string {
	ss := make([]string, 0, len(hh))
	for _, h := range hh {
		ss = append(ss, h.String())
	}
	return "[" + strings.Join(ss, ", ") + "]"
}

// Set returns whole index as a HashesSet.
func (hh Hashes) Set() HashesSet {
	set := make(HashesSet, len(hh))
	for _, h := range hh {
		set[h] = struct{}{}
	}
	return set
}

// Add appends hash to the slice.
func (hh *Hashes) Add(hash ...Hash) {
	*hh = append(*hh, hash...)
}
