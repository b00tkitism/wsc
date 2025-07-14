package proxy

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type User struct {
	Token            string        `json:"token"`
	UsedTrafficBytes uint64        `json:"used_bytes"`
	CreationDate     time.Time     `json:"creation_date"`
	Limitations      []*Limitation `json:"limitations"`
}

func NewEmptyUser() *User {
	return &User{
		Token:            uuid.NewString(),
		UsedTrafficBytes: 0,
		CreationDate:     time.Now(),
		Limitations:      []*Limitation{},
	}
}

func NewUser(token string, usedTrafficBytes uint64, creationDate time.Time, limitations []*Limitation) *User {
	return &User{
		Token:            token,
		UsedTrafficBytes: usedTrafficBytes,
		CreationDate:     creationDate,
		Limitations:      limitations,
	}
}

func (u *User) AddLimitation(limitation *Limitation) bool {
	for _, l := range u.Limitations {
		if l.Type == limitation.Type {
			return false
		}
	}
	u.Limitations = append(u.Limitations, limitation)
	return true
}

func (u *User) IsLimited() bool {
	for _, l := range u.Limitations {
		if l.IsLimited(u) {
			return true
		}
	}
	return false
}

func (u *User) ExportAsJson() (string, error) {
	result, err := json.Marshal(u)
	return string(result), err
}
