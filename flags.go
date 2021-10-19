package main

import (
	"flag"
	"strings"
)

type Flags struct {
	labelSelector string
	kind          kinds
	age           bool
	byCount       bool
	daemon        bool
}

type kinds []string

func (k *kinds) String() string {
	return strings.Join(*k, ", ")
}

func (k *kinds) Set(value string) error {
	values := strings.Split(value, ",")
	for i := range values {
		values[i] = strings.TrimSpace(values[i])
	}
	*k = append(*k, values...)
	return nil
}

func parseFlags() Flags {
	var f Flags

	flag.StringVar(&f.labelSelector, "l", "", "label selector (e.g. env=prod)")
	flag.Var(&f.kind, "k", "object kind or kinds (default pod)")
	flag.BoolVar(&f.age, "a", false, "print also age")
	flag.BoolVar(&f.byCount, "c", false, "sort by count (default by cluster and namespace)")
	flag.BoolVar(&f.daemon, "d", false, "run as daemon exposing prometheus metrics")

	flag.Parse()

	if len(f.kind) == 0 { // set default value
		f.kind = append(f.kind, "pod")
	}

	return f
}
