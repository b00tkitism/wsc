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

const connReadSize = 2048

var _ encoding.TextMarshaler = &User{}

type connData struct {
	time int64
	id   int
}

type User struct {
	ID                   int64        `json:"id"`
	UsedTrafficBytes     atomic.Int64 `json:"used_bytes"`
	ReportedTrafficBytes atomic.Int64 `json:"reported_traffic_bytes"`

	LastTrafficUpdateTick atomic.Int64
	Conns                 map[net.Conn]connData

	Heap []byte

	connMutex    sync.Mutex
	maxConnCount int
	usedIds      []bool
}

func NewUser(id int64, usedTrafficBytes int64, maxConnCount int) *User {
	user := &User{
		ID:           id,
		Conns:        make(map[net.Conn]connData, maxConnCount),
		Heap:         make([]byte, connReadSize*2*maxConnCount),
		usedIds:      make([]bool, maxConnCount),
		maxConnCount: maxConnCount,
	}
	user.UsedTrafficBytes.Store(usedTrafficBytes)
	user.ReportedTrafficBytes.Store(0)
	user.LastTrafficUpdateTick.Store(0)
	return user
}

func (user *User) TCPBuffer(conn net.Conn) []byte {
	user.connMutex.Lock()
	defer user.connMutex.Unlock()
	if d, found := user.Conns[conn]; found {
		bufStart := (connReadSize*2)*(d.id+1) - connReadSize
		bufEnd := bufStart + connReadSize
		return user.Heap[bufStart:bufEnd]
	}
	return make([]byte, connReadSize)
}

func (user *User) WSBuffer(conn net.Conn) []byte {
	user.connMutex.Lock()
	defer user.connMutex.Unlock()
	if d, found := user.Conns[conn]; found {
		bufStart := (connReadSize * 2) * d.id
		bufEnd := bufStart + connReadSize
		return user.Heap[bufStart:bufEnd]
	}
	return make([]byte, connReadSize)
}

func (user *User) AddConn(conn net.Conn) (net.Conn, error) {
	user.connMutex.Lock()
	defer user.connMutex.Unlock()
	if _, exists := user.Conns[conn]; exists {
		return nil, errors.New("connection already exists")
	}
	var selectedConn net.Conn = nil
	selectedConnId := 0
	if len(user.Conns) >= user.maxConnCount {
		minTime := int64(math.MaxInt64)
		for c, d := range user.Conns {
			if d.time < minTime {
				minTime = d.time
				selectedConn = c
				selectedConnId = d.id
			}
		}
		if selectedConn != nil {
			delete(user.Conns, selectedConn)
		}
	} else {
		for i := 0; i < user.maxConnCount; i++ {
			if !user.usedIds[i] {
				selectedConnId = i
				user.usedIds[i] = true
				break
			}
		}
	}
	user.Conns[conn] = connData{
		time: nowns(),
		id:   selectedConnId,
	}
	return selectedConn, nil
}

func (user *User) RemoveConn(conn net.Conn) error {
	user.connMutex.Lock()
	defer user.connMutex.Unlock()
	if d, exists := user.Conns[conn]; exists {
		user.usedIds[d.id] = false
		delete(user.Conns, conn)
		return nil
	}
	return errors.New("connection doesn't exist")
}

func (user *User) MarshalText() (text []byte, err error) {
	return json.Marshal(user)
}
