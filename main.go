package main

import (
	"log"

	"github.com/alecthomas/kong"
	"github.com/alipali737/climbing-society-seats-app/climbing-society-seats-app/cmd/run"
	"github.com/alipali737/climbing-society-seats-app/climbing-society-seats-app/cmd/utility"
)

var cli struct {
	Utility utility.Utility `cmd:"" help:"Choose from a variety of utility commands"`
	Run     run.Run         `cmd:"" help:"Run the main webserver"`
}

func main() {
	ctx := kong.Parse(
		&cli,
		kong.ConfigureHelp(kong.HelpOptions{
			Tree: true,
		}),
	)
	if err := ctx.Run(); err != nil {
		log.Fatal(err)
	}
}
