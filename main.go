package main

import (
	"flag"
	"log"
	"os"
	"sort"
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
			obj, err := CountObjects(cluster, flags.kind, flags.labelSelector, flags.timeout)
			if err != nil {
				log.Print(err)
				return
			}
			mu.Lock()
			objects = append(objects, obj)
			mu.Unlock()
		}(cluster)
	}
	wg.Wait()

	sort.Slice(objects, func(i, j int) bool {
		if objects[i].cluster != objects[j].cluster {
			return objects[i].cluster < objects[j].cluster
		}
		if objects[i].namespace != objects[j].namespace {
			return objects[i].namespace < objects[j].namespace
		}
		if objects[i].count != objects[j].count {
			return objects[i].count > objects[j].count
		}
		return false
	})
	PrintObjects(objects, flags.age)
}
