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
	"fmt"
)

type Metric struct {
	Num  uint32
	Size uint64
}

func (m Metric) String() string {
	return fmt.Sprintf("{Num=%d,Size=%d}", m.Num, m.Size)
}
