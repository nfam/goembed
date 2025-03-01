package main

import (
	"flag"

	"github/nfam/goembed"
)

func main() {
	var (
		in      string
		out     string
		gobuild string
		pkg     string
		tsp     int64
	)

	flag.StringVar(&in, "i", "", "path to a file or directory containing asset")
	flag.StringVar(&out, "o", "", "path to the .go file to generate")
	flag.StringVar(&gobuild, "b", "", "go:build option")
	flag.StringVar(&pkg, "p", "embed", "package name of .go file")
	flag.Int64Var(&tsp, "t", 0, "timestamp in Unix seconds")
	flag.Parse()

	if err := goembed.Generate(in, out, gobuild, "goembed", pkg, tsp); err != nil {
		panic(err)
	}
}
