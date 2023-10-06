package main

import (
	"context"
	"encoding/binary"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/p2p/discovery/mdns"
	"github.com/multiformats/go-multiaddr"
)

const protocolID = "/example/1.0.0"
const discoveryNamespace = "example"

func main() {
	peerAddr := flag.String("peer-address", "", "peer address")
	flag.Parse()

	ctx := context.Background()

	host, err := libp2p.New(libp2p.ListenAddrStrings("/ip4/127.0.0.1/tcp/0"))
	if err != nil {
		panic(err)
	}
	defer host.Close()

	hostPeerInfo := &peer.AddrInfo{
		ID:    host.ID(),
		Addrs: host.Addrs(),
	}
	hostAddrs, err := peer.AddrInfoToP2pAddrs(hostPeerInfo)
	if err != nil {
		panic(err)
	}
	fmt.Println("hostAddrs: ", hostAddrs[0])

	host.SetStreamHandler(protocolID, func(s network.Stream) {
		go writeCounter(s)
		go readCounter(s)
	})

	// Setup peer discovery.
	setupDiscovery(ctx, host)

	if *peerAddr != "" {
		// Parse the multiaddr string.
		peerMA, err := multiaddr.NewMultiaddr(*peerAddr)
		if err != nil {
			panic(err)
		}
		peerAddrInfo, err := peer.AddrInfoFromP2pAddr(peerMA)
		if err != nil {
			panic(err)
		}

		// Connect to the node at the given address.
		if err := host.Connect(ctx, *peerAddrInfo); err != nil {
			panic(err)
		}
		fmt.Println("Connected to", peerAddrInfo.String())

		// Open a stream with the given peer.
		s, err := host.NewStream(ctx, peerAddrInfo.ID, protocolID)
		if err != nil {
			panic(err)
		}

		// Start the write and read threads.
		go writeCounter(s)
		go readCounter(s)
	}

	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGTERM, syscall.SIGINT)
	<-ch
	fmt.Printf("Received signal, shutting down...\n")
	if err = host.Close(); err != nil {
		panic(err)
	}
}

func writeCounter(s network.Stream) {
	var counter uint64

	for {
		<-time.After(time.Second)
		counter++

		fmt.Printf("Sending %d to %s\n", counter, s.ID())

		err := binary.Write(s, binary.BigEndian, counter)
		if err != nil {
			panic(err)
		}
	}
}

func readCounter(s network.Stream) {
	for {
		var counter uint64

		err := binary.Read(s, binary.BigEndian, &counter)
		if err != nil {
			panic(err)
		}

		fmt.Printf("Received %d from %s\n", counter, s.ID())
	}
}

type discoveryNotifee struct {
	ctx context.Context
	h   host.Host
}

func (n *discoveryNotifee) HandlePeerFound(peerInfo peer.AddrInfo) {
	fmt.Println("found peer", peerInfo.String())
	if err := n.h.Connect(n.ctx, peerInfo); err != nil {
		fmt.Println("error adding peer", err)
	}
}

func setupDiscovery(ctx context.Context, host host.Host) {
	discoveryService := mdns.NewMdnsService(
		host,
		discoveryNamespace,
		&discoveryNotifee{ctx: ctx, h: host},
	)
	discoveryService.Start()
}
