package main

import (
	"github.com/alexflint/go-arg"

	"github.com/cablehead/x/cmd/exec"
	"github.com/cablehead/x/cmd/merge"
	"github.com/cablehead/x/cmd/split"
)

// port range: 8163-8180

// ideas:
// https://github.com/liujianping/job

// alternates:
// https://github.com/cosiner/flag

type Command interface {
	Do() error
}

func main() {
	var args struct {
		Merge *merge.Command `arg:"subcommand:merge"`
		Split *split.Command `arg:"subcommand:split"`
		Exec  *exec.Command  `arg:"subcommand:exec"`
	}

	p := arg.MustParse(&args)

	cmd, ok := p.Subcommand().(Command)
	if !ok {
		p.Fail("missing subcommand")
	}

	err := cmd.Do()
	if err != nil {
		panic(err)
	}
}
