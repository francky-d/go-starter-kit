package testutil

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

// Client calls an http.Handler in memory, the way an API client would.
type Client struct {
	t       *testing.T
	handler http.Handler
	token   string
}

func NewClient(t *testing.T, handler http.Handler) *Client {
	return &Client{t: t, handler: handler}
}

// WithToken returns a copy that sends token as a bearer token.
func (c *Client) WithToken(token string) *Client {
	authenticated := *c
	authenticated.token = token

	return &authenticated
}

func (c *Client) Get(path string) *Response {
	c.t.Helper()
	return c.send(http.MethodGet, path, nil)
}

func (c *Client) Post(path string, body any) *Response {
	c.t.Helper()
	return c.send(http.MethodPost, path, c.encode(body))
}

func (c *Client) send(method, path string, body io.Reader) *Response {
	c.t.Helper()
	request := httptest.NewRequest(method, path, body)
	request.Header.Set("Content-Type", "application/json")
	if c.token != "" {
		request.Header.Set("Authorization", "Bearer "+c.token)
	}

	recorder := httptest.NewRecorder()
	c.handler.ServeHTTP(recorder, request)

	return &Response{t: c.t, ResponseRecorder: recorder}
}

func (c *Client) encode(body any) io.Reader {
	c.t.Helper()
	encoded, err := json.Marshal(body)
	require.NoError(c.t, err)

	return bytes.NewReader(encoded)
}

type Response struct {
	t *testing.T
	*httptest.ResponseRecorder
}

// ErrorBody is the shape of every error response.
type ErrorBody struct {
	Message string              `json:"message"`
	Errors  map[string][]string `json:"errors"`
}

// Data decodes the data member of a success response into target.
func (r *Response) Data(target any) {
	r.t.Helper()
	var envelope struct {
		Data json.RawMessage `json:"data"`
	}
	require.NoError(r.t, json.Unmarshal(r.Body.Bytes(), &envelope), r.Body.String())
	require.NoError(r.t, json.Unmarshal(envelope.Data, target), r.Body.String())
}

func (r *Response) ErrorBody() ErrorBody {
	r.t.Helper()
	var body ErrorBody
	require.NoError(r.t, json.Unmarshal(r.Body.Bytes(), &body), r.Body.String())

	return body
}
