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
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
	"golang.org/x/sync/errgroup"
)

const (
	PingRequestProtocolID  = "/ping/req/1.0.0"
	PingResponseProtocolID = "/ping/rsp/1.0.0"
	DiscoveryNamespace     = "ping"
)

type (
	Handler func(ctx context.Context, msg ReadMessage, resp Writer) error

	PingService struct {
		s *Server
	}

	ReadMessage struct {
		From peer.ID
		Msg  []byte
	}

	ResponseWriter interface {
		Write(peer.ID, protocol.ID, []byte) error
	}

	PingMessage struct {
		Msg string
	}
)

func newPingService(s *Server) *PingService {
	p := &PingService{
		s: s,
	}
	p.RegisterHandler(PingRequestProtocolID, onPingRequest)
	p.RegisterHandler(PingResponseProtocolID, onPingResponse)
	return p
}

func (p *PingService) RegisterHandler(protocolID protocol.ID, handler Handler) {
	p.s.host.SetStreamHandler(protocolID, func(s network.Stream) {
		msgBytes, err := io.ReadAll(s)
		if err != nil {
			fmt.Printf("Error reading ping request: %s\n", err)
		}
		req := ReadMessage{
			From: s.Conn().RemotePeer(),
			Msg:  msgBytes,
		}
		err = handler(p.s.ctx, req, NewWriter(p.s, s.Conn().RemotePeer()))
		if err != nil {
			fmt.Printf("Error handling ping request: %s\n", err)
		}
	})
}

func (p *PingService) ping(peerID peer.ID, msg PingMessage) {
	req := PingMessage{Msg: msg.Msg}
	reqBytes, err := json.Marshal(req)
	w := NewWriter(p.s, peerID)
	err = w.Write(PingRequestProtocolID, reqBytes)
	if err != nil {
		fmt.Printf("Error sending ping request: %s\n", err)
	}
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
