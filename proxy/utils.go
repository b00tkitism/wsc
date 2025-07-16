package proxy

import (
	"context"
	"github.com/itsabgr/ge"
	"net"
	"strconv"
	"time"
)

func nowns() int64 {
	return time.Now().UnixNano()
}

func parseEndpoint(ctx context.Context, resolver *net.Resolver, endpoint string) (*net.TCPAddr, error) {
	host, portStr, err := net.SplitHostPort(endpoint)
	if err != nil {
		return nil, err
	}

	ips, err := resolver.LookupIPAddr(ctx, host)
	if err != nil {
		return nil, err
	}

	port, _ := strconv.Atoi(portStr)

	return &net.TCPAddr{
		IP:   ips[0].IP,
		Port: port,
		Zone: ips[0].Zone,
	}, nil
}

func isTimeoutErr(err error) bool {
	if nErr, ok := ge.As[net.Error](err); ok && nErr.Timeout() {
		return true
	}
	return false
}
