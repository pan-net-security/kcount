package main

import (
	"flag"
	"log"
	"os"
	"sync"
)

func main() {
	flags := ParseFlags()

	log.SetPrefix(os.Args[0] + ": ")
	log.SetFlags(0)

	clusters, err := Clusters(flag.Args())
	if err != nil {
		log.Fatalf("getting cluster configs: %v", err)
	}
	if len(clusters) == 0 {
		log.Fatal("run in cluster or supply at least one kubeconfig")
	}

	var mu sync.Mutex
	var objects []Object

	var wg sync.WaitGroup
	for _, cluster := range clusters {
		wg.Add(1)
		go func(cluster Cluster) {
			defer wg.Done()
			obj, err := CountObjects(cluster, flags.kind, flags.labelSelector)
			if err != nil {
				log.Fatal(err)
			}
			mu.Lock()
			objects = append(objects, obj)
			mu.Unlock()
		}(cluster)
	}
	wg.Wait()

	SortObjects(objects, flags.byCount)
	PrintObjects(objects, flags.age)
}
