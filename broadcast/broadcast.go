package main

import (
	"bufio"
	"fmt"

	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
)

const (
	BroadcastProtocolID = "/broadcast/1.0.0"
)

type (
	ReadWriters map[peer.ID]*bufio.ReadWriter

	BroadcastService struct {
		s        *Server
		channels BroadcastChannels
	}
)

func NewBroadcastService(s *Server) *BroadcastService {
	b := &BroadcastService{
		s:        s,
		channels: make(BroadcastChannels, 0),
	}
	return b
}

func (b *BroadcastService) AddChannel(protocolID protocol.ID) {
	b.channels[protocolID] = &BroadcastChannel{
		downReadWriters: make(ReadWriters, 0),
		upReadWriters:   make(ReadWriters, 0),
	}
	b.s.host.SetStreamHandler(BroadcastProtocolID, b.OnBroadcast)
}

func (b *BroadcastService) ReadWriters(protocolID protocol.ID) (*BroadcastChannel) {
	// prune RWs for peers that are no longer connected
	for peerID, rw := range b.channels[protocolID].upReadWriters {
		if b.s.host.Network().Connectedness(peerID) != network.Connected {
			fmt.Printf("Removing RW for %s\n", peerID)
			b.channels.RemoveStream(peerID, protocolID)
			rw.Flush()
			rw.Reader.Reset(nil)
			rw.Writer.Reset(nil)
		}
	}
	return b.channels[protocolID]
}

func (b *BroadcastService) BroadcastMessage(protocolID protocol.ID, msg []byte) error {
	bc := b.ReadWriters(protocolID)

	for _, rw := range bc.downReadWriters {
		_, err := rw.Write(msg)
		if err != nil {
			fmt.Printf("Error writing to %s: %s\n", rw, err)
			return err
		}
		if err := rw.Flush(); err != nil {
			fmt.Printf("Error flushing %s: %s\n", rw, err)
			return err
		}
	}
	for _, rw := range bc.upReadWriters {
		_, err := rw.Write(msg)
		if err != nil {
			fmt.Printf("Error writing to %s: %s\n", rw, err)
			return err
		}
		if err := rw.Flush(); err != nil {
			fmt.Printf("Error flushing: %s\n", err)
			return err
		}
	}

	return nil
}

func readData(rw *bufio.ReadWriter) {
	for {
		str, _ := rw.ReadString('\n')

		if str == "" {
			return
		}
		if str != "\n" {
			fmt.Printf("\x1b[32m%s\x1b[0m", str)
		}
	}
}

func (b *BroadcastService) OnBroadcast(s network.Stream) {
	fmt.Printf("Received broadcast from %s\n", s.Conn().RemotePeer().String())
	rw := bufio.NewReadWriter(bufio.NewReader(s), bufio.NewWriter(s))
	b.channels.AddDownstream(s.Conn().RemotePeer(), s.Protocol(), rw)
}

func (b *BroadcastService) OnPeerFound(peerInfo peer.AddrInfo) {
	rw, err := b.s.newReadWriter(peerInfo.ID, BroadcastProtocolID)
	if err != nil {
		fmt.Printf("Error creating readwriter for %s: %s\n", peerInfo.ID, err)
		return
	}
	b.channels.AddUpstream(peerInfo.ID, BroadcastProtocolID, rw)
	go readData(rw)
}
