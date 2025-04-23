// Copyright (c) 2025 Fantom Foundation
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at fantom.foundation/bsl11.
//
// Change Date: 2028-4-16
//
// On the date above, in accordance with the Business Source License, use of
// this software will be governed by the GNU Lesser General Public License v3.

package dagindexer

import (
	"github.com/0xsoniclabs/consensus/consensus"
)

// BranchesInfo contains information about global branches of each validator
type BranchesInfo struct {
	BranchIDLastSeq     []consensus.Seq              // branchID -> highest e.Seq in the branch
	BranchIDCreatorIdxs []consensus.ValidatorIndex   // branchID -> validator idx
	BranchIDByCreators  [][]consensus.ValidatorIndex // validator idx -> list of branch IDs
}

// InitBranchesInfo loads BranchesInfo from store
func (vi *Index) InitBranchesInfo() {
	if vi.branchesInfo == nil {
		// if not cached
		vi.branchesInfo = vi.getBranchesInfo()
		if vi.branchesInfo == nil {
			// first run
			vi.branchesInfo = newInitialBranchesInfo(vi.validators)
		}
	}
}

func newInitialBranchesInfo(validators *consensus.Validators) *BranchesInfo {
	branchIDCreators := validators.SortedIDs()
	branchIDCreatorIdxs := make([]consensus.ValidatorIndex, len(branchIDCreators))
	for i := range branchIDCreators {
		branchIDCreatorIdxs[i] = consensus.ValidatorIndex(i)
	}

	branchIDLastSeq := make([]consensus.Seq, len(branchIDCreatorIdxs))
	branchIDByCreators := make([][]consensus.ValidatorIndex, validators.Len())
	for i := range branchIDByCreators {
		branchIDByCreators[i] = make([]consensus.ValidatorIndex, 1, validators.Len()/2+1)
		branchIDByCreators[i][0] = consensus.ValidatorIndex(i)
	}
	return &BranchesInfo{
		BranchIDLastSeq:     branchIDLastSeq,
		BranchIDCreatorIdxs: branchIDCreatorIdxs,
		BranchIDByCreators:  branchIDByCreators,
	}
}

func (vi *Index) AtLeastOneFork() bool {
	return consensus.ValidatorIndex(len(vi.branchesInfo.BranchIDCreatorIdxs)) > vi.validators.Len()
}

func (vi *Index) BranchesInfo() *BranchesInfo {
	return vi.branchesInfo
}
