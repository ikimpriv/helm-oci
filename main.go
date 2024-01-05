package main

import (
	"fmt"
	cli "github.com/mkideal/cli"
	"os"
)

type TokenResponseStruct struct {
	Token string `json:"token"`
}

var root = &cli.Command{
	Desc: "",
	Fn: func(ctx *cli.Context) error {
		ctx.String(ctx.Usage())
		return nil
	},
}

var help = cli.HelpCommand("display help information")

func main() {
	if err := cli.Root(root,
		cli.Tree(help),
		cli.Tree(ls),
		cli.Tree(tags),
		cli.Tree(rm),
	).Run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
