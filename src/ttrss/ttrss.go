// vi: set noet ts=4 sw=4 ft=go tw=79:

package ttrss

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
)

// Status values returned from an API request.
const (
	API_STATUS_OK = iota
	API_STATUS_ERR
)

type Client struct {
	ApiEP     string
	Client    http.Client
	SessionID string
}

// Resp represents the JSON response returned by the TTRSS API.
type Resp struct {
	// Same as request "seq" number, if provided.
	// Otherwise mostly 0, but sometimes null.
	Seq int

	// API_STATUS_* value (hopefully)
	Status int

	// Content["error"] wrapped as an error; nil if not present or not string
	Error error

	// Content of the response.
	Content map[string]interface{}
}

// Call issues an API request.
// If an error status is returned, tt.Error will be set.
// If an HTTP connection error occurs, returns nil and an error.
func (tt *Client) Call(op string, body map[string]interface{}) (resp Resp, err error) {
	body["op"] = op
	if tt.SessionID != "" {
		body["sid"] = tt.SessionID
	}
	fmt.Println("### issuing call:", body)

	buffer, err := AsJSONBuffer(body)
	if err != nil {
		return
	}

	httpResp, err := tt.Client.Post(tt.ApiEP, "application/json", &buffer)
	if err != nil {
		err = fmt.Errorf("connection error: %v\n", err)
		return
	}

	defer httpResp.Body.Close()
	dec := json.NewDecoder(httpResp.Body)
	err = dec.Decode(&resp)
	if err != nil {
		err = fmt.Errorf("API JSON response was malformed: %v - "+
			"are you sure you supplied the correct URL?\n", err)
		return
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

type ConnInfo struct {
	HostURL  string
	User     string
	Password string
}

// Logs into the host as the designated user.
// Updates tt.ApiEP and tt.SessionID if successful.
func (tt *Client) Login(conn ConnInfo) (ok bool, err error) {
	apiEP := conn.HostURL
	if !strings.HasSuffix(apiEP, "/") {
		apiEP += "/"
	}
	apiEP += "api/"
	tt.ApiEP = apiEP
	fmt.Println("### trying to log in as", conn.User)

	loginMap := map[string]interface{}{
		"user":     conn.User,
		"password": conn.Password,
	}
	resp, err := tt.Call("login", loginMap)
	if err != nil {
		return
	}

	sessionID, ok := resp.Content["session_id"]
	if !ok || resp.Status != API_STATUS_OK {
		ok = false
		msg := "error: failed to log in at %s as %s"
		if resp.Error != nil {
			msg += ": " + resp.Error.Error()
		}
		err = fmt.Errorf(msg, apiEP, conn.User)
		return
	}
	tt.SessionID = sessionID.(string)
	fmt.Println("### logged in as", conn.User, "with sessionID", tt.SessionID)
	return
}

type SubscribeStatus int

// Status codes returned by ttrss.Subscribe().
const (
	SUB_ALREADY_ADDED SubscribeStatus = iota
	SUB_ADDED
	SUB_INVALID_URL
	SUB_HTML_NO_FEEDS
	SUB_HTML_MULTIPLE_FEEDS
	SUB_GET_FAILED
	SUB_XML_INVALID
)

func (status SubscribeStatus) String() (text string) {
	switch status {
	case SUB_ALREADY_ADDED:
		text = "already subscribed to feed"
	case SUB_ADDED:
		text = ""
	case SUB_INVALID_URL:
		text = "invalid feed URL"
	case SUB_HTML_NO_FEEDS:
		text = "no feed link found in HTML at URL"
	case SUB_HTML_MULTIPLE_FEEDS:
		text = "multiple feed links found in HTML at URL"
	case SUB_GET_FAILED:
		text = "unable to GET URL"
	case SUB_XML_INVALID:
		text = "invalid XML at URL"
	default:
		fmt.Sprintf("unknown Subscribe status: %d", status)
	}
	return
}

type SubscribeError struct {
	Status SubscribeStatus

	// Error message provided by the API.
	Message string
}

func (err *SubscribeError) Error() (text string) {
	text = fmt.Sprintf("%s: %s", err.Status, err.Message)
	return
}

func (tt *Client) Subscribe(feedURL string, categoryID int, feedUsername string, feedPassword string) (didSubscribe bool, err error) {
	// An auth'd call that contains a feed URL will always "succeed".
	// The actual return value is buried in Content["status"] as a map
	// "code" => int, "message" => string (underlying error).
	subscribeMap := map[string]interface{}{
		"feed_url": feedURL,
		"category_id": categoryID,
	}
	if feedUsername != "" {
		subscribeMap["login"] = feedUsername
		subscribeMap["password"] = feedPassword
	}
	resp, err := tt.Call("subscribeToFeed", subscribeMap)

	if err != nil {
		return
	}

	if resp.Error != nil {
		err = fmt.Errorf("API error: %s", resp.Error)
		return
	}

	subscribeStatus, ok := resp.Content["status"].(map[string]interface{})
	if !ok {
		err = fmt.Errorf("error: no subscription status: have instead %#v",
			resp.Content)
		return
	}

	jsonCode, ok := subscribeStatus["code"].(float64)
	code := SubscribeStatus(jsonCode)
	if tok := SUB_ADDED <= code && code <= SUB_XML_INVALID; !ok || !tok {
		err = fmt.Errorf("Unknown SubscribeStatus: %#v",
			subscribeStatus)
		return
	}

	message, ok := subscribeStatus["message"].(string)
	if !ok {
		message = "(no underlying error returned by API)"
	}

	err = &SubscribeError{code, message}

	didSubscribe = code == SUB_ADDED || code == SUB_ALREADY_ADDED
	return
}

// Returns map converted to JSON as a buffer.
// If an encoding error occurs, buffer will be nil and err will be set.
func AsJSONBuffer(v interface{}) (buffer bytes.Buffer, err error) {
	enc := json.NewEncoder(&buffer)
	err = enc.Encode(v)
	if err != nil {
		err = fmt.Errorf("error encoding JSON: %v - trying to encode %#v\n",
			err, v)
	}
	return
}
