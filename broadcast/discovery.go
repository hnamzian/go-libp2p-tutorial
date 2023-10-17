package main

import (
	"fmt"

	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/peerstore"
	"github.com/libp2p/go-libp2p/p2p/discovery/mdns"
)

const DiscoveryNamespace = "ping"

type discoveryHandler func(peerInfo peer.AddrInfo)

type discoveryNotifee struct {
	s        *Server
	handlers []discoveryHandler
}

func NewDiscovery(s *Server) *discoveryNotifee {
	return &discoveryNotifee{s: s, handlers: make([]discoveryHandler, 0)}
}

func (n *discoveryNotifee) RegisterHandler(handlers ...discoveryHandler) {
	n.handlers = append(n.handlers, handlers...)
}

func (n *discoveryNotifee) HandlePeerFound(peerInfo peer.AddrInfo) {
	fmt.Println("found peer", peerInfo.String())
	if err := n.s.host.Connect(n.s.ctx, peerInfo); err != nil {
		fmt.Println("error adding peer", err)
	}
	n.s.host.Peerstore().AddAddr(peerInfo.ID, peerInfo.Addrs[0], peerstore.PermanentAddrTTL)
	for _, handler := range n.handlers {
		handler(peerInfo)
	}
}

func (n *discoveryNotifee) setupDiscovery() {
	discoveryService := mdns.NewMdnsService(
		n.s.host,
		DiscoveryNamespace,
		n,
	)
	discoveryService.Start()
}
