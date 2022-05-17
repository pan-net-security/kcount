package main

import (
	"flag"
	"strings"
)

// Flags are command line flags.
type Flags struct {
	allNamespaces bool
	age           bool
	daemon        bool
	kind          kinds
	labelSelector string
	namespace     string
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

// parseFlags parses and returns command line flags (options).
func parseFlags() Flags {
	var f Flags

	flag.BoolVar(&f.allNamespaces, "A", false, "count objects across all namespaces")
	flag.BoolVar(&f.age, "a", false, "show also age of newest and oldest object")
	flag.BoolVar(&f.daemon, "d", false, "run as daemon exposing prometheus metrics")
	flag.Var(&f.kind, "k", "object kind or kinds (default is all supported)")
	flag.StringVar(&f.labelSelector, "l", "", "label selector (e.g. env=prod)")
	flag.StringVar(&f.namespace, "n", "", "namespace (default comes from kubeconfig)")

	flag.Parse()

	// Set default values.
	if len(f.kind) == 0 {
		f.kind = []string{
			"deployment",
			"pod",
			"configmap",
			"secret",
			"ingress",
			"service",
		}
	}

	return f
}
