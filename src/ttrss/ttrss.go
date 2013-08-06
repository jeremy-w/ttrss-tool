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
