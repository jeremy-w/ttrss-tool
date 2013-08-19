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
	"path/filepath"
	"sort"
	"strings"
	"ttrss"
)

// Exit Codes
const (
	EX_SUCCESS  = 0
	EX_USAGE    = 64
	EX_DATAERR  = 65
	EX_PROTOCOL = 76
)

// General Flags
var (
	flAddr        string
	flUser        string
	flPass        string
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

func flagSetPrintUsage(fl flag.FlagSet, w io.Writer, progname string) {
	fmt.Fprintf(w, "Usage of %s:\n", progname)
	fl.SetOutput(w)
	fl.PrintDefaults()
}

type Ln struct {
	flHelp bool
	flags  flag.FlagSet
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
		flagSetPrintUsage(ln.flags, os.Stdout, "ln")
		os.Exit(EX_SUCCESS)
	}

	argc := ln.flags.NArg()
	if argc < 1 {
		flagSetPrintUsage(ln.flags, os.Stderr, "ln")
		os.Exit(EX_USAGE)
	}

	feed := ln.flags.Arg(0)
	catpath := ln.flags.Arg(1)
	item, err := ResolveCatPath(catpath)
	if err != nil {
		log.Fatalln(err)
	}

	if item.Type != ttrss.Category {
		log.Fatalln("error: not a category:", catpath)
	}

	subscribed, err := tt.Subscribe(feed, item.ID, "", "")

	if s, ok := err.(*ttrss.SubscribeError); ok {
		if (s.Status != ttrss.SUB_ADDED) {
			fmt.Fprintln(os.Stderr, s.Message)
		}
	}

	if subscribed {
		os.Exit(EX_SUCCESS)
	}
	os.Exit(EX_DATAERR)
}

type Ls struct {
	flHelp    bool
	flRecurse bool
	flags     flag.FlagSet
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
	fmt.Println("### parsing `ls` args")
	_ = ls.flags.Parse(args)
	if ls.flHelp {
		flagSetPrintUsage(ls.flags, os.Stdout, "ls")
		return
	}
	fmt.Printf("### parsed: %#v\n", ls)

	catpath := "/"
	if len(args) > 0 {
		catpath = args[0]
	}

	root, err := ResolveCatPath(catpath)
	if err != nil {
		log.Fatalf("unable to list %q: %v", catpath, err)
	}

	for _, item := range root.Items {
		fmt.Println(item.Name)
	}
}

func xdgConfigSearch(subpath string, onlyIfExists bool) (filePath string) {
	home := os.Getenv("HOME")
	dir := os.Getenv("XDG_CONFIG_HOME")
	if dir == "" {
		dir = path.Join(home, ".config")
	}
	dirs := []string{dir}

	dirsString := os.Getenv("XDG_CONFIG_DIRS")
	if dirsString != "" {
		moreDirs := strings.Split(dirsString, ":")
		dirs = append(dirs, moreDirs...)
	}

	fallbackPath := ""
	for _, dir := range dirs {
		if !strings.HasPrefix(dir, "/") {
			continue
		}
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
	if flAddr == "" {
		flAddr = config.Addr
	}
	if flUser == userDefault {
		flUser = config.User
	}
	if flPass == "" {
		flPass = config.Pass
	}
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

func PathComponents(path string) (parts []string) {
	// Trim initial slash; "/" is treated the same as "".
	if strings.HasPrefix(path, "/") {
		path = path[1:]
	}

	// Split into rough parts. This does NOT respect backslash escapes.
	roughParts := strings.Split(path, "/")
	fmt.Println(path, "=> roughly", roughParts)

	// Now clean up rough parts to get the various levels.
	partial := ""
	for i := 0 ; i < len(roughParts)+1; i++ {
		if i < len(roughParts) {
			part := roughParts[i]
			if strings.HasSuffix(part, "\\") {
				partial += part[:len(part) - 1]
				partial += "/"
				continue
			}
			// No escape, so this is the end of a part.
			partial += part
		}
		if partial != "" {
			parts = append(parts, partial)
			partial = ""
		}
	}
	fmt.Println(path, "=>", parts)
	return
}

type catPathResult struct {
	item *ttrss.FeedTreeItem
}

func (err *catPathResult) Error() string {
	return ""
}

func ResolveCatPath(catpath string) (item *ttrss.FeedTreeItem, err error) {
	fmt.Println("### resolving", catpath)
	parts := PathComponents(catpath)
	tree, err := tt.GetFeedTree(true)
	if err != nil {
		return
	}

	walkParts := parts

	/* Gradually eat walkParts till there are none left.
	 * At that point, we've reached our category. */
	walkFn := func(item *ttrss.FeedTreeItem) error {
		fmt.Println("walk:", item.Name, item.Type, item.ID, "-", walkParts)
		isCat := item.Type == ttrss.Category
		if len(walkParts) == 0 {
			return &catPathResult{item}
		}

		if item.Name == walkParts[0] {
			walkParts = walkParts[1:len(walkParts)-1]
			return nil
		}

		if isCat {
			return filepath.SkipDir
		}
		return nil
	}

	err = ttrss.WalkFeedTree(&tree, walkFn)
	result, ok := err.(*catPathResult)
	if ok {
		item = result.item
		err = nil
	}
	err = fmt.Errorf("not found: %q", catpath)
	return
}
