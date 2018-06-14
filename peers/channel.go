package peers

import (
	"gobby/logs"
	"gobby/pwp"
	"net"
	"os"
	"syscall"
)

const (
	_MAX_MESSAGE_SIZE = 1024 * 100
	_MAX_BUFFER_SIZE  = _MAX_MESSAGE_SIZE * 2
)

// TODO - improve error handling
type peerChannel struct {
	socket     net.Conn
	buffer     []byte
	outgoingCh chan pwp.Message
	stopCh     chan struct{}
}

func newPeerChannel(socket net.Conn) *peerChannel {
	return &peerChannel{
		socket:     socket,
		buffer:     make([]byte, 0, _MAX_BUFFER_SIZE),
		outgoingCh: make(chan pwp.Message, 10),
		stopCh:     make(chan struct{}, 0),
	}
}

func (c *peerChannel) Start(incoming chan<- pwp.Message) {
	go c.sending()
	go c.receiving(incoming)
}

func (c *peerChannel) Stop() {
	close(c.stopCh)
	c.socket.Close()
}

func (c *peerChannel) Send(message pwp.Message) {
	c.outgoingCh <- message
}

func (c *peerChannel) sending() {
	for {
		select {
		case msg := <-c.outgoingCh:
			data := msg.Encode()
			totalwc := 0
			for totalwc < len(data) {
				wc, err := c.socket.Write(data[totalwc:])
				if err != nil {
					select {
					case <-c.stopCh:
						return
					default:
						logs.Warn("PeerChannel", "Failed to write to socket: %s", err)
						c.socket.Close()
						return
					}
				}
				totalwc += wc
			}
		case <-c.stopCh:
			return
		}
	}
}

func (c *peerChannel) receiving(incoming chan<- pwp.Message) {
	var toRead int
	data := make([]byte, _MAX_MESSAGE_SIZE)

	for {
		leftInBuffer := _MAX_BUFFER_SIZE - len(c.buffer)
		if leftInBuffer < len(data) {
			toRead = leftInBuffer
		} else {
			toRead = len(data)
		}

		rc, err := c.socket.Read(data[:toRead])
		if err != nil {
			if opErr, ok := err.(*net.OpError); ok {
				if syscallErr, ok := opErr.Err.(*os.SyscallError); ok {
					if syscallErr.Err == syscall.ECONNRESET {
						logs.Debug("PeerChannel", "Remote peer closed connection")
						close(incoming)
						return
					}
				}
			}

			select {
			case <-c.stopCh:
				return
			default:
				logs.Warn("PeerChannel", "Failed to read data: %s", err)
				close(incoming)
				return
			}
		}

		c.buffer = append(c.buffer, data[:rc]...)
		messages, leftover, err := pwp.DecodeMessages(c.buffer)
		if err != nil {
			logs.Warn("PeerChannel", "Failed to decode messages from %s: %s. Buffer length: %d", c.socket.RemoteAddr().String(), err, len(c.buffer))
			c.Stop()
			close(incoming)
			return
		}

		copy(c.buffer, leftover)
		c.buffer = c.buffer[:len(leftover)]
		for _, message := range messages {
			incoming <- message
		}
	}
}
