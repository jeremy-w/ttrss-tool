// vi: set noet ts=4 sw=4 ft=go tw=79:
/* ttrss-tool manipulates tiny-tiny-rss subscriptions. */
package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"sort"
	"strings"
)

var _ = http.StatusContinue

// Exit Codes
const (
	EX_USAGE = 64
)

// General Flags
var (
	flagAddr string
	flagUser string
	flagPass string
)

type Cmd struct {
	Flag flag.FlagSet
	Run  func(cmd *Cmd, args []string)
}

func runList(cmd *Cmd, args []string) {
	fmt.Println("RUNNING LIST")
}

var cmds = map[string]Cmd{
	"ls": {flag.FlagSet{}, runList},
}

func init() {
	flag.StringVar(&flagAddr, "addr", "",
		"address (example: https://example.com/tt-rss/)")
	flag.StringVar(&flagUser, "user", "admin", "user to connect as")
	flag.StringVar(&flagPass, "pass", "password", "password to use")

	for name, cmd := range cmds {
		cmd.Flag.Init(name, flag.PanicOnError)
	}
}

func main() {
	flag.Parse()

	if flag.NArg() < 1 {
		fmt.Fprintf(os.Stderr,
			"%s: error: expected at least 1 argument\n",
			os.Args[0])
		flag.Usage()
		os.Exit(EX_USAGE)
	}

	if !strings.HasPrefix(flagAddr, "http") {
		fmt.Fprintf(os.Stderr,
			"%s: error: address %q must start with \"http\"\n",
			os.Args[0], flagAddr)
		os.Exit(EX_USAGE)
	}

	requestedName := flag.Arg(0)
	var chosenCmd *Cmd
	for name, cmd := range cmds {
		if name == requestedName {
			chosenCmd = &cmd
			break
		}
	}
	if chosenCmd == nil {
		availableCommands := make([]string, len(cmds))
		for name := range cmds {
			availableCommands = append(availableCommands, name)
		}
		sort.Strings(availableCommands)

		fmt.Fprintf(os.Stderr,
			"%s: error: unknown command %q: expected one of %v\n",
			os.Args[0], requestedName, availableCommands)
		os.Exit(EX_USAGE)
	}
	chosenCmd.Run(chosenCmd, flag.Args())
}
