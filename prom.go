package main

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	objectsCount = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "objects_count",
			Help: "The total number of kubernetes objects",
		},
		[]string{"cluster", "namespace", "labelSelector", "kind"},
	)
	objectsNewest = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "objects_newest",
			Help: "The age of the newest kubernetes object in Unix time",
		},
		[]string{"cluster", "namespace", "labelSelector", "kind"},
	)
	objectsOldest = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "objects_oldest",
			Help: "The age of the oldest kubernetes object in Unix time",
		},
		[]string{"cluster", "namespace", "labelSelector", "kind"},
	)
)

func exposeMetrics(addr, urlPath string) error {
	http.Handle(urlPath, promhttp.Handler())
	return http.ListenAndServe(addr, nil)
}
