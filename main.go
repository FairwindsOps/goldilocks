package main

import (
	"github.com/reactiveops/vpa-analysis/cmd"
)

var (
	// VERSION is set during build
	VERSION = "v0.2.0"
)

func main() {
	cmd.Execute(VERSION)
}
