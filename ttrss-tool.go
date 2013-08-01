// vi: set noet ts=4 sw=4 ft=go tw=79:
/* ttrss-tool manipulates tiny-tiny-rss subscriptions. */
package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
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
	Run func(cmd *Cmd, args []string)
}

func runList(cmd *Cmd, args []string) {
	fmt.Println("RUNNING LIST")
}

var cmds = map[string]Cmd {
	"ls": {flag.FlagSet{}, runList},
}

func init() {
	flag.StringVar(&flagAddr, "addr", "",
		"address (example: https://example.com/tt-rss/)")
	flag.StringVar(&flagUser, "user", "admin", "user to connect as")
	flag.StringVar(&flagPass, "pass", "password", "password to use")

	if !strings.HasPrefix(flagAddr, "http") {
		fmt.Fprintf(os.Stderr,
		"%s: error: address %q must start with \"http\"\n",
			os.Args[0], flagAddr)
		os.Exit(EX_USAGE)
	}

	for k, v := range cmds {
		v.Flag.Init(k, flag.PanicOnError)
	}
}

func main() {
	flag.Parse()

	if flag.NArg() < 1 {
		flag.Usage()
		os.Exit(EX_USAGE)
	}

	fmt.Println(flagAddr, flagUser, flagPass, flag.Args())
}
