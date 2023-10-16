package main

import (
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
)

type Writer interface {
	Write([]byte) error
}

type writer struct {
	s          *Server
	PeerID     peer.ID
	ProtocolID protocol.ID
}

func NewWriter(s *Server, pid peer.ID, protocolID protocol.ID) *writer {
	return &writer{
		s:          s,
		PeerID:     pid,
		ProtocolID: protocolID,
	}
}

func (w *writer) Write(msg []byte) error {
	return w.s.Write(w.PeerID, w.ProtocolID, msg)
}
