package model

import "time"

type AnonymousChat struct {
	ChatNumber      int
	AdminId         int64
	UserId          int64
	Note            string
	ChatStart       bool
	LastMessageTime time.Time
}
