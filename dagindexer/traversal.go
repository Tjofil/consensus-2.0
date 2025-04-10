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
	"errors"

	"github.com/0xsoniclabs/consensus/consensus"
)

// DfsSubgraph iterates all the event which are observed by head, and accepted by a filter
// Excluding head
// filter MAY BE called twice for the same event.
func (vi *Index) DfsSubgraph(head consensus.Event, walk func(consensus.EventHash) (godeeper bool)) error {
	stack := make(consensus.EventHashStack, 0, vi.validators.Len()*5)

	// first element
	stack.PushAll(head.Parents())

	for next := stack.Pop(); next != nil; next = stack.Pop() {
		curr := *next

		// filter
		if !walk(curr) {
			continue
		}

		event := vi.getEvent(curr)
		if event == nil {
			return errors.New("event not found " + curr.String())
		}

		// memorize parents
		stack.PushAll(event.Parents())
	}

	return nil
}
