package main

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strconv"
	"time"

	socks "github.com/firefart/gosocks"
	"github.com/gobwas/ws"
	"github.com/gobwas/ws/wsutil"
	"github.com/itsabgr/ge"
)

var _ socks.ProxyHandler = &CustomHandler{}

type CustomHandler struct {
	Auth string
	Host string
	Path string
}

func (c CustomHandler) socksErr(err error) *socks.Error {
	return &socks.Error{
		Err:    err,
		Reason: socks.RequestReplyGeneralFailure,
	}
}

func (c CustomHandler) Init(ctx context.Context, request socks.Request) (context.Context, io.ReadWriteCloser, *socks.Error) {
	pURL := url.URL{
		Scheme:   "ws",
		Host:     c.Host,
		Path:     c.Path,
		RawQuery: "",
	}
	pQuery := pURL.Query()
	pQuery.Set("auth", c.Auth)
	pQuery.Set("ep", request.GetDestinationString())
	pURL.RawQuery = pQuery.Encode()

	conn, _, _, err := ws.Dial(ctx, pURL.String())
	if err != nil {
		return ctx, nil, c.socksErr(err)
	}

	return ctx, conn, nil
}

func (c CustomHandler) ReadFromClient(ctx context.Context, client io.ReadCloser, remote io.WriteCloser) error {
	clientConn, ok := client.(net.Conn)
	if !ok {
		return errors.New("invalid client connection")
	}
	remoteConn, ok := remote.(net.Conn)
	if !ok {
		return errors.New("invalid remote connection")
	}

	pack := make([]byte, 2048)
	for {
		if ctx.Err() != nil {
			return nil
		}

		if err := clientConn.SetReadDeadline(time.Now().Add(time.Second)); err != nil {
			return err
		}

		n, err := clientConn.Read(pack)
		if err != nil {
			if errors.Is(err, io.EOF) {
				return nil
			}
			if isTimeoutErr(err) {
				continue
			}
			return err
		}

		if wErr := wsutil.WriteClientBinary(remoteConn, pack[:n]); wErr != nil {
			return wErr
		}
	}
}

func (c CustomHandler) ReadFromRemote(ctx context.Context, remote io.ReadCloser, client io.WriteCloser) error {
	remoteConn, ok := remote.(net.Conn)
	if !ok {
		return errors.New("invalid remote connection")
	}

	wsReader := wsutil.NewReader(remoteConn, ws.StateClientSide)
	pack := make([]byte, 2048)
	for {
		if ctx.Err() != nil {
			return nil
		}

		if err := remoteConn.SetReadDeadline(time.Now().Add(time.Second)); err != nil {
			return nil
		}

		header, err := wsReader.NextFrame()
		if err != nil {
			if errors.Is(err, io.EOF) {
				return nil
			}
			if isTimeoutErr(err) {
				continue
			}
			return err
		}

		switch header.OpCode {
		case ws.OpPing:
			wsutil.WriteClientMessage(remoteConn, ws.OpPong, nil)
			continue
		case ws.OpPong:
			continue
		case ws.OpClose:
			wsutil.WriteClientMessage(remoteConn, ws.OpClose, nil)
			return errors.New("connection closed")
		}

		for {
			n, err := wsReader.Read(pack)
			if n > 0 {
				if _, wErr := client.Write(pack[:n]); wErr != nil {
					return wErr
				}
			}
			if err != nil {
				if errors.Is(err, io.EOF) {
					break
				}
				return err
			}
		}
	}
}

func (c CustomHandler) Close(ctx context.Context) error {
	return nil
}

func (c CustomHandler) Refresh(ctx context.Context) {
}

func main() {
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug})))

	ctx, cancel := signal.NotifyContext(context.Background(), os.Kill, os.Interrupt)
	defer cancel()

	proxy := socks.Proxy{
		ServerAddr: ":1080",
		// Proxyhandler: CustomHandler{Auth: "mobinyentoken", Host: "localhost:4040", Path: "/"},
		Proxyhandler: CustomHandler{Auth: "mobinyentoken", Host: "93.127.180.181:4040", Path: "/"},
		Timeout:      time.Second * 10,
		Done:         make(chan struct{}),
	}

	slog.Info("starting socks server.", slog.String("address", proxy.ServerAddr))

	if err := proxy.Start(ctx); err != nil {
		slog.Error("error is: " + err.Error())
		return
	}

	slog.Info("server started")

	go func() {
		<-ctx.Done()
		proxy.Stop()
	}()

	<-proxy.Done

	dCtx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()
	if err := cleanup(dCtx, "mobinyentoken"); err != nil {
		ge.Throw(err)
	}
}

func cleanup(ctx context.Context, auth string) error {
	sURL := url.URL{
		Scheme: "http",
		// Host:     "localhost:4040",93.127.180.181:4040
		Host:     "93.127.180.181:4040",
		Path:     "/cleanup",
		RawQuery: "",
	}
	q := sURL.Query()
	q.Set("auth", auth)
	sURL.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, "POST", sURL.String(), nil)
	if err != nil {
		return err
	}

	client := http.Client{}
	res, err := client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.StatusCode != 200 {
		return errors.New("failed to cleanup user: " + res.Status + " (" + strconv.Itoa(res.StatusCode) + ")")
	}

	return nil
}

func isTimeoutErr(err error) bool {
	if nErr, ok := ge.As[net.Error](err); ok && nErr.Timeout() {
		return true
	}
	return false
}
