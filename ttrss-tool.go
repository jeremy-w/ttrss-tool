// vi: set noet ts=4 sw=4 ft=go tw=79:

/*ttrss-tool is a commandline interface for managing your Tiny Tiny RSS
subscriptions.

Its interface is based on standard Unix shell file and directory management:
the category hierarchy acts as directories, and feeds act as files.

Features:

  - Subscribe to a feed by linking it into a category with `ln url catpath`.
  - List categories and feeds using `ls`.
  - Unsubscribe using `rm`.
  - And so on.

Bonus:

  - Includes a nascent ttrss library.
    Get a jumpstart on building your own go-lang TTRSS tool today!

LICENSE: ISC (https://github.com/jeremy-w/ttrss-tool/blob/master/LICENSE)
*/
package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path"
	"sort"
	"strings"
	"ttrss"
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
	flAddr string
	flUser string
	flPass string
	flDotfilePath string
)

// tt is logged in by main() prior to running any command.
var tt ttrss.Client

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

var userDefault = "admin"

func init() {
	const noDefault = ""
	addrHelp := "address (example: https://example.com/tt-rss/)"
	flag.StringVar(&flAddr, "addr", noDefault, addrHelp)
	flag.StringVar(&flAddr, "a", noDefault, addrHelp)

	userHelp := "user to connect as"
	flag.StringVar(&flUser, "user", userDefault, userHelp)
	flag.StringVar(&flUser, "u", userDefault, userHelp)

	passwordHelp := "password to use"
	flag.StringVar(&flPass, "pass", noDefault, passwordHelp)
	flag.StringVar(&flPass, "p", noDefault, passwordHelp)

	dotfileDefault := xdgConfigSearch("ttrss-tool/config", false)
	dotfileHelp :=
		"dotfile path (defaults to $XDG_CONFIG_HOME/ttrss-tool/config"
	flag.StringVar(&flDotfilePath, "dotfile", dotfileDefault, dotfileHelp)

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

	err := applyDotfile(flDotfilePath)
	if err != nil {
		log.Fatal(err.Error())
	}

	if flag.NArg() < 1 {
		fmt.Fprintf(os.Stderr,
			"%s: error: expected at least 1 argument\n",
			os.Args[0])
		flag.Usage()
		os.Exit(EX_USAGE)
	}

	if !strings.HasPrefix(flAddr, "http") {
		fmt.Fprintf(os.Stderr,
			"%s: error: address %q must start with \"http\"\n",
			os.Args[0], flAddr)
		os.Exit(EX_USAGE)
	}

	if flPass == "" {
		flPass, err = readPassword(os.Stdin, os.Stdout)
		if err != nil {
			log.Fatal(err.Error())
		}
	}

	requestedName := flag.Arg(0)
	chosenCmd := cmds[requestedName]
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

	tt.Login(ttrss.ConnInfo{flAddr, flUser, flPass})

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
	fmt.Println("ln feed [catpath] -- subscribe to a new feed")
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

	// An auth'd call that contains a feed URL will always "succeed".
	// The actual return value is buried in Content["status"] as a map
	// "code" => int, "message" => string (underlying error).
	subscribeMap := map[string]interface{} {
		"feed_url": feed,
		//"category_id": catID  // int - defaults to 0 aka Uncategorized
		//"login": ln.flLogin  // if required
		//"password": ln.flPassword // if required
	}
	resp, err := tt.Call("subscribeToFeed", subscribeMap)

	// Subscription status values.
	const (
		SUB_STATUS_ALREADY_ADDED = iota
		SUB_STATUS_ADDED
		SUB_STATUS_INVALID_URL
		SUB_STATUS_HTML_NO_FEEDS
		SUB_STATUS_HTML_MULTIPLE_FEEDS
		SUB_STATUS_GET_FAILED
		SUB_STATUS_XML_INVALID
		SUB_STATUS_COUNT
	)

	die := func(err string) {
		log.Fatalf("error: unable to link feed %s at catpath %s: %s",
			feed, catpath, err)
	}

	if err != nil {
		die(err.Error())
	}

	if resp.Error != nil {
		die(resp.Error.Error())
	}

	subscribeStatus, ok := resp.Content["status"].(map[string]interface{})
	if !ok {
		log.Fatalf("error: no subscription status returned: have instead %#v",
			resp.Content)
	}

	code, ok := subscribeStatus["code"].(float64)
	if !ok || code >= SUB_STATUS_COUNT {
		log.Fatalf("error: unexpected result from API: %#v", subscribeStatus)
	}

	message, ok := subscribeStatus["message"].(string)
	if !ok {
		message = "(no underlying error returned by API)"
	}

	good := code == SUB_STATUS_ALREADY_ADDED || code == SUB_STATUS_ADDED
	text := fmt.Sprintf("???: unknown return code: %d (message: %s)",
		code, message)
	switch code {
	case SUB_STATUS_ALREADY_ADDED:
		text = fmt.Sprintf("warning: already subscribed to [%s]", feed)
	case SUB_STATUS_ADDED:
		text = ""
	case SUB_STATUS_INVALID_URL:
		text = fmt.Sprintf("error: invalid URL [%s]: %s", feed, message)
	case SUB_STATUS_HTML_NO_FEEDS:
		text = fmt.Sprintf("error: no feed link found in HTML of [%s]: %s",
			feed, message)
	case SUB_STATUS_HTML_MULTIPLE_FEEDS:
		text = fmt.Sprintf(
			"error: multiple feed links found in HTML of [%s]: %s",
			feed, message)
	case SUB_STATUS_GET_FAILED:
		text = fmt.Sprintf("error: unable to GET [%s]: %s", feed, message)
	case SUB_STATUS_XML_INVALID:
		text = fmt.Sprintf("error: XML of [%s] is invalid: %s", feed, message)
	default:
		// already set text in this case
	}
	if text != "" {
		fmt.Fprintln(os.Stderr, text)
	}
	if good {
		os.Exit(EX_SUCCESS)
	}
	os.Exit(EX_DATAERR)
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

func xdgConfigSearch(subpath string, onlyIfExists bool) (filePath string) {
	home := os.Getenv("HOME")
	dir := os.Getenv("XDG_CONFIG_HOME")
	if dir == "" { dir = path.Join(home, ".config") }
	dirs := []string{dir}

	dirsString := os.Getenv("XDG_CONFIG_DIRS")
	if dirsString != "" {
		moreDirs := strings.Split(dirsString, ":")
		dirs = append(dirs, moreDirs...)
	}

	fallbackPath := ""
	for _, dir := range dirs {
		if !strings.HasPrefix(dir, "/") { continue }
		if _, err := os.Stat(dir); err != nil {
			continue
		}
		path := path.Join(dir, subpath)
		if fallbackPath == "" {
			fallbackPath = path
		}
		_, err := os.Stat(path)
		if err == nil {
			return path
		}
	}

	if onlyIfExists {
		return ""
	}
	return fallbackPath
}

// Updates global flags based on the dotfile at path.
// If the file does not exist, nothing happens.
// If it does exist, but cannot be read or parsed, the program terminates.
func applyDotfile(path string) (err error) {
	bytes, err := ioutil.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			err = nil
			return
		}

		err = fmt.Errorf("error: unable to read dotfile [%s]: %s", path, err)
		return
	}

	type Config struct {
		Addr string
		User string
		Pass string
	}
	var config Config
	err = json.Unmarshal(bytes, &config)
	if err != nil {
		err = fmt.Errorf("error: unable to parse contents of dotfile [%s]: %s",
			path, err)
		return
	}

	// Only update values that were not set on the command line.
	if flAddr == "" { flAddr = config.Addr }
	if flUser == userDefault { flUser = config.User }
	if flPass == "" { flPass = config.Pass }
	return
}

// Reads a password from r after writing prompt to w.
func readPassword(r io.Reader, w io.Writer) (pass string, err error) {
	scanner := bufio.NewScanner(r)

	for {
		fmt.Fprint(w, "password (will be echoed): ")
		ok := scanner.Scan()
		if !ok {
			msg := "error: failed reading password"
			if err := scanner.Err(); err != nil {
				msg += ": " + err.Error()
			}
			err = fmt.Errorf("%s", msg)
			return
		}
		if pass = scanner.Text(); pass == "" {
			continue
		}
		return
	}
}
