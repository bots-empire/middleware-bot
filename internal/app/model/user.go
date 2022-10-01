package model

type User struct {
	ID      int64 `json:"id"`
	IsAdmin bool  `json:"is_admin"`
}
