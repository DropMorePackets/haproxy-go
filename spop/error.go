package spop

import (
	"fmt"
)

type errorCode int

func (e errorCode) String() string {
	switch e {
	case ErrorNone:
		return "normal"
	case ErrorIO:
		return "I/O error"
	case ErrorTimeout:
		return "a timeout occurred"
	case ErrorTooBig:
		return "frame is too big"
	case ErrorInvalid:
		return "invalid frame received"
	case ErrorNoVSN:
		return "version value not found"
	case ErrorNoFrameSize:
		return "max-frame-size value not found"
	case ErrorNoCap:
		return "capabilities value not found"
	case ErrorBadVsn:
		return "unsupported version"
	case ErrorBadFrameSize:
		return "max-frame-size too big or too small"
	case ErrorFragNotSupported:
		return "fragmentation not supported"
	case ErrorInterlacedFrames:
		return "invalid interlaced frames"
	case ErrorFrameIDNotfound:
		return "frame-id not found"
	case ErrorRes:
		return "resource allocation error"
	case ErrorUnknown:
		return "an unknown error occurred"
	default:
		return fmt.Sprintf("unknown spoe error code: %d", e)
	}
}

const (
	ErrorNone errorCode = iota
	ErrorIO
	ErrorTimeout
	ErrorTooBig
	ErrorInvalid
	ErrorNoVSN
	ErrorNoFrameSize
	ErrorNoCap
	ErrorBadVsn
	ErrorBadFrameSize
	ErrorFragNotSupported
	ErrorInterlacedFrames
	ErrorFrameIDNotfound
	ErrorRes
	ErrorUnknown errorCode = 99
)
