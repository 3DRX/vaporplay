package utils

import (
	"log/slog"

	"github.com/pion/rtp"
)

type FrameTypeData struct {
	FrameType FrameTypeEnum
	Start     bool
	FrameID   uint64
}

type FrameTypeEnum int

const (
	FrameTypeUnknown FrameTypeEnum = iota
	FrameTypeKeyFrame
	FrameTypeDeltaFrame
	FrameTypeSEI
)

func GetFrameTypeDataFromH264Packet(rtpPacket *rtp.Packet) FrameTypeData {
	var nalUnitType, S, h264NalUnitType byte
	frameTypeData := FrameTypeData{
		FrameType: FrameTypeUnknown,
	}
	nalUnitType = rtpPacket.Payload[0] & 0x1F
	if 1 <= nalUnitType && nalUnitType <= 23 {
		if nalUnitType == 6 {
			frameTypeData.FrameType = FrameTypeSEI
		} else {
			frameTypeData.FrameType = FrameTypeDeltaFrame
			frameTypeData.Start = true
		}
	} else if nalUnitType == 28 {
		S = rtpPacket.Payload[1] >> 7
		h264NalUnitType = rtpPacket.Payload[1] & 0x1F
		frameTypeData.Start = S == 1
		switch h264NalUnitType {
		case 5:
			frameTypeData.FrameType = FrameTypeKeyFrame
		default:
			frameTypeData.FrameType = FrameTypeDeltaFrame
		}
	} else if nalUnitType == 24 {
		// For STAP-A, we only read the first NAL unit header,
		// and assume the rest are the same as the first one.
		S = rtpPacket.Payload[3] >> 7
		h264NalUnitType = rtpPacket.Payload[3] & 0x1F
		frameTypeData.Start = S == 1
		switch h264NalUnitType {
		case 5:
			frameTypeData.FrameType = FrameTypeKeyFrame
		default:
			frameTypeData.FrameType = FrameTypeDeltaFrame
		}
	} else {
		slog.Warn("FrameTypeInterceptor nalUnitType not supported", "nalUnitType", nalUnitType)
	}
	return frameTypeData
}
