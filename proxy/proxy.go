package proxy

import (
	"net"
	"time"
)

type Proxy struct {
	ListenAddress         net.Addr
	ListenPort            uint16
	MetricSendingInterval time.Duration
	Users                 []User
}

func NewProxy() {}
