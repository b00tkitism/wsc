package proxy

import "time"

type LimitationType uint8

const (
	LimitationTypeTrafficBytes LimitationType = iota
	LimitationTypeDuration
)

type Limitation struct {
	Type  LimitationType
	Value any
}

func NewTrafficLimitation(trafficBytes uint64) *Limitation {
	return &Limitation{
		Type:  LimitationTypeTrafficBytes,
		Value: trafficBytes,
	}
}

func NewDurationLimitation(duration time.Duration) *Limitation {
	return &Limitation{
		Type:  LimitationTypeDuration,
		Value: duration,
	}
}

func (l *Limitation) IsLimited(user *User) bool {
	switch l.Type {
	case LimitationTypeTrafficBytes:
		return user.usedTrafficBytes >= l.Value.(uint64)
	case LimitationTypeDuration:
		return user.creationDate.Before(user.creationDate.Add(l.Value.(time.Duration)))
	default:
		return false
	}
}
