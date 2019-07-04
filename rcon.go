package rcon

import (
	"bufio"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/pkg/errors"
)

type RconClient struct {
	Conn net.Conn
}

func (r *RconClient) Close() {
	r.Conn.Close()
}

func (r *RconClient) Write(message string) (int, error) {
	b := []byte(fmt.Sprintf("%s\n", message))
	return r.Conn.Write(b)
}

func (r *RconClient) WriteTimeout(message string, timeout time.Duration) (int, error) {
	r.Conn.SetWriteDeadline(time.Now().Add(timeout))
	n, err := r.Write(message)
	r.Conn.SetWriteDeadline(time.Time{})
	return n, err
}

func (r *RconClient) Read() (string, error) {
	message, err := bufio.NewReader(r.Conn).ReadString('\n')
	message = strings.TrimRight(message, "\n")
	return message, err
}

func (r *RconClient) ReadTimeout(timeout time.Duration) (string, error) {
	r.Conn.SetReadDeadline(time.Now().Add(timeout))
	message, err := r.Read()
	r.Conn.SetReadDeadline(time.Time{})
	return message, err
}

func IsTimeoutError(err error) bool {
	if err, ok := err.(net.Error); ok && err.Timeout() {
		return true
	}
	return false
}

func DialRcon(address string, password string, timeout time.Duration) (RconClient, error) {
	c, err := net.DialTimeout("tcp", address, timeout)
	rcon := RconClient{c}

	if err != nil {
		return rcon, errors.Wrap(err, "could not connect to rcon server")
	}

	// rcon.Conn.SetDeadline(time.Now().Add(timeout))
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

	// rcon.Conn.SetDeadline(time.Time{})

	return rcon, nil
}

// Rcon commands

func (r *RconClient) Message(message string) error {
	_, err := r.Write(fmt.Sprintf("/msg %s", message))
	return err
}
