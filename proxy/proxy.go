package proxy

import (
	"net"
	"time"

	"github.com/b00tkitism/wsc/metric"
	"github.com/google/uuid"
)

type Proxy struct {
	ListenAddress             net.Addr
	MetricSendingInterval     time.Duration
	Metric                    *metric.Metric
	MaximumConnectionsPerUser uint8

	Users map[string]*User
}

func NewProxy(listenAddress net.Addr, metricSendingInterval time.Duration, metricServer *metric.Metric, maximumConnectionsPerUser uint8) *Proxy {
	return &Proxy{
		ListenAddress:             listenAddress,
		MetricSendingInterval:     metricSendingInterval,
		Metric:                    metricServer,
		MaximumConnectionsPerUser: maximumConnectionsPerUser,
		Users:                     map[string]*User{},
	}
}

func (p *Proxy) AddUser(user *User) string {
	token := uuid.NewString()
	p.Users[token] = user
	return token
}

func (p *Proxy) Run() {
	// foo
}

func (p *Proxy) SendMetrics() {
	for _, user := range p.Users {
		data, err := user.ExportAsJson()
		if err != nil {
			continue
		}
		if err := p.Metric.SendMetric(data); err != nil {
		}
	}
}
