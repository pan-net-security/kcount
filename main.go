package main

import (
	"flag"
	"log"
	"os"
)

func main() {
	flags := parseFlags()

	clusters, err := Clusters(flag.Args())
	if err != nil {
		log.Fatalf("getting cluster configs: %v", err)
	}
	if len(clusters) == 0 {
		log.Fatal("run in cluster or supply at least one kubeconfig")
	}

	if flags.daemon {
		recordMetrics(clusters, flags)
		exposeMetrics()
	} else { // running as CLI app
		log.SetPrefix(os.Args[0] + ": ")
		log.SetFlags(0)

		objects := CountObjectsAcrossClusters(clusters, flags)
		SortObjects(objects)
		PrintObjects(objects, flags.age)
	}
}
