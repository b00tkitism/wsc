package proxy

import (
	"encoding"
	"encoding/json"
	"errors"
	"math"
	"net"
	"sync"
	"sync/atomic"
)

var _ encoding.TextMarshaler = &User{}

type User struct {
	ID                   int64        `json:"id"`
	UsedTrafficBytes     atomic.Int64 `json:"used_bytes"`
	ReportedTrafficBytes atomic.Int64 `json:"reported_traffic_bytes"`

	LastTrafficUpdateTick atomic.Int64
	Conns                 map[net.Conn]int64

	connMutex sync.Mutex
}

func NewUser(id int64, usedTrafficBytes int64, maxConnCount int) *User {
	user := &User{
		ID:    id,
		Conns: make(map[net.Conn]int64, maxConnCount),
	}
	user.UsedTrafficBytes.Store(usedTrafficBytes)
	user.ReportedTrafficBytes.Store(0)
	user.LastTrafficUpdateTick.Store(0)
	return user
}

func (user *User) AddConn(conn net.Conn, limit int) (net.Conn, error) {
	user.connMutex.Lock()
	defer user.connMutex.Unlock()
	if _, exists := user.Conns[conn]; exists {
		return nil, errors.New("connection already exists")
	}
	var selectedConn net.Conn = nil
	if len(user.Conns) >= limit {
		minTime := int64(math.MaxInt64)
		for c, time := range user.Conns {
			if time < minTime {
				minTime = time
				selectedConn = c
			}
		}
		if selectedConn != nil {
			delete(user.Conns, selectedConn)
		}
	}
	user.Conns[conn] = nowns()
	return selectedConn, nil
}

func (user *User) RemoveConn(conn net.Conn) error {
	user.connMutex.Lock()
	defer user.connMutex.Unlock()
	if _, exists := user.Conns[conn]; !exists {
		return errors.New("connection doesn't exist")
	}
	delete(user.Conns, conn)
	return nil
}

func (user *User) MarshalText() (text []byte, err error) {
	return json.Marshal(user)
}
