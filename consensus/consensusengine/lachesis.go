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
	"github.com/0xsoniclabs/consensus/consensus"
	"github.com/0xsoniclabs/consensus/consensus/consensusstore"
	"github.com/0xsoniclabs/consensus/dagindexer"
)

var _ consensus.Consensus = (*Lachesis)(nil)

// Lachesis performs events ordering and detects cheaters
// It's a wrapper around Orderer, which adds features which might potentially be application-specific:
// confirmed events traversal, cheaters detection.
// Use this structure if need a general-purpose consensus. Instead, use lower-level abft.Orderer.
type Lachesis struct {
	*Orderer
	dagIndex *dagindexer.Index
	callback consensus.ConsensusCallbacks
}

// NewLachesis creates Lachesis instance.
func NewLachesis(store *consensusstore.Store, input EventSource, dagIndex *dagindexer.Index, crit func(error), config Config) *Lachesis {
	p := &Lachesis{
		Orderer:  NewOrderer(store, input, dagIndex, crit, config),
		dagIndex: dagIndex,
	}

	return p
}

func (p *Lachesis) confirmEvents(frame consensus.Frame, atropos consensus.EventHash, onEventConfirmed func(consensus.Event)) error {
	err := p.dfsSubgraph(atropos, func(e consensus.Event) bool {
		decidedFrame := p.store.GetEventConfirmedOn(e.ID())
		if decidedFrame != 0 {
			return false
		}
		// mark all the walked events as confirmed
		p.store.SetEventConfirmedOn(e.ID(), frame)
		if onEventConfirmed != nil {
			onEventConfirmed(e)
		}
		return true
	})
	return err
}

func (p *Lachesis) applyAtropos(decidedFrame consensus.Frame, atropos consensus.EventHash) *consensus.Validators {
	atroposVecClock := p.dagIndex.GetMergedHighestBefore(atropos).VSeq

	validators := p.store.GetValidators()
	// cheaters are ordered deterministically
	cheaters := make([]consensus.ValidatorID, 0, validators.Len())
	for creatorIdx, creator := range validators.SortedIDs() {
		if atroposVecClock.Get(consensus.ValidatorIndex(creatorIdx)).IsForkDetected() {
			cheaters = append(cheaters, creator)
		}
	}

	if p.callback.BeginBlock == nil {
		return nil
	}
	blockCallback := p.callback.BeginBlock(&consensus.Block{
		Atropos:  atropos,
		Cheaters: cheaters,
	})

	// traverse newly confirmed events
	err := p.confirmEvents(decidedFrame, atropos, blockCallback.ApplyEvent)
	if err != nil {
		p.crit(err)
	}

	if blockCallback.EndBlock != nil {
		return blockCallback.EndBlock()
	}
	return nil
}

func (p *Lachesis) Bootstrap(callback consensus.ConsensusCallbacks) error {
	return p.BootstrapWithOrderer(callback, p.OrdererCallbacks())
}

func (p *Lachesis) BootstrapWithOrderer(callback consensus.ConsensusCallbacks, ordererCallbacks OrdererCallbacks) error {
	err := p.Orderer.Bootstrap(ordererCallbacks)
	if err != nil {
		return err
	}
	p.callback = callback
	return nil
}

func (p *Lachesis) OrdererCallbacks() OrdererCallbacks {
	return OrdererCallbacks{
		ApplyAtropos: p.applyAtropos,
	}
}
