// vi: set noet ts=4 sw=4 ft=go tw=79:
/* ttrss-tool manipulates tiny-tiny-rss subscriptions. */
package main

import (
	"bytes"
	"errors"
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
	flAddr string
	flUser string
	flPass string
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
	const noDefault = ""
	addrHelp := "address (example: https://example.com/tt-rss/)"
	flag.StringVar(&flAddr, "addr", noDefault, addrHelp)
	flag.StringVar(&flAddr, "a", noDefault, addrHelp)

	userDefault := "admin"
	userHelp := "user to connect as"
	flag.StringVar(&flUser, "user", userDefault, userHelp)
	flag.StringVar(&flUser, "u", userDefault, userHelp)

	passwordHelp := "password to use"
	flag.StringVar(&flPass, "pass", noDefault, passwordHelp)
	flag.StringVar(&flPass, "p", noDefault, passwordHelp)

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

	if !strings.HasPrefix(flAddr, "http") {
		fmt.Fprintf(os.Stderr,
			"%s: error: address %q must start with \"http\"\n",
			os.Args[0], flAddr)
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

type TTRSSClient struct {
	ApiEP string
	Client http.Client
	SessionID string
}

// TTRSSResp represents the JSON response returned by the TTRSS API.
type TTRSSResp struct {
	// Same as request "seq" number, if provided.
	// Otherwise mostly 0, but sometimes null.
	Seq int

	// TTRSS_API_STATUS_* value (hopefully)
	Status int

	// Content["error"] wrapped as an error; nil if not present or not string
	Error error

	// Content of the response.
	Content map[string]interface{}
}

// Call issues an API request.
// If an error status is returned, tt.Error will be set.
// If an HTTP connection error occurs, the application terminates with an
// error message, and the call does not return.
func (tt *TTRSSClient) Call(op string, body map[string]interface{}) (resp TTRSSResp) {
	body["op"] = op
	if tt.SessionID != "" {
		body["sid"] = tt.SessionID
	}
	fmt.Println("issuing call:", body)

	buffer := asJSONBuffer(body)
	httpResp, err := tt.Client.Post(tt.ApiEP, "application/json", &buffer)
	if err != nil {
		fmt.Fprintf(os.Stderr, "connection error: %v\n", err)
		os.Exit(EX_DATAERR)
	}

	defer httpResp.Body.Close()
	dec := json.NewDecoder(httpResp.Body)
	err = dec.Decode(&resp)
	if err != nil {
		fmt.Fprintf(os.Stderr,
			"API JSON response was malformed: %v - "+
			"are you sure you supplied the correct URL?\n", err)
		os.Exit(EX_PROTOCOL)
	}

	resp.Error = nil
	if apiError, ok := resp.Content["error"]; ok {
		if errorString, ok := apiError.(string); ok {
			resp.Error = errors.New(errorString)
		}
	}
	if resp.Status != API_STATUS_OK && resp.Error == nil {
		resp.Error = errors.New("(response contained no error text)")
	}
	return
}

// Logs into the host as the designated user.
// Terminates the program with an error message if login fails.
// Otherwise, updates tt.ApiEP and tt.SessionID.
func (tt *TTRSSClient) Login(hostURL string, user string, password string) {
	apiEP := hostURL
	if !strings.HasSuffix(apiEP, "/") {
		apiEP += "/"
	}
	apiEP += "api/"
	tt.ApiEP = apiEP
	fmt.Println("trying to log in as", user)

	loginMap := map[string]interface{} {
		"user": user,
		"password": password,
	}
	resp := tt.Call("login", loginMap)

	sessionID, ok := resp.Content["session_id"]
	if !ok || resp.Status != API_STATUS_OK {
		msg := "error: failed to log in at %s as %s"
		if resp.Error != nil {
			msg += ": " + resp.Error.Error()
		}
		log.Fatalf(msg, apiEP, flUser)
	}
	tt.SessionID = sessionID.(string)
	fmt.Println("logged in as", user, "with sessionID", tt.SessionID)
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

	var tt TTRSSClient
	tt.Login(flAddr, flUser, flPass)

	// An auth'd call that contains a feed URL will always "succeed".
	// The actual return value is buried in Content["status"] as a map
	// "code" => int, "message" => string (underlying error).
	subscribeMap := map[string]interface{} {
		"feed_url": feed,
		//"category_id": catID  // int - defaults to 0 aka Uncategorized
		//"login": ln.flLogin  // if required
		//"password": ln.flPassword // if required
	}
	resp := tt.Call("subscribeToFeed", subscribeMap)

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

// Returns map converted to JSON as a buffer.
// If an encoding error occurs, logs to stderr and exits with EX_DATAERR.
func asJSONBuffer(v interface{}) (buffer bytes.Buffer) {
	enc := json.NewEncoder(&buffer)
	err := enc.Encode(v)
	if err != nil {
		fmt.Fprintf(os.Stderr, "encoding error: %v\n", err)
		os.Exit(EX_DATAERR)
	}
	return
}
