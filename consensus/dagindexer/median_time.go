package dagindexer

import (
	"fmt"
	"sort"

	"github.com/0xsoniclabs/consensus/consensus"
)

// medianTimeIndex is a handy index for the MedianTime() func
type medianTimeIndex struct {
	weight       consensus.Weight
	creationTime Timestamp
}

// MedianTime calculates weighted median of claimed time within highest observed events.
func (vi *Index) MedianTime(id consensus.EventHash, defaultTime Timestamp) Timestamp {
	vi.InitBranchesInfo()
	// Get event by hash
	before := vi.GetMergedHighestBefore(id)
	if before == nil {
		vi.crit(fmt.Errorf("event=%s not found", id.String()))
	}

	honestTotalWeight := consensus.Weight(0) // isn't equal to validators.TotalWeight(), because doesn't count cheaters
	highests := make([]medianTimeIndex, 0, len(vi.validatorIdxs))
	// convert []HighestBefore -> []medianTimeIndex
	for creatorIdxI := range vi.validators.IDs() {
		creatorIdx := consensus.ValidatorIndex(creatorIdxI)
		highest := medianTimeIndex{}
		highest.weight = vi.validators.GetWeightByIdx(creatorIdx)
		highest.creationTime = before.VTime.Get(creatorIdx)
		seq := before.VSeq.Get(creatorIdx)

		// edge cases
		if seq.IsForkDetected() {
			// cheaters don't influence medianTime
			highest.weight = 0
		} else if seq.Seq == 0 {
			// if no event was observed from this node, then use genesisTime
			highest.creationTime = defaultTime
		}

		highests = append(highests, highest)
		honestTotalWeight += highest.weight
	}
	// it's technically possible honestTotalWeight == 0 (all validators are cheaters)

	// sort by claimed time (partial order is enough here, because we need only creationTime)
	sort.Slice(highests, func(i, j int) bool {
		a, b := highests[i], highests[j]
		return a.creationTime < b.creationTime
	})

	// Calculate weighted median
	halfWeight := honestTotalWeight / 2
	var currWeight consensus.Weight
	var median Timestamp
	for _, highest := range highests {
		currWeight += highest.weight
		if currWeight >= halfWeight {
			median = highest.creationTime
			break
		}
	}

	// sanity check
	if currWeight < halfWeight || currWeight > honestTotalWeight {
		vi.crit(fmt.Errorf("median wasn't calculated correctly, median=%d, currWeight=%d, totalWeight=%d, len(highests)=%d, id=%s",
			median,
			currWeight,
			honestTotalWeight,
			len(highests),
			id.String(),
		))
	}

	return median
}
