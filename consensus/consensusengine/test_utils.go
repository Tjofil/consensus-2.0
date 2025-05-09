// Copyright (c) 2025 Fantom Foundation
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at fantom.foundation/bsl11.
//
// Change Date: 2028-4-16
//
// On the date above, in accordance with the Business Source License, use of
// this software will be governed by the GNU Lesser General Public License v3.

package consensusengine

import (
	"github.com/0xsoniclabs/consensus/consensus/dagindexer"

	"github.com/0xsoniclabs/consensus/consensus"
	"github.com/0xsoniclabs/consensus/consensus/consensusstore"
	"github.com/0xsoniclabs/consensus/consensus/consensustest"
	"github.com/0xsoniclabs/kvdb/memorydb"
)

type applyBlockFn func(block *consensus.Block) *consensus.Validators

type BlockKey struct {
	Epoch consensus.Epoch
	Frame consensus.Frame
}

type BlockResult struct {
	Atropos    consensus.EventHash
	Cheaters   consensus.Cheaters
	Validators *consensus.Validators
}

// CoreLachesis extends Indexed Orderer for tests.
type CoreLachesis struct {
	*IndexedLachesis

	blocks      map[BlockKey]*BlockResult
	lastBlock   BlockKey
	epochBlocks map[consensus.Epoch]consensus.Frame
	applyBlock  applyBlockFn
}

// NewBootstrappedCoreConsensus creates a simple bootstrapped consensus engine with mem store and optional node weights w.o. some callbacks usually instantiated by the Client
func NewBootstrappedCoreConsensus(
	nodes []consensus.ValidatorID,
	weights []consensus.Weight,
	mods ...memorydb.Mod,
) (*CoreLachesis, *consensusstore.Store, *consensustest.TestEventSource, *dagindexer.Index) {
	engine, store, eventSource, dagIndexer := NewCoreConsensus(nodes, weights)

	extended := &CoreLachesis{
		IndexedLachesis: engine,
		blocks:          map[BlockKey]*BlockResult{},
		epochBlocks:     map[consensus.Epoch]consensus.Frame{},
	}

	if err := extended.Bootstrap(consensus.ConsensusCallbacks{
		BeginBlock: func(block *consensus.Block) consensus.BlockCallbacks {
			return consensus.BlockCallbacks{
				EndBlock: func() (sealEpoch *consensus.Validators) {
					// track blocks
					key := BlockKey{
						Epoch: extended.store.GetEpoch(),
						Frame: extended.store.GetLastDecidedFrame() + 1,
					}
					extended.blocks[key] = &BlockResult{
						Atropos:    block.Atropos,
						Cheaters:   block.Cheaters,
						Validators: extended.store.GetValidators(),
					}
					// check that prev block exists
					if extended.lastBlock.Epoch != key.Epoch && key.Frame != 1 {
						panic("first frame must be 1")
					}
					extended.epochBlocks[key.Epoch]++
					extended.lastBlock = key
					if extended.applyBlock != nil {
						return extended.applyBlock(block)
					}
					return nil
				},
			}
		},
	}); err != nil {
		panic(err)
	}

	return extended, store, eventSource, dagIndexer
}

// NewCoreConsensus creates a simple consensus engine with mem store and optional node weights
func NewCoreConsensus(
	nodes []consensus.ValidatorID,
	weights []consensus.Weight,
) (*IndexedLachesis, *consensusstore.Store, *consensustest.TestEventSource, *dagindexer.Index) {
	validators := make(consensus.ValidatorsBuilder, len(nodes))
	for i, v := range nodes {
		if weights == nil {
			validators[v] = 1
		} else {
			validators[v] = weights[i]
		}
	}
	store := consensusstore.NewMemStore()

	err := store.ApplyGenesis(&consensusstore.Genesis{
		Validators: validators.Build(),
		Epoch:      consensus.FirstEpoch,
	})
	if err != nil {
		panic(err)
	}

	input := consensustest.NewTestEventSource()

	config := DefaultConfig()
	crit := func(err error) {
		panic(err)
	}
	dagIndexer := dagindexer.NewIndex(crit, dagindexer.LiteConfig())
	return NewIndexedLachesis(store, input, dagIndexer, crit, config), store, input, dagIndexer
}

func mutateValidators(validators *consensus.Validators) *consensus.Validators {
	r := consensustest.NewIntSeededRandGenerator(uint64(validators.TotalWeight()))
	builder := consensus.NewValidatorsBuilder()
	for _, vid := range validators.IDs() {
		stake := uint64(validators.Get(vid))*uint64(500+r.IntN(500))/1000 + 1
		builder.Set(vid, consensus.Weight(stake))
	}
	return builder.Build()
}
