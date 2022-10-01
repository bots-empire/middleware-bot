package model

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

//goland:noinspection ALL
var (
	//restart
	HandleRestart = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "count_of_restart",
			Help: "Total count of restart",
		},
		[]string{"service_restart"},
	)

	//livez
	HandleLiveTime = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "live_time",
			Help: "Show live time",
		},
		[]string{"service_live"},
	)

	// updates
	HandleUpdates = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "count_of_handle_updates",
			Help: "Total count of handle updates",
		},
		[]string{"bot_link", "bot_name"},
	)
)
