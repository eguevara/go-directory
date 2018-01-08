package directory

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"strings"
	"testing"
)

var (
	mux    *http.ServeMux
	ctx    = context.TODO()
	client *Client
	server *httptest.Server
)

func setup() {
	mux = http.NewServeMux()
	server = httptest.NewServer(mux)

	client = NewClient()
	url, _ := url.Parse(server.URL)
	client.BaseURL = url
}

func teardown() {
	server.Close()
}

func testMethod(t *testing.T, r *http.Request, expected string) {
	if expected != r.Method {
		t.Errorf("Request method = %v, expected %v", r.Method, expected)
	}
}

type values map[string]string

func testFormValues(t *testing.T, r *http.Request, values values) {
	expected := url.Values{}
	for k, v := range values {
		expected.Add(k, v)
	}

	err := r.ParseForm()
	if err != nil {
		t.Fatalf("parseForm(): %v", err)
	}

	if !reflect.DeepEqual(expected, r.Form) {
		t.Errorf("Request parameters = %v, expected %v", r.Form, expected)
	}
}

func testURLParseError(t *testing.T, err error) {
	if err == nil {
		t.Errorf("Expected error to be returned")
	}
	if err, ok := err.(*url.Error); !ok || err.Op != "parse" {
		t.Errorf("Expected URL parse error, got %+v", err)
	}
}

func testClientDefaultUserAgent(t *testing.T, c *Client) {
	if c.UserAgent != userAgent {
		t.Errorf("NewClick UserAgent = %v, expected %v", c.UserAgent, userAgent)
	}
}

func testClientDefaults(t *testing.T, c *Client) {
	testClientDefaultUserAgent(t, c)
}

func TestNewClient(t *testing.T) {
	c := NewClient()
	testClientDefaults(t, c)
}

func TestNew(t *testing.T) {
	c, err := New(SetBaseURL("htt://localhost/"))

	if err != nil {
		t.Fatalf("New(): %v", err)
	}
	testClientDefaults(t, c)
}

func TestNewRequest_badURL(t *testing.T) {
	c, err := New(SetBaseURL("http://localhost/"))
	_, err = c.NewRequest("GET", ":", nil)
	testURLParseError(t, err)
}

func TestDo(t *testing.T) {
	setup()
	defer teardown()

	type foo struct {
		A string
	}

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if m := "GET"; m != r.Method {
			t.Errorf("Request method = %v, expected %v", r.Method, m)
		}
		fmt.Fprint(w, `{"A":"a"}`)
	})

	req, _ := client.NewRequest("GET", "/", nil)
	body := new(foo)
	_, err := client.Do(context.Background(), req, body)
	if err != nil {
		t.Fatalf("Do(): %v", err)
	}

	expected := &foo{"a"}
	if !reflect.DeepEqual(body, expected) {
		t.Errorf("Response body = %v, expected %v", body, expected)
	}
}

func TestDo_httpError(t *testing.T) {
	setup()
	defer teardown()

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "Bad Request", 400)
	})

	req, _ := client.NewRequest("GET", "/", nil)
	_, err := client.Do(context.Background(), req, nil)

	if err == nil {
		t.Error("Expected HTTP 400 error.")
	}
}

// Test handling of an error caused by the internal http client's Do()
// function.
func TestDo_redirectLoop(t *testing.T) {
	setup()
	defer teardown()

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/", http.StatusFound)
	})

	req, _ := client.NewRequest("GET", "/", nil)
	_, err := client.Do(context.Background(), req, nil)

	if err == nil {
		t.Error("Expected error to be returned.")
	}
	if err, ok := err.(*url.Error); !ok {
		t.Errorf("Expected a URL error; got %#v.", err)
	}
}

func TestDo_noContent(t *testing.T) {
	setup()
	defer teardown()

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	var body json.RawMessage

	req, _ := client.NewRequest("GET", "/", nil)
	_, err := client.Do(context.Background(), req, &body)

	// Check that empty resp.Body does not return error.
	if err != nil {
		t.Fatalf("Do returned unexpected error: %v", err)
	}
}

func TestCheckResponse(t *testing.T) {
	res := &http.Response{
		Request:    &http.Request{},
		StatusCode: http.StatusBadRequest,
		Body: ioutil.NopCloser(strings.NewReader(`{"message":"",
			"errors": [{"resource": "r", "field": "f", "code": "c"}]}`)),
	}
	err := CheckResponse(res).(*ErrorResponse)

	if err == nil {
		t.Fatalf("Expected error response.")
	}

	expected := &ErrorResponse{
		Response: res,
	}
	if !reflect.DeepEqual(err, expected) {
		t.Errorf("Error = %#v, expected %#v", err, expected)
	}
}

// ensure that we properly handle API errors that do not contain a response
// body
func TestCheckResponse_noBody(t *testing.T) {
	res := &http.Response{
		Request:    &http.Request{},
		StatusCode: http.StatusBadRequest,
		Body:       ioutil.NopCloser(strings.NewReader("")),
	}
	err := CheckResponse(res).(*ErrorResponse)

	if err == nil {
		t.Errorf("Expected error response.")
	}

	expected := &ErrorResponse{
		Response: res,
	}
	if !reflect.DeepEqual(err, expected) {
		t.Errorf("Error = %#v, expected %#v", err, expected)
	}
}

func TestNewRequest(t *testing.T) {
	base := "http://fo.com/"
	c, _ := New(SetBaseURL(base))

	inURL, outURL := "foo", base+"foo"
	inBody, outBody := &User{CoreID: "c", FullName: "f", Status: "s", ID: "i"}, `{"coreId":"c","fullName":"f","status":"s","id":"i"}`+"\n"

	req, _ := c.NewRequest("GET", inURL, inBody)

	// test that body was JSON encoded
	body, _ := ioutil.ReadAll(req.Body)
	if got, want := string(body), outBody; got != want {
		t.Errorf("NewRequest(%q) Body is -%v-, want %v", inBody, got, want)
	}

	// test that relative URL was expanded
	if got, want := req.URL.String(), outURL; got != want {
		t.Errorf("NewRequest(%q) URL is %v, want %v", inURL, got, want)
	}
}

func TestNewRequest_withCustomUserAgent(t *testing.T) {
	ua := "testing"
	c, err := New(SetUserAgent(ua), SetBaseURL("http://localhost/"))

	if err != nil {
		t.Fatalf("New() unexpected error: %v", err)
	}

	req, err := c.NewRequest("GET", "/foo", nil)
	if err != nil {
		t.Fatalf("New() unexpected error: %v", err)
	}

	expected := fmt.Sprintf("%s+%s", ua, userAgent)
	if got := req.Header.Get("User-Agent"); got != expected {
		t.Errorf("New() UserAgent = %s; expected %s", got, expected)
	}
}
func TestNewRequest_withBaseURL(t *testing.T) {

	base := "http://localhost/foo"
	c, err := New(SetBaseURL(base))

	if err != nil {
		t.Fatalf("New() unexpected error %v", err)
	}

	req, _ := c.NewRequest("GET", "/foo", nil)
	expected := fmt.Sprintf("%s", base)

	if got := req.URL.String(); got != expected {
		t.Errorf("New() BaseURL = %s; expected %s", got, expected)
	}

}
func TestNewRequest_withBaseURL_badURL(t *testing.T) {

	base := ":"
	_, err := New(SetBaseURL(base))

	testURLParseError(t, err)
}

func TestErrorResponse_error(t *testing.T) {
	res := &http.Response{Request: &http.Request{}}
	customError := CustomError{Code: 0, Message: "m"}

	err := ErrorResponse{
		Response:    res,
		CustomError: customError,
	}
	if err.Error() == "" {
		t.Errorf("Expected non-empty ErrorResponse.Error()")
	}
}
func TestNewRequest_emptyBody(t *testing.T) {
	c, _ := New(SetBaseURL("http://localhost/"))
	req, err := c.NewRequest("GET", "/", nil)
	if err != nil {
		t.Fatalf("NewRequest returned unexpected error: %v", err)
	}
	if req.Body != nil {
		t.Fatalf("constructed request contains a non-nil Body")
	}
}
