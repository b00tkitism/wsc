package proxy

import (
	"context"
	"errors"
	"github.com/gobwas/ws"
	"github.com/gobwas/ws/wsutil"
	"golang.org/x/sync/errgroup"
	"io"
	"log/slog"
	"net"
	"net/http"
	"sync"
	"time"
)

var _ http.Handler = &Proxy{}

type Authenticator interface {
	Authenticate(ctx context.Context, auth string) (int64, int64, error)
	ReportUsage(ctx context.Context, id int64, usedTraffic int64) error
}

type Proxy struct {
	MaximumConnectionsPerUser  int
	UsageReportTimeInterval    time.Duration
	UsageReportTrafficInterval int64
	Users                      map[int64]*User
	Auth                       Authenticator

	ipResolver *net.Resolver
	dialer     *net.Dialer
	userMutex  sync.Mutex
}

func NewProxy(authenticator Authenticator, maximumConnectionsPerUser int, usageReportTimeInterval time.Duration, usageReportTrafficInterval int64) *Proxy {
	return &Proxy{
		MaximumConnectionsPerUser:  maximumConnectionsPerUser,
		UsageReportTimeInterval:    usageReportTimeInterval,
		UsageReportTrafficInterval: usageReportTrafficInterval,
		Users:                      map[int64]*User{},
		Auth:                       authenticator,
		ipResolver:                 &net.Resolver{},
		dialer:                     &net.Dialer{},
	}
}

func (pro *Proxy) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	ctx := request.Context()

	auth := request.URL.Query().Get("auth")
	if auth == "" {
		http.Error(writer, "Authentication required", http.StatusBadRequest)
		slog.Debug("Request failed. Authentication required.", slog.String("client", request.RemoteAddr))
		return
	}

	uid, rate, err := pro.Auth.Authenticate(ctx, auth)
	if err != nil {
		if uid != 0 {
			if err := pro.cleanupUser(ctx, uid, false); err != nil {
				slog.Debug("Request failed. Couldn't cleanup user: "+err.Error(), slog.String("client", request.RemoteAddr), slog.Int64("user-id", uid))
			}
		}
		http.Error(writer, "Authentication failed: "+err.Error(), http.StatusBadRequest)
		slog.Debug("Request failed. Authentication failed: "+err.Error(), slog.String("client", request.RemoteAddr))
		return
	}

	if request.Method == "POST" && request.URL.Path == "/cleanup" {
		if err := pro.cleanupUser(ctx, uid, true); err != nil {
			http.Error(writer, "Failed to cleanup user: "+err.Error(), http.StatusInternalServerError)
			slog.Debug("Request failed. Couldn't cleanup user: "+err.Error(), slog.String("client", request.RemoteAddr), slog.Int64("user-id", uid))
			return
		}
		writer.WriteHeader(http.StatusOK)
		return
	}

	user := pro.findUser(ctx, uid, rate)

	endpoint := request.URL.Query().Get("ep")
	tcpAddr, err := parseEndpoint(ctx, pro.ipResolver, endpoint)
	if err != nil {
		http.Error(writer, "Failed to parse endpoint: "+err.Error(), http.StatusBadRequest)
		slog.Debug("Request failed. Failed to parse endpoint: "+err.Error(), slog.String("client", request.RemoteAddr))
		return
	}

	slog.Debug("New request", slog.String("client", request.RemoteAddr), slog.String("auth", auth), slog.Int64("user-id", uid), slog.String("tcp-addr", tcpAddr.String()))

	conn, _, _, err := ws.UpgradeHTTP(request, writer)
	if err != nil {
		http.Error(writer, "WebSocket upgrade failed: "+err.Error(), http.StatusBadRequest)
		slog.Debug("Failed to upgrade WebSocket: "+err.Error(), slog.String("client", request.RemoteAddr), slog.Int64("user-id", uid))
		return
	}

	defer func() {
		if err := pro.cleanupUserConn(ctx, user, conn); err != nil {
			slog.Error("Failed to cleanup user connection: "+err.Error(), slog.String("client", request.RemoteAddr), slog.Int64("user-id", uid))
		}
		if err := conn.Close(); err != nil {
			slog.Debug("Failed to close connection: "+err.Error(), slog.String("client", request.RemoteAddr), slog.Int64("user-id", uid))
		}
	}()

	if err := pro.pipeConn(ctx, user, conn, tcpAddr); err != nil {
		slog.Debug("Failed to pipe connection: "+err.Error(), slog.String("client", request.RemoteAddr), slog.Int64("user-id", uid))
		http.Error(writer, "Failed to pipe connection: "+err.Error(), http.StatusInternalServerError)
	}
}

func (pro *Proxy) pipeConn(ctx context.Context, user *User, conn net.Conn, target *net.TCPAddr) error {
	if poppedConn, err := user.AddConn(conn); err != nil {
		return err
	} else {
		if poppedConn != nil {
			poppedConn.Close()
		}
	}

	tcpConn, err := pro.dialer.DialContext(ctx, "tcp", target.String())
	if err != nil {
		return err
	}
	defer tcpConn.Close()

	eg, ctx := errgroup.WithContext(ctx)
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	eg.Go(func() error {
		return pro.pipeWSToTCP(ctx, user, conn, tcpConn)
	})
	eg.Go(func() error {
		err := pro.pipeTCPToWS(ctx, user, tcpConn, conn)
		cancel()
		return err
	})

	return eg.Wait()
}

func (pro *Proxy) pipeWSToTCP(ctx context.Context, user *User, wsConn net.Conn, tcpConn net.Conn) error {
	wsLReader, err := user.ConnReader(wsConn)
	if err != nil {
		return err
	}
	wsWriter, err := user.ConnWriter(wsConn)
	if err != nil {
		return err
	}

	wsReader := wsutil.NewReader(wsLReader, ws.StateServerSide)
	pack := user.WSBuffer(wsConn)

	for {
		if ctx.Err() != nil {
			return nil
		}

		if err := wsConn.SetReadDeadline(time.Now().Add(time.Second)); err != nil {
			return err
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
			wsutil.WriteServerMessage(wsWriter, ws.OpPong, nil)
			continue
		case ws.OpPong:
			continue
		case ws.OpClose:
			wsutil.WriteServerMessage(wsWriter, ws.OpClose, nil)
			return nil
		}

		for {
			n, err := wsReader.Read(pack)
			if n > 0 {
				if _, wErr := tcpConn.Write(pack[:n]); wErr != nil {
					return wErr
				} else {
					user.UsedTrafficBytes.Add(int64(n))
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

func (pro *Proxy) pipeTCPToWS(ctx context.Context, user *User, tcpConn net.Conn, wsConn net.Conn) error {
	wsWriter, err := user.ConnWriter(wsConn)
	if err != nil {
		return err
	}

	pack := user.TCPBuffer(wsConn)

	for {
		if ctx.Err() != nil {
			return nil
		}

		if err := tcpConn.SetReadDeadline(time.Now().Add(time.Second)); err != nil {
			return err
		}

		n, err := tcpConn.Read(pack)
		if err != nil {
			if errors.Is(err, io.EOF) {
				return nil
			}
			if isTimeoutErr(err) {
				continue
			}
			return err
		}

		user.UsedTrafficBytes.Add(int64(n))

		if err := wsutil.WriteServerBinary(wsWriter, pack[:n]); err != nil {
			return err
		}
	}
}

func (pro *Proxy) findUser(ctx context.Context, uid int64, rateLimit int64) *User {
	pro.userMutex.Lock()
	defer pro.userMutex.Unlock()
	if user, exists := pro.Users[uid]; exists {
		pro.reportUser(ctx, user, false)
		return user
	}
	user := NewUser(uid, 0, pro.MaximumConnectionsPerUser, rateLimit)
	pro.Users[uid] = user
	return user
}

func (pro *Proxy) cleanupUser(ctx context.Context, uid int64, forceReport bool) error {
	pro.userMutex.Lock()
	defer pro.userMutex.Unlock()
	if user, exists := pro.Users[uid]; !exists {
		return errors.New("user doesn't exist")
	} else {
		pro.reportUser(ctx, user, forceReport)
		user.Cleanup()
		delete(pro.Users, uid)
		return nil
	}
}

func (pro *Proxy) reportUser(ctx context.Context, user *User, force bool) bool {
	usedTraffic := user.UsedTrafficBytes.Load()
	reportedTraffic := user.ReportedTrafficBytes.Load()
	trafficResult := usedTraffic - reportedTraffic
	now := nowns()
	if !force {
		if trafficResult == 0 {
			return false
		}
		if trafficResult < pro.UsageReportTrafficInterval && time.Duration(now-user.LastTrafficUpdateTick.Load()) < pro.UsageReportTimeInterval {
			return false
		}
	}
	go func() {
		err := pro.Auth.ReportUsage(ctx, user.ID, trafficResult)
		if err == nil {
			user.ReportedTrafficBytes.Store(usedTraffic)
			user.LastTrafficUpdateTick.Store(now)
		}
	}()
	return true
}

func (pro *Proxy) cleanupUserConn(ctx context.Context, user *User, conn net.Conn) error {
	pro.userMutex.Lock()
	defer pro.userMutex.Unlock()
	sent := pro.reportUser(ctx, user, false)
	err := user.RemoveConn(conn)
	if user.ConnCount() == 0 {
		if !sent {
			pro.reportUser(ctx, user, true)
		}
		delete(pro.Users, user.ID)
	}
	return err
}
