package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/libp2p/go-libp2p"
	"golang.org/x/sync/errgroup"
)

func main() {
	ctx := context.Background()
	ctx, cancel := signal.NotifyContext(ctx, os.Interrupt, os.Kill, syscall.SIGTERM)
	defer cancel()

	host, err := libp2p.New(libp2p.ListenAddrStrings("/ip4/127.0.0.1/tcp/0"))
	if err != nil {
		panic(err)
	}
	server := newServer(ctx, host)
	fmt.Printf("Host created. We are: %s\n", host.ID().Pretty())
	defer host.Close()

	bs := NewBroadcastService(server)
	bs.AddChannel(BroadcastProtocolID)

	discovery := NewDiscovery(server)
	discovery.RegisterHandler(bs.OnPeerFound)
	discovery.setupDiscovery()

	var errGroup errgroup.Group
	errGroup.Go(func() error {
		stdReader := bufio.NewReader(os.Stdin)
		for {
			input, err := stdReader.ReadString('\n')
			if err != nil {
				fmt.Println("Error reading from stdin")
			}
			for _, peerID := range host.Peerstore().Peers() {
				if peerID == host.ID() {
					continue
				}
			}
			fmt.Printf("broadcasting to peers: %s\n", strings.Trim(input, "\n"))
			go bs.BroadcastMessage(BroadcastProtocolID, []byte(input))
		}
	})
	errGroup.Go(func() error {
		<-ctx.Done()
		fmt.Printf("Shutting down...\n")
		if err = host.Close(); err != nil {
			fmt.Printf("Error closing host: %s\n", err)
			return err
		}
		return nil
	})
	if err = errGroup.Wait(); err != nil {
		fmt.Printf("Error in main: %s\n", err)
	}
}
