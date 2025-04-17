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
	"math"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/0xsoniclabs/consensus/consensus"
	"github.com/0xsoniclabs/consensus/consensus/consensustest"
)

func TestConfirmBlocks_1(t *testing.T) {
	testConfirmBlocks(t, []consensus.Weight{1}, 0)
}

func TestConfirmBlocks_big1(t *testing.T) {
	testConfirmBlocks(t, []consensus.Weight{math.MaxUint32 / 2}, 0)
}

func TestConfirmBlocks_big2(t *testing.T) {
	testConfirmBlocks(t, []consensus.Weight{math.MaxUint32 / 4, math.MaxUint32 / 4}, 0)
}

func TestConfirmBlocks_big3(t *testing.T) {
	testConfirmBlocks(t, []consensus.Weight{math.MaxUint32 / 8, math.MaxUint32 / 8, math.MaxUint32 / 4}, 0)
}

func TestConfirmBlocks_4(t *testing.T) {
	testConfirmBlocks(t, []consensus.Weight{1, 2, 3, 4}, 0)
}

func TestConfirmBlocks_3_1(t *testing.T) {
	testConfirmBlocks(t, []consensus.Weight{1, 1, 1, 1}, 1)
}

func TestConfirmBlocks_67_33(t *testing.T) {
	testConfirmBlocks(t, []consensus.Weight{33, 67}, 1)
}

func TestConfirmBlocks_67_33_4(t *testing.T) {
	testConfirmBlocks(t, []consensus.Weight{11, 11, 11, 67}, 3)
}

func TestConfirmBlocks_67_33_5(t *testing.T) {
	testConfirmBlocks(t, []consensus.Weight{11, 11, 11, 33, 34}, 3)
}

func TestConfirmBlocks_2_8_10(t *testing.T) {
	testConfirmBlocks(t, []consensus.Weight{1, 2, 1, 2, 1, 2, 1, 2, 1, 2}, 3)
}

func testConfirmBlocks(t *testing.T, weights []consensus.Weight, cheatersCount int) {
	t.Helper()
	assertar := assert.New(t)

	nodes := consensustest.GenNodes(len(weights))
	lch, _, input, _ := NewBootstrappedCoreConsensus(nodes, weights)

	var (
		frames []consensus.Frame
		blocks []*consensus.Block
	)
	lch.applyBlock = func(block *consensus.Block) *consensus.Validators {
		frames = append(frames, lch.store.GetLastDecidedFrame()+1)
		blocks = append(blocks, block)

		return nil
	}

	eventCount := int(TestMaxEpochEvents)
	parentCount := 5
	if parentCount > len(nodes) {
		parentCount = len(nodes)
	}
	r := consensustest.NewIntSeededRandGenerator(uint64(len(nodes) + cheatersCount))
	consensustest.ForEachRandFork(nodes, nodes[:cheatersCount], eventCount, parentCount, 10, r, consensustest.ForEachEvent{
		Process: func(e consensus.Event, name string) {
			input.SetEvent(e)
			assertar.NoError(
				lch.Process(e))

		},
		Build: func(e consensus.MutableEvent, name string) error {
			e.SetEpoch(consensus.FirstEpoch)
			return lch.Build(e)
		},
	})

	// unconfirm all events
	it := lch.store.EpochTable.ConfirmedEvent.NewIterator(nil, nil)
	batch := lch.store.EpochTable.ConfirmedEvent.NewBatch()
	for it.Next() {
		assertar.NoError(batch.Delete(it.Key()))
	}
	assertar.NoError(batch.Write())
	it.Release()

	for i, block := range blocks {
		frame := frames[i]
		atropos := blocks[i].Atropos

		// call confirmBlock again
		_, err := lch.onFrameDecided(frame, atropos)
		gotBlock := lch.blocks[lch.lastBlock]

		if !assertar.NoError(err) {
			break
		}
		if !assertar.LessOrEqual(len(gotBlock.Cheaters), cheatersCount) {
			break
		}
		if !assertar.Equal(block.Cheaters, gotBlock.Cheaters) {
			break
		}
		if !assertar.Equal(block.Atropos, gotBlock.Atropos) {
			break
		}
	}
	assertar.GreaterOrEqual(len(blocks), TestMaxEpochEvents/5)
}
