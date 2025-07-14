package metric

import "net"

type Metric struct {
	snmpServerAddress net.Addr
}

func NewMetric(address net.Addr) *Metric {
	return &Metric{snmpServerAddress: address}
}
func (m *Metric) SendMetric(data string) error {
	return nil
}
