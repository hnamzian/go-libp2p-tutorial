package main

import (
	"bufio"
	"context"
	"encoding/binary"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/libp2p/go-libp2p"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/p2p/discovery/mdns"
)

const (
	protocolID         = "/example/1.0.0"
	discoveryNamespace = "example"
	gossipRoom         = "librum"
)

func main() {
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

	gossipSub, err := pubsub.NewGossipSub(ctx, host)
	if err != nil {
		panic(err)
	}
	topic, err := gossipSub.Join(gossipRoom)
	if err != nil {
		panic(err)
	}
	subscriber, err := topic.Subscribe()
	if err != nil {
		panic(err)
	}

	// Setup peer discovery.
	setupDiscovery(ctx, host)

	go subscribe(ctx, subscriber, host.ID())
	go publish(ctx, topic)

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

func publish(ctx context.Context, topic *pubsub.Topic) {
	for {
		scanner := bufio.NewScanner(os.Stdin)

		for scanner.Scan() {
			fmt.Print("Enter a message to publish: ")
			msg := scanner.Text()

			if len(msg) != 0 {
				topic.Publish(ctx, []byte(msg))
			}
		}
	}
}

func subscribe(ctx context.Context, subscriber *pubsub.Subscription, hostID peer.ID) {
	for {
		msg, err := subscriber.Next(ctx)
		if err != nil {
			panic(err)
		}

		if msg.ReceivedFrom == hostID {
			continue
		}

		fmt.Printf("Received: %s from %s\n", string(msg.Data), msg.GetFrom().String())
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
