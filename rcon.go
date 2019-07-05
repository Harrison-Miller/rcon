package rcon

import (
	"bufio"
	"fmt"
	"net"
	"strings"
	"time"

	"regexp"

	"github.com/pkg/errors"
)

type Client struct {
	Conn     net.Conn
	handlers []*Handler
	reader   *bufio.Reader
}

func (r *Client) Close() {
	r.Conn.Close()
}

func IsValidMessage(message string) bool {
	messages := strings.Split(message, "\n")
	for _, m := range messages {
		if m == "" {
			return false
		}
	}
	return true
}

func (r *Client) Write(message string) (int, error) {
	message = strings.TrimRight(message, "\n")
	if !IsValidMessage(message) {
		return 0, fmt.Errorf("a message was of length 0 which would cause a disconnect")
	}

	b := []byte(fmt.Sprintf("%s\n", message))
	return r.Conn.Write(b)
}

func (r *Client) WriteTimeout(message string, timeout time.Duration) (int, error) {
	r.Conn.SetWriteDeadline(time.Now().Add(timeout))
	n, err := r.Write(message)
	r.Conn.SetWriteDeadline(time.Time{})
	return n, err
}

func (r *Client) Read() (string, error) {
	message, err := r.reader.ReadString('\n')
	message = strings.TrimRight(message, "\n")
	return message, err
}

func (r *Client) ReadTimeout(timeout time.Duration) (string, error) {
	r.Conn.SetReadDeadline(time.Now().Add(timeout))
	message, err := r.Read()
	r.Conn.SetReadDeadline(time.Time{})
	return message, err
}

func RemoveTimestamp(message string) string {
	reg := regexp.MustCompile("^\\[[0-9][0-9]:[0-9][0-9]:[0-9][0-9]\\]\\s")
	return reg.ReplaceAllString(message, "")
}

func IsTimeoutError(err error) bool {
	if err, ok := err.(net.Error); ok && err.Timeout() {
		return true
	}
	return false
}

func DialRcon(address string, password string, timeout time.Duration) (Client, error) {
	c, err := net.DialTimeout("tcp", address, timeout)
	rcon := Client{c, nil, bufio.NewReader(c)}

	if err != nil {
		return rcon, errors.Wrap(err, "could not connect to rcon server")
	}

	rcon.Conn.SetDeadline(time.Now().Add(timeout))
	// We need to send an extra character after the password otherwise we wont error on the next read
	// Sending tcpr('hello') so even if tcpr_everything is turned off we can still read something
	_, err = rcon.Write(password + "\ntcpr('hello')")
	if IsTimeoutError(err) {
		return rcon, errors.Wrap(err, "client timed out while sending passowrd")
	} else if err != nil {
		return rcon, errors.Wrap(err, "error occured while sending the password")
	}

	// Read to check if the connection was closed
	_, err = rcon.Read()
	if IsTimeoutError(err) {
		return rcon, errors.Wrap(err, "client timed out while waiting to be accepted")
	} else if err, ok := err.(*net.OpError); ok {
		return rcon, errors.Wrap(err, "wrong password")
	} else if err != nil {
		return rcon, errors.Wrap(err, "something went wrong while authenticating")
	}

	rcon.Conn.SetDeadline(time.Time{})

	return rcon, nil
}

// Rcon commands

func (r *Client) Message(message string) error {
	_, err := r.Write(fmt.Sprintf("/msg %s", message))
	return err
}

// Handlers

type Handler struct {
	pattern         *regexp.Regexp
	callback        func(Message, *Client) error
	removeTimestamp bool
}

func (h *Handler) RemoveTimestamp() *Handler {
	h.removeTimestamp = true
	return h
}

func (h *Handler) Match(message string, client *Client) error {
	if h.pattern.MatchString(message) {
		args := make(map[string]string)

		for i := 0; i < h.pattern.NumSubexp(); i++ {
			name := h.pattern.SubexpNames()[i+1]
			value := h.pattern.ReplaceAllString(message, fmt.Sprintf("${%s}", name))
			args[name] = value
		}

		return h.callback(Message{message, args}, client)
	}
	return nil
}

type Message struct {
	Text string
	Args map[string]string
}

func (r *Client) HandleFunc(pattern string, handler func(Message, *Client) error) *Handler {
	re := regexp.MustCompile(pattern)
	h := Handler{re, handler, false}
	r.handlers = append(r.handlers, &h)
	return &h
}

func (r *Client) Handle() error {
	for {
		message, err := r.Read()
		if err != nil {
			return errors.Wrap(err, "error reading message")
		}

		err = r.Match(message)
		if err != nil {
			return errors.Wrap(err, "error matching message")
		}
	}
}

func (r *Client) Match(message string) error {
	for _, h := range r.handlers {
		notimestamp := RemoveTimestamp(message)
		var err error
		if h.removeTimestamp {
			err = h.Match(notimestamp, r)
		} else {
			err = h.Match(message, r)
		}

		if err != nil {
			return err
		}
	}
	return nil
}
