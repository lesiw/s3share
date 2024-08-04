package main

import (
	"os"

	"labs.lesiw.io/ops/golang"
	"lesiw.io/ops"
)

type Ops struct {
	golang.Ops
}

func main() {
	if len(os.Args) < 2 {
		os.Args = append(os.Args, "noop")
	}
	ops.Handle(Ops{})
}

func (Ops) Noop() {}
