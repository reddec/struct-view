package bbol

import "time"

type User struct {
	ID        int64  `bbolt:",primary"`
	Name      string `bbolt:",index"`
	EMail     string `bbolt:",unique"`
	Tags      []string
	CreatedAt time.Time `bbolt:",range"`
}
