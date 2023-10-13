package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/peerstore"
	"github.com/libp2p/go-libp2p/core/protocol"
	"github.com/libp2p/go-libp2p/p2p/discovery/mdns"
	"golang.org/x/sync/errgroup"
)

const (
	PingRequestProtocolID  = "/ping/req/1.0.0"
	PingResponseProtocolID = "/ping/rsp/1.0.0"
	DiscoveryNamespace     = "ping"
)

type (
	Server struct {
		host host.Host
		ctx  context.Context
	}

	PingService struct {
		s *Server
	}

	PingMessage struct {
		Msg string
	}

	discoveryNotifee struct {
		s *Server
	}
)

func newServer(ctx context.Context, host host.Host) *Server {
	s := &Server{
		host: host,
		ctx:  ctx,
	}
	return s
}

func newPingService(s *Server) *PingService {
	p := &PingService{
		s: s,
	}
	p.s.host.SetStreamHandler(PingRequestProtocolID, p.onPingRequest)
	p.s.host.SetStreamHandler(PingResponseProtocolID, p.onPingResponse)
	return p
}

func (p *PingService) onPingRequest(s network.Stream) {
	msgBytes, err := io.ReadAll(s)
	if err != nil {
		fmt.Printf("Error reading ping request: %s\n", err)
	}
	var msg PingMessage
	if err := json.Unmarshal(msgBytes, &msg); err != nil {
		fmt.Printf("Error unmarshaling ping request: %s\n", err)
	}
	fmt.Printf("-> %s: %s\n", s.Conn().RemotePeer().String(), msg.Msg)
	err = p.sendMessage(p.s.ctx, s.Conn().RemotePeer(), PingResponseProtocolID, PingMessage{Msg: "pong"})
	if err != nil {
		fmt.Printf("Error sending ping response: %s\n", err)
	}
}

func (p *PingService) onPingResponse(s network.Stream) {
	msgBytes, err := io.ReadAll(s)
	if err != nil {
		fmt.Printf("Error reading ping response: %s\n", err)
	}
	var msg PingMessage
	if err := json.Unmarshal(msgBytes, &msg); err != nil {
		fmt.Printf("Error unmarshaling ping response: %s\n", err)
	}
	fmt.Printf("<- %s: %s\n", s.Conn().RemotePeer().String(), msg.Msg)
}

func (p *PingService) ping(peerID peer.ID, msg PingMessage) {
	err := p.sendMessage(p.s.ctx, peerID, PingRequestProtocolID, msg)
	if err != nil {
		fmt.Printf("Error sending ping request: %s\n", err)
	}
}

func (p *PingService) sendMessage(ctx context.Context, peerID peer.ID, protocolID protocol.ID, msg PingMessage) error {
	s, err := p.s.host.NewStream(ctx, peerID, protocolID)
	if err != nil {
		fmt.Printf("Error opening stream: %s\n", err)
		return err
	}
	msgBytes, err := json.Marshal(msg)
	_, err = s.Write(msgBytes)
	if err != nil {
		err = s.Reset()
		if err != nil {
			fmt.Printf("Error resetting stream: %s\n", err)
			return err
		}
	}
	err = s.Close()
	if err != nil {
		err = s.Reset()
		if err != nil {
			fmt.Printf("Error resetting stream: %s\n", err)
			return err
		}
	}
	return nil
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

	setupDiscovery(server)

	var errGroup errgroup.Group
	errGroup.Go(func() error {
		p := newPingService(server)
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
				p.ping(peerID, PingMessage{Msg: strings.Trim(input, "\n")})
			}
		}
	})
	errGroup.Go(func() error {
		<-ctx.Done()
		fmt.Printf("Shutting down...\n")
		if err = host.Close(); err != nil {
			fmt.Printf("Error closing host: %s\n", err)
		}
		return nil
	})
	if err = errGroup.Wait(); err != nil {
		fmt.Printf("Error in main: %s\n", err)
	}
}
