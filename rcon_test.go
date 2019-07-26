package rcon

import (
	"bufio"
	"fmt"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

const testPassword = "asdf"

type RconServer struct {
	l       net.Listener
	events  []Event
	current int
}

type Event struct {
	read    bool
	message string
}

func (m *RconServer) ExpectRead(message string) {
	m.events = append(m.events, Event{true, message})
}

func (m *RconServer) AddWrite(message string) {
	m.events = append(m.events, Event{false, message})
}

func (m *RconServer) PlayOut(conn net.Conn) {
	if m.current >= len(m.events) {
		return
	}

	for !m.events[m.current].read {
		fmt.Fprintf(conn, "%s\n", m.events[m.current].message)
		m.current++
		if m.current >= len(m.events) {
			return
		}
	}
}

func (m *RconServer) ExpectationsMet(t *testing.T) {
	if m.current != len(m.events) {
		for i := m.current; i < len(m.events); i++ {
			if m.events[i].read {
				t.Errorf("expected %s to be read, it was not", m.events[i].message)
			} else {
				t.Errorf("expected %s to be written, it was not", m.events[i].message)
			}
		}
	}
	m.l.Close()
}

func (m *RconServer) Listen(t *testing.T, address string, password string) {
	l, err := net.Listen("tcp", address)
	assert.Nil(t, err)

	m.l = l
	m.events = make([]Event, 0)

	// These are default expectations needed for the implementation
	m.ExpectRead(password)
	m.ExpectRead("tcpr('hello')")
	m.AddWrite("hello")

	defer l.Close()

	for {
		conn, err := l.Accept()
		assert.Nil(t, err)

		defer conn.Close()

		reader := bufio.NewReader(conn)
		for {
			message, err := reader.ReadString('\n')
			assert.Nil(t, err)

			if m.current >= len(m.events) {
				t.Errorf("read another message %s when nothing more was expected", message)
				return
			}

			message = strings.TrimRight(message, "\n")

			if m.events[m.current].read {
				assert.Equal(t, m.events[m.current].message, message)
				m.current++
			}

			m.PlayOut(conn)
		}

	}
}

func TestConnect(t *testing.T) {
	testMessage := "test"
	testMessage2 := "something"

	// Setup the mock rcon server
	server := RconServer{}
	go server.Listen(t, ":1337", testPassword)
	time.Sleep(50 * time.Millisecond)

	server.ExpectRead(testMessage)
	server.AddWrite(testMessage2)

	// Create the rcon server and test it
	client, err := DialRcon("localhost:1337", testPassword, 1*time.Second)
	assert.Nil(t, err)

	err = client.Write(testMessage)
	assert.Nil(t, err)

	actual, err := client.Read()
	assert.Equal(t, testMessage2, actual)

	// check if all the expectations were met
	server.ExpectationsMet(t)
}

func TestHandler(t *testing.T) {
	// Setup the mock rcon server
	server := RconServer{}
	go server.Listen(t, ":1337", testPassword)
	time.Sleep(50 * time.Millisecond)

	server.AddWrite("foo")
	server.ExpectRead("bar")

	server.AddWrite("baz qux")

	// Create the rcon server and test it
	client, err := DialRcon("localhost:1337", testPassword, 1*time.Second)
	assert.Nil(t, err)

	client.HandleFunc("foo", func(m Message, c *Client) error {
		c.Write("bar")
		return nil
	})

	bazOut := make(chan string)
	client.HandleFunc("baz (?P<text>.*)", func(m Message, c *Client) error {
		bazOut <- m.Args["text"]
		return nil
	})

	go func() {
		err := client.Handle()
		assert.Nil(t, err)
	}()

	actual := <-bazOut
	assert.Equal(t, "qux", actual)

	// check if all the expectations were met
	server.ExpectationsMet(t)
}
