package metric

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	RoomCountPrivate = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "active_rooms",
		Help: "Number of active rooms",
		ConstLabels: map[string]string{
			"type": "private",
		},
	})
	RoomCountPublic = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "active_rooms",
		Help: "Number of active rooms",
		ConstLabels: map[string]string{
			"type": "public",
		},
	})
	NewUsersGenerated = promauto.NewCounter(prometheus.CounterOpts{
		Name: "new_users",
		Help: "Number of users generated",
	})
)
