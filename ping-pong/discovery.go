package main

import (
	"fmt"

	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/peerstore"
	"github.com/libp2p/go-libp2p/p2p/discovery/mdns"
)

const DiscoveryNamespace = "ping"

type discoveryNotifee struct {
	s *Server
}

func (n *discoveryNotifee) HandlePeerFound(peerInfo peer.AddrInfo) {
	fmt.Println("found peer", peerInfo.String())
	if err := n.s.host.Connect(n.s.ctx, peerInfo); err != nil {
		fmt.Println("error adding peer", err)
	}
	n.s.host.Peerstore().AddAddr(peerInfo.ID, peerInfo.Addrs[0], peerstore.PermanentAddrTTL)
}

func setupDiscovery(s *Server) {
	discoveryService := mdns.NewMdnsService(
		s.host,
		DiscoveryNamespace,
		&discoveryNotifee{s: s},
	)
	discoveryService.Start()
}
