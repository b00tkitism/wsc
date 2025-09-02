package proxy

import (
	"context"
	"net"
	"strconv"
	"time"

	"github.com/itsabgr/ge"
)

type endpointAddr struct {
	ip   net.IP
	port uint16
	zone string
}

func nowns() int64 {
	return time.Now().UnixNano()
}

func parseEndpointAddr(ctx context.Context, resolver *net.Resolver, endpoint string) (*endpointAddr, error) {
	host, portStr, err := net.SplitHostPort(endpoint)
	if err != nil {
		return nil, err
	}

	ips, err := resolver.LookupIPAddr(ctx, host)
	if err != nil {
		return nil, err
	}

	port, _ := strconv.Atoi(portStr)

	return &endpointAddr{
		ip:   ips[0].IP,
		port: uint16(port),
		zone: ips[0].Zone,
	}, nil
}

// func parseEndpointTCP(ctx context.Context, resolver *net.Resolver, endpoint string) (*net.TCPAddr, error) {
// 	host, portStr, err := net.SplitHostPort(endpoint)
// 	if err != nil {
// 		return nil, err
// 	}

// 	ips, err := resolver.LookupIPAddr(ctx, host)
// 	if err != nil {
// 		return nil, err
// 	}

// 	port, _ := strconv.Atoi(portStr)

// 	return &net.TCPAddr{
// 		IP:   ips[0].IP,
// 		Port: port,
// 		Zone: ips[0].Zone,
// 	}, nil
// }

// func parseEndpointUDP(ctx context.Context, resolver *net.Resolver, endpoint string) (*net.UDPAddr, error) {
// 	host, portStr, err := net.SplitHostPort(endpoint)
// 	if err != nil {
// 		return nil, err
// 	}

// 	ips, err := resolver.LookupIPAddr(ctx, host)
// 	if err != nil {
// 		return nil, err
// 	}

// 	port, _ := strconv.Atoi(portStr)

// 	return &net.UDPAddr{
// 		IP:   ips[0].IP,
// 		Port: port,
// 		Zone: ips[0].Zone,
// 	}, nil
// }

func isTimeoutErr(err error) bool {
	if nErr, ok := ge.As[net.Error](err); ok && nErr.Timeout() {
		return true
	}
	return false
}
