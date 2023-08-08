package spop

import (
	"fmt"
)

type spopErrorCode int

func (s spopErrorCode) String() string {
	switch s {
	case spoeErrorNone:
		return "normal"
	case spoeErrorIO:
		return "I/O error"
	case spoeErrorTimeout:
		return "a timeout occurred"
	case spoeErrorTooBig:
		return "frame is too big"
	case spoeErrorInvalid:
		return "invalid frame received"
	case spoeErrorNoVSN:
		return "version value not found"
	case spoeErrorNoFrameSize:
		return "max-frame-size value not found"
	case spoeErrorNoCap:
		return "capabilities value not found"
	case spoeErrorBadVsn:
		return "unsupported version"
	case spoeErrorBadFrameSize:
		return "max-frame-size too big or too small"
	case spoeErrorFragNotSupported:
		return "fragmentation not supported"
	case spoeErrorInterlacedFrames:
		return "invalid interlaced frames"
	case spoeErrorFrameIDNotfound:
		return "frame-id not found"
	case spoeErrorRes:
		return "resource allocation error"
	case spoeErrorUnknown:
		return "an unknown error occurred"
	default:
		return fmt.Sprintf("unknown spoe error code: %d", s)
	}
}

const (
	spoeErrorNone spopErrorCode = iota
	spoeErrorIO
	spoeErrorTimeout
	spoeErrorTooBig
	spoeErrorInvalid
	spoeErrorNoVSN
	spoeErrorNoFrameSize
	spoeErrorNoCap
	spoeErrorBadVsn
	spoeErrorBadFrameSize
	spoeErrorFragNotSupported
	spoeErrorInterlacedFrames
	spoeErrorFrameIDNotfound
	spoeErrorRes
	spoeErrorUnknown spopErrorCode = 99
)
