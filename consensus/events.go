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
	"strings"
)

// Events is a ordered slice of events.
type Events []Event

// String returns human readable representation.
func (ee Events) String() string {
	ss := make([]string, len(ee))
	for i := 0; i < len(ee); i++ {
		ss[i] = ee[i].String()
	}
	return strings.Join(ss, " ")
}

func (ee Events) Metric() (metric Metric) {
	metric.Num = uint32(len(ee))
	for _, e := range ee {
		metric.Size += uint64(e.Size())
	}
	return metric
}

func (ee Events) IDs() EventHashes {
	ids := make(EventHashes, len(ee))
	for i, e := range ee {
		ids[i] = e.ID()
	}
	return ids
}
