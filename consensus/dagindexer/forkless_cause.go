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
	"fmt"

	"github.com/0xsoniclabs/consensus/consensus"
)

type kv struct {
	a, b consensus.EventHash
}

// ForklessCause calculates "sufficient coherence" between the events.
// The A.HighestBefore array remembers the sequence number of the last
// event by each validator that is an ancestor of A. The array for
// B.LowestAfter remembers the sequence number of the earliest
// event by each validator that is a descendant of B. Compare the two arrays,
// and find how many elements in the A.HighestBefore array are greater
// than or equal to the corresponding element of the B.LowestAfter
// array. If there are more than 2n/3 such matches, then the A and B
// have achieved sufficient coherency.
//
// If B1 and B2 are forks, then they cannot BOTH forkless-cause any specific event A,
// unless more than 1/3W are Byzantine.
// This great property is the reason why this function exists,
// providing the base for the BFT algorithm.
func (vi *Index) ForklessCause(aID, bID consensus.EventHash) bool {
	if res, ok := vi.cache.ForklessCause.Get(kv{aID, bID}); ok {
		return res.(bool)
	}

	vi.InitBranchesInfo()
	res := vi.forklessCause(aID, bID)

	vi.cache.ForklessCause.Add(kv{aID, bID}, res, 1)
	return res
}

func (vi *Index) forklessCause(aID, bID consensus.EventHash) bool {
	// Get events by hash
	a := vi.GetHighestBefore(aID).VSeq
	if a == nil {
		vi.crit(fmt.Errorf("event A=%s not found", aID.String()))
		return false
	}

	// check A doesn't observe any forks from B
	if vi.AtLeastOneFork() {
		bBranchID := vi.GetEventBranchID(bID)
		if a.Get(bBranchID).IsForkDetected() { // B is observed as cheater by A
			return false
		}
	}

	// check A observes that {QUORUM} non-cheater-validators observe B
	b := vi.GetLowestAfter(bID)
	if b == nil {
		vi.crit(fmt.Errorf("event B=%s not found", bID.String()))
		return false
	}

	yes := vi.validators.NewCounter()
	// calculate forkless causing using the indexes
	branchIDs := vi.BranchesInfo().BranchIDCreatorIdxs
	for branchIDint, creatorIdx := range branchIDs {
		branchID := consensus.ValidatorIndex(branchIDint)

		// bLowestAfter := vi.GetLowestAfterSeq_(bID, branchID)   // lowest event from creator on branchID, which observes B
		bLowestAfter := b.Get(branchID)   // lowest event from creator on branchID, which observes B
		aHighestBefore := a.Get(branchID) // highest event from creator, observed by A

		// if lowest event from branchID which observes B <= highest from branchID observed by A
		// then {highest from branchID observed by A} observes B
		if bLowestAfter <= aHighestBefore.Seq && bLowestAfter != 0 && !aHighestBefore.IsForkDetected() {
			// we may count the same creator multiple times (on different branches)!
			// so not every call increases the counter
			yes.CountVoteByIndex(creatorIdx)
		}
	}
	return yes.HasQuorum()
}

func (vi *Index) ForklessCauseProgress(aID, bID consensus.EventHash, candidateParents, chosenParents consensus.EventHashes) (*consensus.WeightCounter, []*consensus.WeightCounter) {
	// This function is used to determine progress of event bID in forkless causing aID.
	// It may be used to determine progress toward the forkless cause condition for an event not in vi, but whose parents are in vi.
	// To do so, aID should be the self-parent while chosenParents should be the parents of the not-yet-created event.
	// Further, this function can be used to determine the incremental improvement in progress toward satisfying the forkless
	// cause condition beyond the progress of aId and chosenParents, obtained by inclusion of one additional candidate head at a time.
	// This function is useful in parent selection and event creation timing.

	// The first return is ForklessCause(a + chosenParents, b).
	// The second return argument is a slice containing ForklessCause(a + chosenParents + candidateParent, b) with each element in the
	// slice corresponding to each candidate parent in candidateParents.

	// create the counters that measure the forkless cause progress
	candidateParentsFCProgress := make([]*consensus.WeightCounter, len(candidateParents))
	for i := range candidateParentsFCProgress {
		candidateParentsFCProgress[i] = vi.validators.NewCounter() // initialise the counter for each candidate parent
	}
	chosenParentsFCProgress := vi.validators.NewCounter() // initialise the counter for chosen parents only

	// Get events by hash
	aHB := vi.GetHighestBefore(aID).VSeq
	if aHB == nil {
		vi.crit(fmt.Errorf("event A=%s not found", aID.String()))
		return chosenParentsFCProgress, candidateParentsFCProgress
	}

	candidateParentsHB := make([]*HighestBeforeSeq, len(candidateParents))
	for i := range candidateParents {
		candidateParentsHB[i] = vi.GetHighestBefore(candidateParents[i]).VSeq
		if candidateParentsHB[i] == nil {
			vi.crit(fmt.Errorf("candidate parent=%s not found", candidateParents[i].String()))
			return chosenParentsFCProgress, candidateParentsFCProgress
		}
	}

	chosenParentsHB := make([]*HighestBeforeSeq, len(chosenParents))
	for i := range chosenParents {
		chosenParentsHB[i] = vi.GetHighestBefore(chosenParents[i]).VSeq
		if chosenParentsHB[i] == nil {
			vi.crit(fmt.Errorf("chosen parent=%s not found", chosenParents[i].String()))
			return chosenParentsFCProgress, candidateParentsFCProgress
		}
	}

	// check A doesn't observe any forks from B
	if vi.AtLeastOneFork() {
		bBranchID := vi.GetEventBranchID(bID)
		if aHB.Get(bBranchID).IsForkDetected() { // B is observed as cheater by A
			return chosenParentsFCProgress, candidateParentsFCProgress
		}
	}

	// check chosenParents don't observe any forks from B
	for i := 0; i < len(chosenParentsHB); i++ {
		if vi.AtLeastOneFork() {
			bBranchID := vi.GetEventBranchID(bID)
			if chosenParentsHB[i].Get(bBranchID).IsForkDetected() { // B is observed as cheater by a chosen parent
				return chosenParentsFCProgress, candidateParentsFCProgress
			}
		}
	}

	// check candidateParents don't observe any forks from B
	for i := 0; i < len(candidateParentsHB); i++ {
		if vi.AtLeastOneFork() {
			bBranchID := vi.GetEventBranchID(bID)
			if candidateParentsHB[i].Get(bBranchID).IsForkDetected() { // B is observed as cheater by a candidate parent
				return chosenParentsFCProgress, candidateParentsFCProgress
			}
		}
	}

	bLA := vi.GetLowestAfter(bID)
	if bLA == nil {
		vi.crit(fmt.Errorf("event B=%s not found", bID.String()))
		return chosenParentsFCProgress, candidateParentsFCProgress
	}

	// calculate forkless causing using the indexes
	branchIDs := vi.BranchesInfo().BranchIDCreatorIdxs
	for branchIDint, creatorIdx := range branchIDs {
		branchID := consensus.ValidatorIndex(branchIDint)

		// bLowestAfter := vi.GetLowestAfterSeq_(bID, branchID)   // lowest event from creator on branchID, which observes B
		bLowestAfter := bLA.Get(branchID)  // lowest event from creator on branchID, which observes B
		HighestBefore := aHB.Get(branchID) // highest event from creator, observed by A

		IsForkDetected := HighestBefore.IsForkDetected()

		for i := range chosenParents {
			chosenParentHighestBefore := chosenParentsHB[i].Get(branchID)                  // highest event from creator, observed by a chosen parent
			HighestBefore.Seq = maxEvent(HighestBefore.Seq, chosenParentHighestBefore.Seq) // find HighestBefore as observed by a and all chosen parents
			IsForkDetected = IsForkDetected || chosenParentHighestBefore.IsForkDetected()
		}

		// first do forkless cause for a + chosenParents only
		if bLowestAfter <= HighestBefore.Seq && bLowestAfter != 0 && !IsForkDetected {
			// we may count the same creator multiple times (on different branches)!
			// so not every call increases the counter
			chosenParentsFCProgress.CountVoteByIndex(creatorIdx)
		}
		// now do forkless cause for a + chosenParents + each candidate parent
		for i := range candidateParents {
			candidateParentHighestBefore := candidateParentsHB[i].Get(branchID)
			candidateParentIsForkDetected := IsForkDetected || candidateParentHighestBefore.IsForkDetected()
			candidateParentHighestBefore.Seq = maxEvent(HighestBefore.Seq, candidateParentHighestBefore.Seq)

			if bLowestAfter <= candidateParentHighestBefore.Seq && bLowestAfter != 0 && !candidateParentIsForkDetected {
				// we may count the same creator multiple times (on different branches)!
				// so not every call increases the counter
				candidateParentsFCProgress[i].CountVoteByIndex(creatorIdx)
			}
		}
	}
	// We want FC progress for new candidate events with parents aID + chosenParents + head
	// aID may not contribute to forkless cause without the heads,
	// but may contribute with the heads. HighestBefore and LowestAfter used above do not incorporate
	// these potential new events, so ensure the contribution of aID's creator is checked and made here
	aCreatorID := vi.getEvent(aID).Creator()
	for _, FC := range candidateParentsFCProgress {
		if FC.Sum() > 0 { // if anything in candidate event's subgraph observes bID, then the candidate must too
			FC.CountVoteByID(aCreatorID)
		}
	}
	return chosenParentsFCProgress, candidateParentsFCProgress
}

func maxEvent(a consensus.Seq, b consensus.Seq) consensus.Seq {
	if a > b {
		return a
	}
	return b
}
