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
	"math/big"
)

// ValidatorsBuilderBig is a helper to create Validators object out of bigint numbers
type ValidatorsBigBuilder map[ValidatorID]*big.Int

// NewBigBuilder creates new mutable ValidatorsBigBuilder
func NewBigBuilder() ValidatorsBigBuilder {
	return ValidatorsBigBuilder{}
}

// Set appends item to ValidatorsBuilder object
func (vv ValidatorsBigBuilder) Set(id ValidatorID, weight *big.Int) {
	if weight == nil || weight.Sign() == 0 {
		delete(vv, id)
	} else {
		vv[id] = weight
	}
}

// Build new read-only Validators object
func (vv ValidatorsBigBuilder) TotalWeight() *big.Int {
	res := new(big.Int)
	for _, w := range vv {
		res = res.Add(res, w)
	}
	return res
}

// Build new read-only Validators object
func (vv ValidatorsBigBuilder) Build() *Validators {
	totalBits := vv.TotalWeight().BitLen()
	// use downscaling by a 2^n ratio, instead of n for simplicity and performance reasons
	shift := uint(0)
	if totalBits > 31 {
		shift = uint(totalBits - 31)
	}

	builder := NewValidatorsBuilder()
	for v, w := range vv {
		weight := new(big.Int).Rsh(w, shift)
		builder.Set(v, Weight(weight.Uint64()))
	}
	return builder.Build()
}
