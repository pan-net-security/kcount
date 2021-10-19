package main

import (
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func recordMetrics(clusters []Cluster, flags Flags) {
	objectCount := promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "objects_total",
			Help: "The total number of kubernetes objects",
		},
		[]string{"cluster", "namespace", "labelSelector", "kind"},
	)
	go func() {
		for {
			objects := CountObjectsAcrossClusters(clusters, flags)
			for _, obj := range objects {
				objectCount.WithLabelValues(obj.cluster, obj.namespace, obj.labelSelector, obj.kind).Set(float64(obj.count))

			}
			time.Sleep(2 * time.Second)
		}
	}()
}

func exposeMetrics() {
	http.Handle("/metrics", promhttp.Handler())
	http.ListenAndServe(":2112", nil)
}
