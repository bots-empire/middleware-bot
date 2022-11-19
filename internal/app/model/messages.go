package model

import "time"

//type UserMessages struct {
//	ChatNumber       int
//	UserID           int64
//	UserMessage      string
//	UserPhotoMessage string
//	MessageTime      time.Time
//}
//
//type AdminMessages struct {
//	ChatNumber        int
//	AdminID           int64
//	AdminMessage      string
//	AdminPhotoMessage string
//	MessageTime       time.Time
//}

type CommonMessages struct {
	ChatNumber        int
	AdminID           int64
	AdminMessage      string
	AdminPhotoMessage string
	UserID            int64
	UserMessage       string
	UserPhotoMessage  string
	MessageTime       time.Time
}
