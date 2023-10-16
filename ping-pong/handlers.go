package main

import (
	"context"
	"encoding/json"
	"fmt"
)

func onPingRequest(ctx context.Context, req ReadMessage, resp Writer) error {
	var msg PingMessage
	if err := json.Unmarshal([]byte(req.Msg), &msg); err != nil {
		fmt.Printf("Error unmarshaling ping message: %s\n", err)
	}
	fmt.Printf("-> %s: %s\n", req.From, msg.Msg)
	rspMsg := PingMessage{Msg: "pong"}
	rspMsgBytes, err := json.Marshal(rspMsg)
	if err != nil {
		fmt.Printf("Error marshaling ping response: %s\n", err)
		return err
	}
	return resp.Write(rspMsgBytes)
}

func onPingResponse(ctx context.Context, req ReadMessage, resp Writer) error {
	var msg PingMessage
	if err := json.Unmarshal([]byte(req.Msg), &msg); err != nil {
		fmt.Printf("Error unmarshaling ping message: %s\n", err)
	}
	fmt.Printf("-> %s: %s\n", req.From, msg.Msg)
	return nil
}
