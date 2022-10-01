package model

import (
	"sync"
)

type UpdateInfo struct {
	Mu      *sync.Mutex
	Counter int
}

var UpdateStatistic *UpdateInfo
