package proxy

import "time"

type User struct {
	usedTrafficBytes uint64
	creationDate     time.Time
	limitations      []Limitation
}

func NewEmptyUser() *User {
	return &User{
		usedTrafficBytes: uint64(0),
		creationDate:     time.Now(),
		limitations:      []Limitation{},
	}
}

func NewUser(usedTrafficBytes uint64, creationDate time.Time, limitations []Limitation) *User {
	return &User{
		usedTrafficBytes: usedTrafficBytes,
		creationDate:     creationDate,
		limitations:      limitations,
	}
}

func (u *User) AddLimitation(limitation *Limitation) bool {
	for _, _limitation := range u.limitations {
		if _limitation.Type == limitation.Type {
			return false
		}
	}

	u.limitations = append(u.limitations, *limitation)
	return true
}

func (u *User) IsLimited() bool {
	for _, limitation := range u.limitations {
		if limitation.IsLimited(u) {
			return true
		}
	}
	return false
}
