// Package nsca is a Go client for the Nagios Service Check Acceptor (NSCA).
package nsca

import (
	"net"
	"time"
)

// ServerInfo contains the configuration information for an NSCA server
type ServerInfo struct {
	// Host is the IP address or host name of the NSCA server. Leave empty for localhost.
	Host string
	// Port is the IP port number (no default)
	Port string
	// EncryptionMethod specifies the message encryption to use on NSCA messages. It defaults to ENCRYPT_NONE.
	EncryptionMethod int
	// Password is used in encryption.
	Password string
	// Timeout is the connect/read/write network timeout
	Timeout time.Duration
}

// Message is the contents of an NSCA message
type Message struct {
	// State is one of {STATE_OK, STATE_WARNING, STATE_CRITICAL, STATE_UNKNOWN}
	State int16
	// Host is the host name to set for the NSCA message
	Host string
	// Service is the service name to set for the NSCA message [optional]
	Service string
	// Message is the "plugin output" of the NSCA message [optional]
	Message string
	// Status is an optional channel that recieves the status of a message delivery attempt
	Status chan<- error
}

// RunEndpoint creates a long-lived connection to an NSCA server. Messages sent into the messages
// channel are sent to the NSCA server. Close the quit channel to end the routine. RunEndpoint
// does it's own initialization, cleanup and error recovery and can safely be used from multiple threads.
func RunEndpoint(connectInfo ServerInfo, quit <-chan interface{}, messages <-chan *Message) {
	server := new(NSCAServer)
	defer server.Close()
	var err error
	for {
		select {
		case <-quit:
			return
		case m := <-messages:
			if server.conn == nil {
				err = server.Connect(connectInfo)
			}
			if err == nil {
				err = server.Send(m)
			}
			if m.Status != nil {
				m.Status <- err
			}
			if err != nil {
				server.Close()
				err = nil
			}
		}
	}
}

// NSCAServer can be used as a lower-level alternative to RunEndpoint. It is NOT safe
// to use an instance across mutiple threads.
type NSCAServer struct {
	conn            net.Conn
	encryption      *encryption
	serverTimestamp uint32
	timeout         time.Duration
}

// Connect to an NSCA server.
func (n *NSCAServer) Connect(connectInfo ServerInfo) error {
	var conn net.Conn
	var err error
	if connectInfo.Timeout > 0 {
		conn, err = net.DialTimeout("tcp", net.JoinHostPort(connectInfo.Host, connectInfo.Port), connectInfo.Timeout)
	} else {
		conn, err = net.Dial("tcp", net.JoinHostPort(connectInfo.Host, connectInfo.Port))
	}
	if err != nil {
		return err
	}
	ip, err := readInitializationPacket(conn)
	if err != nil {
		conn.Close()
		return err
	}
	n.Close()
	n.encryption = newEncryption(connectInfo.EncryptionMethod, ip.iv, connectInfo.Password)
	n.serverTimestamp = ip.timestamp
	n.timeout = connectInfo.Timeout
	n.conn = conn
	return nil
}

// Close the connection and clean up.
func (n *NSCAServer) Close() {
	if n.conn != nil {
		n.conn.Close()
		n.conn = nil
	}
	n.serverTimestamp = 0
	n.encryption = nil
	n.timeout = 0
}

// Send an NSCA message.
func (n *NSCAServer) Send(message *Message) error {
	msg := newDataPacket(n.serverTimestamp, message.State, message.Host, message.Service, message.Message)
	if n.timeout > 0 {
		n.conn.SetDeadline(time.Now().Add(n.timeout))
	}
	err := msg.write(n.conn, n.encryption)
	return err
}
