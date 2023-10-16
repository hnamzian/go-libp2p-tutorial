package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"

	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
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
