// SPDX-FileCopyrightText: 2023 The Pion community <https://pion.ly>
// SPDX-License-Identifier: MIT

package cc

import (
	"fmt"
	"time"

	"github.com/pion/rtcp"
)

// only for stats, not used in cc algorithms
type AcknowledgmentStats struct {
	FrameID uint16 // if not video, set to 0, otherwise a sequence number start from 1
}

// Acknowledgment holds information about a packet and if/when it has been
// sent/received.
type Acknowledgment struct {
	SequenceNumber uint16 // Either RTP SequenceNumber or TWCC
	SSRC           uint32
	Size           int
	Departure      time.Time
	Arrival        time.Time
	ECN            rtcp.ECN

	// only for stats, not used in cc algorithms
	Stats AcknowledgmentStats
}

func (a Acknowledgment) String() string {
	s := "ACK:\n"
	s += fmt.Sprintf("\tTWCC:\t%v\n", a.SequenceNumber)
	s += fmt.Sprintf("\tSIZE:\t%v\n", a.Size)
	s += fmt.Sprintf("\tDEPARTURE:\t%v\n", int64(float64(a.Departure.UnixNano())/1e+6))
	s += fmt.Sprintf("\tARRIVAL:\t%v\n", int64(float64(a.Arrival.UnixNano())/1e+6))

	return s
}
