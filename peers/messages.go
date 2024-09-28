package peers

import (
	"bufio"
	"fmt"
	"io"
	"os"
)

// Handshake is composed by these fields:
//
//	protocol identifier   : HAProxyS
//	version               : 2.1
//	remote peer identifier: the peer name this "hello" message is sent to.
//	local peer identifier : the name of the peer which sends this "hello" message.
//	process ID            : the ID of the process handling this peer session.
//	relative process ID   : the haproxy's relative process ID (0 if nbproc == 1).
type Handshake struct {
	ProtocolIdentifier  string
	Version             string
	RemotePeer          string
	LocalPeerIdentifier string
	ProcessID           int
	RelativeProcessID   int
}

// NewHandshake returns a basic handshake to be used for connecting to
// haproxy peers. It is filled with all necessary information except the remote
// peer hostname.
func NewHandshake(remotePeer string) *Handshake {
	return &Handshake{
		ProtocolIdentifier: "HAProxyS",
		Version:            "2.1",
		RemotePeer:         remotePeer,
		LocalPeerIdentifier: func() string {
			s, _ := os.Hostname()
			return s
		}(),
		ProcessID:         os.Getpid(),
		RelativeProcessID: 0,
	}
}

func (h *Handshake) ReadFrom(r io.Reader) (n int64, err error) {
	scanner := bufio.NewScanner(r)

	scanner.Scan()
	_, err = fmt.Sscanf(scanner.Text(), "%s %s", &h.ProtocolIdentifier, &h.Version)
	if err != nil {
		return -1, err
	}

	scanner.Scan()
	h.RemotePeer = scanner.Text()

	scanner.Scan()
	_, err = fmt.Sscanf(scanner.Text(), "%s %d %d", &h.LocalPeerIdentifier, &h.ProcessID, &h.RelativeProcessID)
	if err != nil {
		return -1, err
	}

	//TODO: find out how many bytes where read.
	return -1, scanner.Err()
}

func (h *Handshake) WriteTo(w io.Writer) (nw int64, err error) {
	n, err := fmt.Fprintf(w, "%s %s\n", h.ProtocolIdentifier, h.Version)
	nw += int64(n)
	if err != nil {
		return nw, err
	}

	n, err = fmt.Fprintf(w, "%s\n", h.RemotePeer)
	nw += int64(n)
	if err != nil {
		return nw, err
	}

	n, err = fmt.Fprintf(w, "%s %d %d\n", h.LocalPeerIdentifier, h.ProcessID, h.RelativeProcessID)
	nw += int64(n)

	return nw, err
}
