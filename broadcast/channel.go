package main

import (
	"bufio"

	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
)

type BroadcastChannels map[protocol.ID]*BroadcastChannel

type BroadcastChannel struct {
	downReadWriters ReadWriters
	upReadWriters   ReadWriters
}

func (c BroadcastChannels) AddUpstream(peerID peer.ID, protocolID protocol.ID, rw *bufio.ReadWriter) {
	c[protocolID].upReadWriters[peerID] = rw
}
func (c BroadcastChannels) AddDownstream(peerID peer.ID, protocolID protocol.ID, rw *bufio.ReadWriter) {
	c[protocolID].downReadWriters[peerID] = rw
}
func (c BroadcastChannels) RemoveStream(peerID peer.ID, protocolID protocol.ID) {
	delete(c[protocolID].upReadWriters, peerID)
	delete(c[protocolID].downReadWriters, peerID)
}
