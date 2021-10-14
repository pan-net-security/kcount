package main

import "flag"

type Flags struct {
	labelSelector string
	kind          string
	age           bool
	timeout       int64
	byCount       bool
}

func ParseFlags() Flags {
	var f Flags
	flag.StringVar(&f.labelSelector, "l", "", "label selector, e.g. env=prod")
	flag.StringVar(&f.kind, "k", "pod", "object kind")
	flag.BoolVar(&f.age, "a", false, "print also age")
	flag.Int64Var(&f.timeout, "t", 5, "cluster API call timeout in seconds")
	flag.BoolVar(&f.byCount, "c", false, "sort by count (default by cluster and namespace)")
	flag.Parse()
	return f
}
