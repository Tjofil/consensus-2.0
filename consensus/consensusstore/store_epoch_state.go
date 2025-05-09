// Copyright (c) 2025 Fantom Foundation
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at fantom.foundation/bsl11.
//
// Change Date: 2028-4-16
//
// On the date above, in accordance with the Business Source License, use of
// this software will be governed by the GNU Lesser General Public License v3.

package consensusstore

import (
	"fmt"

	"github.com/0xsoniclabs/consensus/consensus"
)

const esKey = "e"

type EpochState struct {
	// stored values
	// these values change only after a change of epoch
	Epoch      consensus.Epoch
	Validators *consensus.Validators
}

func (es EpochState) String() string {
	return fmt.Sprintf("%d/%s", es.Epoch, es.Validators.String())
}

// SetEpochState stores epoch.
func (s *Store) SetEpochState(e *EpochState) {
	s.cache.EpochState = e
	s.setEpochState([]byte(esKey), e)
}

// GetEpochState returns stored epoch.
func (s *Store) GetEpochState() *EpochState {
	if s.cache.EpochState != nil {
		return s.cache.EpochState
	}
	e := s.getEpochState([]byte(esKey))
	if e == nil {
		s.crit(ErrNoGenesis)
	}
	s.cache.EpochState = e
	return e
}

func (s *Store) setEpochState(key []byte, e *EpochState) {
	s.set(s.table.EpochState, key, e)
}

func (s *Store) getEpochState(key []byte) *EpochState {
	w, exists := s.get(s.table.EpochState, key, &EpochState{}).(*EpochState)
	if !exists {
		return nil
	}
	return w
}

// GetEpoch returns current epoch
func (s *Store) GetEpoch() consensus.Epoch {
	return s.GetEpochState().Epoch
}

// GetValidators returns current validators
func (s *Store) GetValidators() *consensus.Validators {
	return s.GetEpochState().Validators
}
