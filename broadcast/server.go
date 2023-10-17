package main

import (
	"bufio"
	"context"
	"fmt"

	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
)

type Server struct {
	host host.Host
	ctx  context.Context
}

func newServer(ctx context.Context, host host.Host) *Server {
	s := &Server{
		host: host,
		ctx:  ctx,
	}
	return s
}

func (s *Server) Write(peerID peer.ID, protocolID protocol.ID, data []byte) error {
	stream, err := s.host.NewStream(s.ctx, peerID, protocolID)
	if err != nil {
		fmt.Printf("Error opening stream: %s\n", err)
		return err
	}
	_, err = stream.Write(data)
	if err != nil {
		err = stream.Reset()
		if err != nil {
			fmt.Printf("Error resetting stream: %s\n", err)
			return err
		}
	}
	err = stream.Close()
	if err != nil {
		err = stream.Reset()
		if err != nil {
			fmt.Printf("Error resetting stream: %s\n", err)
			return err
		}
	}
	return nil
}

func (s *Server) newReadWriter(peerID peer.ID, protocolID protocol.ID) (*bufio.ReadWriter, error) {
	stream, err := s.host.NewStream(s.ctx, peerID, protocolID)
	if err != nil {
		fmt.Printf("Error opening stream: %s\n", err)
		return nil, err
	}
	rw := bufio.NewReadWriter(
		bufio.NewReader(stream),
		bufio.NewWriter(stream),
	)
	return rw, nil
}
