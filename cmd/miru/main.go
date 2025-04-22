// Command miru provides a command-line tool for viewing package documentation.
package main

import (
	"fmt"
	"os"

	"github.com/ka2n/miru/cli"
	"github.com/ka2n/miru/log"
	"github.com/morikuni/failure/v2"
)

func main() {
	debug := os.Getenv("MIRU_DEBUG") == "1"
	if debug {
		log.InitLogger()
		log.EnableGlobalHTTP()
	}

	if err := cli.Run(); err != nil {
		if !debug {
			var userMessage string
			if fmsg := failure.MessageOf(err); fmsg != "" {
				userMessage = fmsg.String()
			} else {
				userMessage = err.Error()
			}
			fmt.Fprintf(os.Stderr, "Error: %v\n", userMessage)
		} else {
			fmt.Fprintf(os.Stderr, "Error: %+v\n", err)
			log.Error("Command failed",
				"error", err,
				"stack", fmt.Sprintf("%+v", err),
			)
		}
		os.Exit(1)
	}
}
