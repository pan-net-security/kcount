// Kcount counts Kubernetes objects across clusters.
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
				counts := CountObjectsAcrossClusters(clusters, flags)
				for _, count := range counts {
					objectsCount.WithLabelValues(count.Cluster, count.Namespace, count.LabelSelector, count.Kind).Set(float64(count.Count))
					if flags.age {
						objectsNewest.WithLabelValues(count.Cluster, count.Namespace, count.LabelSelector, count.Kind).Set(float64(count.Newest.Unix()))
						objectsOldest.WithLabelValues(count.Cluster, count.Namespace, count.LabelSelector, count.Kind).Set(float64(count.Oldest.Unix()))
					}
				}
				time.Sleep(2 * time.Second)
			}
		}()
		addr, urlPath := ":2112", "/metrics"
		log.Printf("exposing Prometheus metrics at %s%s", addr, urlPath)
		log.Fatal(exposeMetrics(addr, urlPath))
	} else { // running as CLI app
		counts := CountObjectsAcrossClusters(clusters, flags)
		counts.Sort()
		counts.Print(flags.age)
	}
}
