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
	"github.com/pkg/errors"

	"github.com/0xsoniclabs/consensus/consensus"
)

var (
	ErrWrongFrame = errors.New("claimed frame mismatched with calculated")
)

// Build fills consensus-related fields: Frame, IsRoot
// returns error if event should be dropped
func (p *Orderer) Build(e consensus.MutableEvent) error {
	// sanity check
	if e.Epoch() != p.store.GetEpoch() {
		p.crit(errors.New("event has wrong epoch"))
	}
	if !p.store.GetValidators().Exists(e.Creator()) {
		p.crit(errors.New("event wasn't created by an existing validator"))
	}

	_, frame := p.calcFrameIdx(e)
	e.SetFrame(frame)

	return nil
}

// Process takes event into processing.
// Event order matter: parents first.
// All the event checkers must be launched.
// Process is not safe for concurrent use.
func (p *Orderer) Process(e consensus.Event) (err error) {
	selfParentFrame, err := p.checkAndSaveEvent(e)
	if err != nil {
		return err
	}

	if selfParentFrame == e.Frame() {
		return nil
	}
	if _, err := p.runElectionOnRoot(e.Frame(), e.Creator(), e.ID()); err != nil {
		// election doesn't fail under normal circumstances
		// storage is in an inconsistent state
		p.crit(err)
	}
	return err
}

// checkAndSaveEvent checks consensus-related fields: Frame, IsRoot
func (p *Orderer) checkAndSaveEvent(e consensus.Event) (consensus.Frame, error) {
	// check frame & isRoot
	selfParentFrame, frameIdx := p.calcFrameIdx(e)
	if !p.config.SuppressFramePanic && e.Frame() != frameIdx {
		return 0, ErrWrongFrame
	}

	if selfParentFrame != frameIdx {
		p.store.AddRoot(e)
	}
	return selfParentFrame, nil
}

// runElectionOnRoot runs Atropos election for the root and triggers block closure callbacks if election was decided
func (p *Orderer) runElectionOnRoot(frame consensus.Frame, validatorID consensus.ValidatorID, rootHash consensus.EventHash) (bool, error) {
	decisions, err := p.election.VoteAndAggregate(frame, validatorID, rootHash)
	if err != nil {
		return false, err
	}
	for _, atroposDecision := range decisions {
		sealed, err := p.onFrameDecided(atroposDecision.Frame, atroposDecision.AtroposHash)
		if err != nil {
			return false, err
		}
		if sealed {
			return true, nil
		}
	}
	return false, nil
}

func (p *Orderer) bootstrapElection() error {
	for frame := p.store.GetLastDecidedFrame() + 1; ; frame++ {
		frameRoots := p.store.GetFrameRoots(frame)
		if len(frameRoots) == 0 {
			break
		}
		for _, root := range frameRoots {
			sealed, err := p.runElectionOnRoot(frame, root.ValidatorID, root.RootHash)
			if err != nil {
				return err
			}
			if sealed {
				return nil
			}
		}
	}
	return nil
}

// forklessCausedByQuorumOn returns true if event is forkless caused by 2/3W roots on specified frame
func (p *Orderer) forklessCausedByQuorumOn(e consensus.Event, f consensus.Frame) bool {
	observedCounter := p.store.GetValidators().NewCounter()
	// check "observing" prev roots only if called by creator, or if creator has marked that event as root
	for _, it := range p.store.GetFrameRoots(f) {
		if p.dagIndex.ForklessCause(e.ID(), it.RootHash) {
			observedCounter.CountVoteByID(it.ValidatorID)
		}
		if observedCounter.HasQuorum() {
			break
		}
	}
	return observedCounter.HasQuorum()
}

// calcFrameIdx is not safe for concurrent use.
func (p *Orderer) calcFrameIdx(e consensus.Event) (selfParentFrame, frame consensus.Frame) {
	if e.SelfParent() == nil {
		return 0, 1
	}
	selfParentFrame = p.Input.GetEvent(*e.SelfParent()).Frame()
	frame = selfParentFrame
	for _, parent := range e.Parents() {
		frame = max(frame, p.Input.GetEvent(parent).Frame())
	}

	if p.forklessCausedByQuorumOn(e, frame) {
		frame++
	}
	return selfParentFrame, frame
}

func (p *Orderer) getSelfParentFrame(e consensus.Event) consensus.Frame {
	if e.SelfParent() == nil {
		return 0
	}
	return p.Input.GetEvent(*e.SelfParent()).Frame()
}
