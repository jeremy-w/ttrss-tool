// vi: set noet ts=4 sw=4 ft=go tw=79:
/* ttrss-tool manipulates tiny-tiny-rss subscriptions. */
package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"sort"
	"strings"
)

// Status values returned from an API request.
const (
	API_STATUS_OK = iota
	API_STATUS_ERR
)

// Exit Codes
const (
	EX_SUCCESS = 0
	EX_USAGE = 64
	EX_DATAERR = 65
	EX_PROTOCOL = 76
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

	// Synopsis should print a message like "cmd -- does blah".
	// The printed text should end with a newline.
	Synopsis(w io.Writer)

	// Run is called by main() on the chosen subcommand.
	// args contains the arguments to the subcommand (not including the
	// subcommand name).
	Run(args []string)
}

var cmds = map[string]Cmd{
	"ln": &Ln{},
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

		fmt.Fprintf(w,
			"Usage of %s: %s flags subcommand subflags subargs\n", name, name)
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
	fmt.Fprintf(w, "Usage of %s:\n", progname)
	fl.SetOutput(w)
	fl.PrintDefaults()
}

type Ln struct {
	flHelp bool
	flags flag.FlagSet
}

func (ln *Ln) Init() {
	ln.flags.Init("ln", flag.PanicOnError)

	ln.flags.BoolVar(&ln.flHelp, "h", false, "help")
	ln.flags.BoolVar(&ln.flHelp, "help", false, "help")
}

func (ln *Ln) Synopsis(w io.Writer) {
	fmt.Println("ln feed [catpath] -- subscribes to a new feed")
}

func (ln *Ln) Run(args []string) {
	ln.flags.Parse(args)

	if ln.flHelp {
		FlagSetPrintUsage(ln.flags, os.Stdout, "ln")
		os.Exit(EX_SUCCESS)
	}

	argc := ln.flags.NArg()
	if argc < 1 {
		FlagSetPrintUsage(ln.flags, os.Stderr, "ln")
		os.Exit(EX_USAGE)
	}

	feed := ln.flags.Arg(0)
	catpath := ln.flags.Arg(1)

	loginMap := map[string]string {
		"op": "login",
		"user": flagUser,
		"password": flagPass,
	}

	var loginBuffer bytes.Buffer
	enc := json.NewEncoder(&loginBuffer)
	err := enc.Encode(loginMap)
	if err != nil {
		fmt.Fprintf(os.Stderr, "encoding error: %v\n", err)
		os.Exit(EX_DATAERR)
	}

	apiEP := flagAddr
	if !strings.HasSuffix(apiEP, "/") {
		apiEP += "/"
	}
	apiEP += "api/"

	var client http.Client
	httpResp, err := client.Post(apiEP, "application/json", &loginBuffer)
	if err != nil {
		fmt.Fprintf(os.Stderr, "connection error: %v\n", err)
		os.Exit(EX_DATAERR)
	}

	defer httpResp.Body.Close()
	type TTRSSResp struct {
		Seq int
		Status int
		Content map[string]interface{}
	}
	var resp TTRSSResp
	dec := json.NewDecoder(httpResp.Body)
	err = dec.Decode(&resp)
	if err != nil {
		fmt.Fprintf(os.Stderr, "bad API response: %v\n", err)
		os.Exit(EX_PROTOCOL)
	}
	sessionID, ok := resp.Content["sessionID"]
	if !ok || resp.Status != API_STATUS_OK {
		log.Fatalln("error: login failed")
	}
	fmt.Println("sessionID", sessionID, feed, catpath)
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
	fmt.Fprintln(w, "ls [-R] [catpath...] -- list categories and feeds")
}

func (ls *Ls) Run(args []string) {
	_ = ls.flags.Parse(args)
	if ls.flHelp {
		FlagSetPrintUsage(ls.flags, os.Stdout, "ls")
		return
	}
	fmt.Println("RUNNING LIST:", ls.flRecurse, ls.flags.Args())
}
