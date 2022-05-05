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
		log.SetFlags(0)
		log.SetPrefix(os.Args[0] + ": ")
	}

	clusters, err := Clusters(flag.Args(), flags.allNamespaces)
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
					if flags.age {
						objectsNewest.WithLabelValues(obj.cluster, obj.namespace, obj.labelSelector, obj.kind).Set(float64(obj.newest.Unix()))
						objectsOldest.WithLabelValues(obj.cluster, obj.namespace, obj.labelSelector, obj.kind).Set(float64(obj.oldest.Unix()))
					}
				}
				time.Sleep(2 * time.Second)
			}
		}()
		addr, urlPath := ":2112", "/metrics"
		log.Printf("exposing Prometheus metrics at %s%s", addr, urlPath)
		log.Fatal(exposeMetrics(addr, urlPath))
	} else { // running as CLI app
		objects := CountObjectsAcrossClusters(clusters, flags)
		SortObjects(objects)
		PrintObjects(objects, flags.age)
	}
}
