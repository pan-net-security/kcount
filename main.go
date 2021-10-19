package main

import (
	"flag"
	"log"
	"os"
	"time"
)

func main() {
	flags := parseFlags()

	if !flags.daemon { // running as CLI app
		log.SetPrefix(os.Args[0] + ": ")
		log.SetFlags(0)
	}

	clusters, err := Clusters(flag.Args())
	if err != nil {
		log.Fatalf("getting cluster configs: %v", err)
	}
	if len(clusters) == 0 {
		log.Fatal("run in cluster or supply at least one kubeconfig")
	}

	if flags.daemon {
		go func() {
			for {
				objects := CountObjectsAcrossClusters(clusters, flags)
				for _, obj := range objects {
					objectsCount.WithLabelValues(obj.cluster, obj.namespace, obj.labelSelector, obj.kind).Set(float64(obj.count))
					objectsNewest.WithLabelValues(obj.cluster, obj.namespace, obj.labelSelector, obj.kind).Set(float64(obj.newest.Unix()))
					objectsOldest.WithLabelValues(obj.cluster, obj.namespace, obj.labelSelector, obj.kind).Set(float64(obj.oldest.Unix()))
				}
				time.Sleep(2 * time.Second)
			}
		}()
		log.Fatal(exposeMetrics(":2112", "/metrics"))
	} else { // running as CLI app
		objects := CountObjectsAcrossClusters(clusters, flags)
		SortObjects(objects)
		PrintObjects(objects)
	}
}
