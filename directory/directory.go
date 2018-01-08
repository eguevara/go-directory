package directory

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"reflect"

	"github.com/google/go-querystring/query"
)

const (
	libraryVersion = "1.0.0"
	userAgent      = "go-directory/" + libraryVersion
	mediaType      = "application/json"
)

// Client manages communication with directory V2 API.
type Client struct {
	// HTTP client used to communicate with the directory API.
	client *http.Client

	// Base URL for API requests.
	BaseURL *url.URL

	// User agent for client
	UserAgent string

	// Reuse a single struct instead of allocating one for each service on the heap.
	common service

	// Services used for talking to different parts of the directory API.
	Users UsersService
}

type service struct {
	client *Client
}

// Response is a directory response. This wraps the standard http.Response returned from Directory.
type Response struct {
	*http.Response
}

// An ErrorResponse reports the error caused by an API request
type ErrorResponse struct {

	// HTTP response that caused this error
	Response *http.Response

	// CustomError information from directory api response.
	CustomError
}

// CustomError holds directory error response.
type CustomError struct {
	Code    int    `json:"code,omitempty"`
	Message string `json:"message,omitempty"`
}

func (r *ErrorResponse) Error() string {
	return fmt.Sprintf("%v", r.CustomError.Message)
}

func addOptions(s string, opt interface{}) (string, error) {
	v := reflect.ValueOf(opt)
	if v.Kind() == reflect.Ptr && v.IsNil() {
		return s, nil
	}

	u, err := url.Parse(s)
	if err != nil {
		return s, err
	}

	qs, err := query.Values(opt)
	if err != nil {
		return s, err
	}

	u.RawQuery = qs.Encode()
	return u.String(), nil
}

// NewClient returns a new MM-Directory API client.
func NewClient() *Client {

	httpClient := http.DefaultClient

	c := &Client{client: httpClient, UserAgent: userAgent}
	c.common.client = c
	c.Users = (*UsersServiceOp)(&c.common)

	return c
}

// ClientOpt are options for New.
type ClientOpt func(*Client) error

// New returns a new mm-directory API client instance.
func New(opts ...ClientOpt) (*Client, error) {
	c := NewClient()
	for _, opt := range opts {
		if err := opt(c); err != nil {
			return nil, err
		}
	}

	if c.BaseURL == nil {
		return nil, errors.New("you need to set the baseURL with the SetBaseURL helper")
	}

	return c, nil
}

// SetBaseURL is a client option for setting the base URL.
func SetBaseURL(bu string) ClientOpt {
	return func(c *Client) error {
		u, err := url.Parse(bu)
		if err != nil {
			return err
		}

		c.BaseURL = u
		return nil
	}
}

// SetHTTPClient makes the directory client use the given HTTP client.
func SetHTTPClient(client *http.Client) ClientOpt {
	return func(c *Client) error {
		c.client = client
		return nil
	}
}

// SetUserAgent is a client option for setting the user agent.
func SetUserAgent(ua string) ClientOpt {
	return func(c *Client) error {
		c.UserAgent = fmt.Sprintf("%s+%s", ua, c.UserAgent)
		return nil
	}
}

// NewRequest creates an API request. A relative URL can be provided in urlStr, which will be resolved to the
// BaseURL of the Client. Relative URLS should always be specified without a preceding slash. If specified, the
// value pointed to by body is JSON encoded and included in as the request body.
func (c *Client) NewRequest(method, urlStr string, body interface{}) (*http.Request, error) {
	rel, err := url.Parse(urlStr)
	if err != nil {
		return nil, err
	}

	u := c.BaseURL.ResolveReference(rel)

	var buf io.ReadWriter
	if body != nil {
		buf = new(bytes.Buffer)
		err := json.NewEncoder(buf).Encode(body)
		if err != nil {
			return nil, err
		}
	}

	req, err := http.NewRequest(method, u.String(), buf)
	if err != nil {
		return nil, err
	}

	req.Header.Add("Content-Type", mediaType)
	req.Header.Add("Accept", mediaType)
	req.Header.Add("User-Agent", c.UserAgent)

	// out, err := httputil.DumpRequestOut(req, true)
	// if err != nil {
	// 	log.Fatal(err)
	// }
	// fmt.Println(strings.Replace(string(out), "\r", "", -1))
	// fmt.Println("--newRequest--")

	return req, nil
}

// newResponse creates a new Response for the provided http.Response
func newResponse(r *http.Response) *Response {
	response := Response{Response: r}

	return &response
}

// Do sends an API request and returns the API response. The API response is JSON decoded and stored in the value
// pointed to by v, or returned as an error if an API error has occurred. If v implements the io.Writer interface,
// the raw response will be written to v, without attempting to decode it.
func (c *Client) Do(ctx context.Context, req *http.Request, v interface{}) (*Response, error) {
	req = req.WithContext(ctx)

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() {
		if rerr := resp.Body.Close(); err == nil {
			err = rerr
		}
	}()

	response := newResponse(resp)

	// outResp, err := httputil.DumpResponse(resp, true)
	// if err != nil {
	// 	log.Fatal(err)
	// }
	// fmt.Println(strings.Replace(string(outResp), "\r", "", -1))
	// fmt.Println("-Do---")

	err = CheckResponse(resp)
	if err != nil {
		return response, err
	}

	if v != nil {
		if w, ok := v.(io.Writer); ok {
			_, err := io.Copy(w, resp.Body)
			if err != nil {
				return nil, err
			}
		} else {
			err := json.NewDecoder(resp.Body).Decode(v)
			if err != io.EOF {
				err = nil // ignore EOF
			}
		}
	}

	return response, err
}

// CheckResponse checks the API response for errors, and returns them if present. A response is considered an
// error if it has a status code outside the 200 range. API error responses are expected to have either no response
// body, or a JSON response body that maps to ErrorResponse. Any other response body will be silently ignored.
func CheckResponse(r *http.Response) error {
	if c := r.StatusCode; c >= 200 && c <= 299 {
		return nil
	}

	errorResponse := &ErrorResponse{Response: r}
	data, err := ioutil.ReadAll(r.Body)
	if err == nil && len(data) > 0 {
		err := json.Unmarshal(data, errorResponse)
		if err != nil {
			return err
		}
	}

	return errorResponse
}
