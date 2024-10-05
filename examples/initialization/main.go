package main

import (
	_ "embed"
	"os"

	"github.com/axone-protocol/prolog"
)

//go:embed hello.pl
var hello string

func main() {
	p := prolog.New(nil, os.Stdout)
	if err := p.Exec(hello); err != nil {
		panic(err)
	}
}
