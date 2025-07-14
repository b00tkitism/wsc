package proxy

import (
	"net"
	"sync"
	"time"

	"github.com/b00tkitism/wsc/metric"
)

type Proxy struct {
	ListenAddress             net.Addr
	MetricSendingInterval     time.Duration
	Metric                    *metric.Metric
	MaximumConnectionsPerUser uint8

	mu    sync.RWMutex
	Users []*User
}

func NewProxy(listenAddress net.Addr, metricSendingInterval time.Duration, metricServer *metric.Metric, maximumConnectionsPerUser uint8) *Proxy {
	return &Proxy{
		ListenAddress:             listenAddress,
		MetricSendingInterval:     metricSendingInterval,
		Metric:                    metricServer,
		MaximumConnectionsPerUser: maximumConnectionsPerUser,
		Users:                     []*User{},
	}
}

func (p *Proxy) GetUserByToken(token string) *User {
	p.mu.RLock()
	defer p.mu.RUnlock()
	for _, user := range p.Users {
		if user.Token == token {
			return user
		}
	}
	return nil
}

func (p *Proxy) AddUser(user *User) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.Users = append(p.Users, user)
}

func (p *Proxy) Run() {
	// foo
}

func (p *Proxy) SendMetrics() {
	p.mu.RLock()
	defer p.mu.RUnlock()
	for _, user := range p.Users {
		data, err := user.ExportAsJson()
		if err != nil {
			continue
		}
		if err := p.Metric.SendMetric(data); err != nil {
		}
	}
}
