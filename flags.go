package main

import (
	"flag"
	"strings"
)

type Flags struct {
	daemon        bool
	kind          kinds
	labelSelector string
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
	flag.BoolVar(&f.daemon, "d", false, "run as daemon exposing prometheus metrics")

	flag.Parse()

	if len(f.kind) == 0 { // set default value
		f.kind = append(f.kind, "pod")
	}

	return f
}
