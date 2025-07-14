package proxy

import "time"

type LimitationType uint8

const (
	LimitationTypeTrafficBytes LimitationType = iota
	LimitationTypeDuration
)

type Limitation struct {
	Type         LimitationType `json:"type"`
	TrafficBytes uint64         `json:"traffic_bytes,omitempty"`
	Duration     time.Duration  `json:"duration,omitempty"`
}

func NewTrafficLimitation(trafficBytes uint64) *Limitation {
	return &Limitation{
		Type:         LimitationTypeTrafficBytes,
		TrafficBytes: trafficBytes,
	}
}

func NewDurationLimitation(duration time.Duration) *Limitation {
	return &Limitation{
		Type:     LimitationTypeDuration,
		Duration: duration,
	}
}

func (l *Limitation) IsLimited(user *User) bool {
	switch l.Type {
	case LimitationTypeTrafficBytes:
		return user.UsedTrafficBytes >= l.TrafficBytes
	case LimitationTypeDuration:
		return time.Since(user.CreationDate) >= l.Duration
	default:
		return false
	}
}
