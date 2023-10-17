package main

import (
	"bufio"
	"fmt"

	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
)

const (
	BroadcastProtocolID = "/broadcast/1.0.0"
)

type (
	// mapping from Sender->Recipient->ReadWriter
	ReadWriters map[peer.ID]*bufio.ReadWriter

	BroadcastService struct {
		s               *Server
		downReadWriters ReadWriters
		upReadWriters   ReadWriters
	}
)

func NewBroadcastService(s *Server) *BroadcastService {
	b := &BroadcastService{
		s:               s,
		downReadWriters: make(ReadWriters, 0),
		upReadWriters:   make(ReadWriters, 0),
	}
	b.s.host.SetStreamHandler(BroadcastProtocolID, b.OnBroadcast)
	return b
}

func (b *BroadcastService) ReadWriters() (*ReadWriters, *ReadWriters) {
	// prune RWs for peers that are no longer connected
	for peerID, rw := range b.upReadWriters {
		if b.s.host.Network().Connectedness(peerID) != network.Connected {
			fmt.Printf("Removing RW for %s\n", peerID)
			delete(b.upReadWriters, peerID)
			rw.Flush()
			rw.Reader.Reset(nil)
			rw.Writer.Reset(nil)
		}
	}

	// add RWs for peers that are connected but don't have one yet
	for _, peerID := range b.s.host.Peerstore().Peers() {
		if peerID == b.s.host.ID() {
			continue
		}
		if _, ok := b.upReadWriters[peerID]; !ok {
			rw, err := b.s.newReadWriter(peerID, BroadcastProtocolID)
			if err != nil {
				continue
			}
			b.upReadWriters[peerID] = rw
			go readData(rw)
		}
	}

	return &b.downReadWriters, &b.upReadWriters
}

func (b *BroadcastService) BroadcastMessage(msg []byte) error {
	inpRWs, outRWs := b.ReadWriters()

	for _, rw := range *inpRWs {
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
	for _, rw := range *outRWs {
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
	b.downReadWriters[s.Conn().RemotePeer()] = rw
	// go readData(rw)
}

func (b *BroadcastService) OnPeerFound(peerInfo peer.AddrInfo) {
	rw, err := b.s.newReadWriter(peerInfo.ID, BroadcastProtocolID)
	if err != nil {
		fmt.Printf("Error creating readwriter for %s: %s\n", peerInfo.ID, err)
		return
	}
	b.upReadWriters[peerInfo.ID] = rw
	go readData(rw)
}
