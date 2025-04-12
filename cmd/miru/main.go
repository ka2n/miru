// Command miru provides a command-line tool for viewing package documentation.
package main

import (
	"fmt"
	"os"

	"github.com/ka2n/miru/cli"
	"github.com/morikuni/failure/v2"
)

func main() {
	if err := cli.Run(); err != nil {
		var userMessage string
		if fmsg := failure.MessageOf(err); fmsg != "" {
			userMessage = fmsg.String()
		} else {
			userMessage = err.Error()
		}
		fmt.Fprintf(os.Stderr, "Error: %v\n", userMessage)
		// TODO: if verbose mode, print detials like error code and callstack
		os.Exit(1)
	}
}
