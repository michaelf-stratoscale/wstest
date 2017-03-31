package wstest

import (
	"bufio"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"

	"github.com/gorilla/websocket"
)

// Client is a websocket client for unit testing
type Client struct {
	httptest.ResponseRecorder
	sConn    net.Conn
	cConn    net.Conn
	wsConn   *websocket.Conn
	hijacked bool
}

// NewClient returns a new client
func NewClient() *Client {
	sConn, cConn := newConnPair()
	return &Client{
		sConn: sConn,
		cConn: cConn,
	}
}

// Connect a wstest Client to an http.Handler which accepts websocket upgrades.
// This send an HTTP request to the http.Handler, and wait for the connection upgrade response.
// it uses the gorilla's websocket.Dial function, over a fake net.Conn struct.
// it runs the server's ServeHTTP function in a goroutine, so server can communicate with a
// client running on the current program flow
func (c *Client) Connect(h http.Handler) error {

	// run the runServer in a goroutine, so when the Dial send the request to
	// the server on the connection, it will be parsed as an HTTPRequest and
	// sent to the Handler function.
	go c.runServer(h)

	// use the websocket.Dialer.Dial with the fake net.Conn to communicate with
	// the server
	dialer := websocket.Dialer{NetDial: func(network, addr string) (net.Conn, error) { return c.cConn, nil }}
	wsConn, _, err := dialer.Dial("ws://ws", nil)
	if err != nil {
		return err
	}
	c.wsConn = wsConn
	return nil
}

// dialer handler reads the request sent on the connection to the server
// from the websocket.Dialer.Dial function, and pass it to the server.
// once this is done, the communication is done on the wsConn
func (c *Client) runServer(h http.Handler) {
	req, err := http.ReadRequest(bufio.NewReader(c.sConn))
	if err != nil {
		panic(err)
	}
	h.ServeHTTP(c, req)
}

// Receive a message from the websocket server
func (c *Client) Receive() (*Message, error) {
	mType, data, err := c.wsConn.ReadMessage()
	if err != nil {
		return nil, err
	}
	return &Message{Type: mType, Data: data}, nil
}

// Send a message to the websocket server
func (c *Client) Send(m *Message) error {
	return c.wsConn.WriteMessage(m.Type, m.Data)
}

// Close the connection
func (c *Client) Close() error {
	return c.wsConn.Close()
}

// Hijack the connection
func (c *Client) Hijack() (net.Conn, *bufio.ReadWriter, error) {

	rw := bufio.NewReadWriter(bufio.NewReader(c.sConn), bufio.NewWriter(c.sConn))

	if c.hijacked {
		return nil, nil, errors.New("already hijacked")
	}
	c.hijacked = true
	return c.sConn, rw, nil
}