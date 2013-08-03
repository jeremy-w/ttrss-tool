// vi: set noet ts=4 sw=4 ft=go tw=79:
/* ttrss-tool manipulates tiny-tiny-rss subscriptions. */
package main

import (
	"flag"
	"fmt"
	"io"
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

// Cmd is how main interacts with the subcommands.
type Cmd interface {
	// Init is used to configure any flags.
	// It is called during program init().
	Init()

	// Synopsis should print two spaces, the command name, a one-line
	// description, and a final newline.
	Synopsis(w io.Writer)

	// Run is called by main() on the chosen subcommand.
	// args contains the arguments to the subcommand (not including the
	// subcommand name).
	Run(args []string)
}

var cmds = map[string]Cmd{
	"ls": &Ls{},
}

func init() {
	flag.StringVar(&flagAddr, "addr", "",
		"address (example: https://example.com/tt-rss/)")
	flag.StringVar(&flagUser, "user", "admin", "user to connect as")
	flag.StringVar(&flagPass, "pass", "password", "password to use")

	for _, cmd := range cmds {
		cmd.Init()
	}

	flag.Usage = func() {
		name := os.Args[0]
		w := os.Stderr

		fmt.Fprintln(w,
			"Usage of ", name, ": ", name, "flags subcommand subflags subargs")
		flag.PrintDefaults()
		fmt.Fprintln(w, "Subcommands:")
		for _, cmd := range cmds {
			fmt.Fprint(w, "  ")
			cmd.Synopsis(w)
		}
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
	var chosenCmd Cmd
	for name, cmd := range cmds {
		if name == requestedName {
			chosenCmd = cmd
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
	chosenCmd.Run(flag.Args()[1:])
}

func FlagSetPrintUsage(fl flag.FlagSet, w io.Writer, progname string) {
	fmt.Fprintln(w, "Usage of ", progname, ":")
	fl.SetOutput(w)
	fl.PrintDefaults()
}

type Ls struct {
	flHelp bool
	flRecurse bool
	flags flag.FlagSet
}

func (ls *Ls) Init() {
	ls.flags.Init("ls", flag.PanicOnError)

	ls.flags.BoolVar(&ls.flHelp, "h", false, "help")
	ls.flags.BoolVar(&ls.flHelp, "help", false, "help")

	recurseUsage := "recurse into categories"
	ls.flags.BoolVar(&ls.flRecurse, "R", false, recurseUsage)
	ls.flags.BoolVar(&ls.flRecurse, "Recurse", false, recurseUsage)
}

func (ls *Ls) Synopsis(w io.Writer) {
	fmt.Fprintln(w, "ls -- list categories and feeds")
}

func (ls *Ls) Run(args []string) {
	_ = ls.flags.Parse(args)
	if ls.flHelp {
		FlagSetPrintUsage(ls.flags, os.Stdout, "ls")
		return
	}
	fmt.Println("RUNNING LIST:", ls.flRecurse, ls.flags.Args())
}
