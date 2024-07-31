package main

import (
	"labs.lesiw.io/ops/golang"
	"lesiw.io/ops"
)

type Ops struct {
	golang.Ops
}

func main() { ops.Handle(Ops{}) }
