package main

import (
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
)

type Writer interface {
	Write(protocol.ID, []byte) error
}

type writer struct {
	s          *Server
	PeerID     peer.ID
}

func NewWriter(s *Server, pid peer.ID) *writer {
	return &writer{
		s:          s,
		PeerID:     pid,
	}
}

func (w *writer) Write(protocolID protocol.ID, msg []byte) error {
	return w.s.Write(w.PeerID, protocolID, msg)
}
