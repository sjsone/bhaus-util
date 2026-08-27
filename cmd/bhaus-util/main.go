package main

import (
	"os"

	"github.com/sjsone/bhaus-util/pkg/cli"
)

func main() {
	os.Exit(cli.Run(os.Args))
}
