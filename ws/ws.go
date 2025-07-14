package ws

import (
	"context"
	"net"
	"net/http"

	"github.com/b00tkitism/wsc/proxy"
)

type WSHandler struct {
	Proxy *proxy.Proxy
}

func NewWSHandler(proxyInstance *proxy.Proxy) *WSHandler {
	return &WSHandler{
		Proxy: proxyInstance,
	}
}

func (h *WSHandler) Start(address string) error {
	return nil
}

func (h *WSHandler) handleWS(w http.ResponseWriter, r *http.Request) {
	return
}

func (h *WSHandler) pipeWSToTCP(ctx context.Context, wsConn net.Conn, tcpConn net.Conn, user *proxy.User) error {
	for {

	}
}

func (h *WSHandler) pipeTCPToWS(ctx context.Context, tcpConn net.Conn, wsConn net.Conn, user *proxy.User) error {
	for {
	}
}
