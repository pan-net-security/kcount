package main

import (
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	podsCount = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "pods_total",
			Help: "The total number of pods",
		},
		[]string{"cluster", "namespace", "labelSelector"},
	)
)

func recordMetrics(clusters []Cluster, flags Flags) {
	go func() {
		for {
			objects := CountObjectsAcrossClusters(clusters, flags)
			for _, obj := range objects {
				switch obj.kind {
				case "pod":
					podsCount.WithLabelValues(obj.cluster, obj.namespace, obj.labelSelector).Set(float64(obj.count))

				}
			}
			time.Sleep(2 * time.Second)
		}
	}()
}

func exposeMetrics() {
	http.Handle("/metrics", promhttp.Handler())
	http.ListenAndServe(":2112", nil)
}
