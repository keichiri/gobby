package peers

import (
	"bytes"
	"fmt"
	"gobby/pwp"
	"io"
	"net"
	"os"
	"strconv"
	"sync"
	"syscall"
	"testing"
	"time"
)

const (
	_MOCK_PEER_PORT = 29001
)

type mockPeer struct {
	conn     net.Conn
	port     string
	toSend   []byte
	received []byte
}

func newMockPeer(port string, toSend []byte) *mockPeer {
	return &mockPeer{
		port:     port,
		toSend:   toSend,
		received: make([]byte, 0),
	}
}

func (m *mockPeer) Run() error {
	var err error
	var errMutex sync.Mutex
	var listener net.Listener
	listener, err = net.Listen("tcp", ":"+m.port)
	if err != nil {
		return fmt.Errorf("Failed to open listener socket: %s", err)
	}
	defer listener.Close()

	m.conn, err = listener.Accept()
	if err != nil {
		return fmt.Errorf("Failed to accept connection: %s", err)
	}
	defer m.conn.Close()

	wg := sync.WaitGroup{}
	wg.Add(2)

	go func() {
		err1 := m.sending()
		if err1 != nil {
			errMutex.Lock()
			if err != nil {
				err = err1
			}
			errMutex.Unlock()
		}
		wg.Done()
	}()
	go func() {
		err2 := m.receiving()
		if err2 != nil {
			errMutex.Lock()
			if err != nil {
				err = err2
			}
			errMutex.Unlock()
		}
		wg.Done()
	}()
	time.Sleep(time.Millisecond * 5)
	m.conn.Close()
	wg.Wait()

	return err
}

func (m *mockPeer) sending() error {
	totalwc := 0
	for totalwc < len(m.toSend) {
		wc, err := m.conn.Write(m.toSend[totalwc:])
		if err != nil {
			return fmt.Errorf("Failed to send data over socket: %s", err)
		}
		totalwc += wc
	}

	return nil
}

func (m *mockPeer) receiving() error {
	for {
		buffer := make([]byte, 10000)
		rc, err := m.conn.Read(buffer)
		if opErr, ok := err.(*net.OpError); ok {
			if syscallErr, ok := opErr.Err.(*os.SyscallError); ok {
				if syscallErr.Err == syscall.ECONNRESET {
					return nil
				}
			}
		}

		if err != nil {
			if err == io.EOF {
				return nil
			}

			return fmt.Errorf("Failed to read data from socket: %s", err)
		}

		m.received = append(m.received, buffer[:rc]...)
	}
}

func (m *mockPeer) getReceived() []byte {
	return m.received
}

func TestChannel(t *testing.T) {
	mockPeerToSend := []pwp.Message{
		&pwp.BitfieldMsg{[]byte("test_bitfield")},
		&pwp.KeepAliveMsg{},
		&pwp.ChokeMsg{},
		&pwp.HaveMsg{1000},
		&pwp.PieceMsg{0, 10000, make([]byte, 10000)},
		&pwp.UnchokeMsg{},
		&pwp.CancelMsg{1, 10000, 10000},
		&pwp.InterestedMsg{},
		&pwp.UninterestedMsg{},
		&pwp.RequestMsg{1000, 2000, 1000},
		&pwp.PieceMsg{1, 10000, make([]byte, 10000)},
	}
	channelToSend := []pwp.Message{
		&pwp.BitfieldMsg{[]byte("test_bitfield_2")},
		&pwp.InterestedMsg{},
		&pwp.UnchokeMsg{},
		&pwp.PieceMsg{1000, 10000, make([]byte, 10000)},
		&pwp.PieceMsg{1000, 20000, make([]byte, 10000)},
		&pwp.UninterestedMsg{},
		&pwp.HaveMsg{100},
		&pwp.RequestMsg{101, 10000, 10000},
		&pwp.PieceMsg{1000, 30000, make([]byte, 10000)},
		&pwp.ChokeMsg{},
	}
	mockPeerToSendBytes := make([]byte, 0)
	for _, message := range mockPeerToSend {
		mockPeerToSendBytes = append(mockPeerToSendBytes, message.Encode()...)
	}
	channelToSendBytes := make([]byte, 0)
	for _, message := range channelToSend {
		channelToSendBytes = append(channelToSendBytes, message.Encode()...)
	}

	peer := newMockPeer(strconv.Itoa(_MOCK_PEER_PORT), mockPeerToSendBytes)
	go peer.Run()
	time.Sleep(time.Millisecond * 1)

	conn, err := net.Dial("tcp", "127.0.0.1:"+strconv.Itoa(_MOCK_PEER_PORT))
	if err != nil {
		t.Fatalf("Failed to connect to mock peer. Err: %s", err)
	}

	receivedBytes := make([]byte, 0)
	channel := newPeerChannel(conn)
	messagesCh := make(chan pwp.Message, 10)
	channel.Start(messagesCh)
	go func() {
		for msg := range messagesCh {
			receivedBytes = append(receivedBytes, msg.Encode()...)
		}
	}()

	go func() {
		for _, message := range channelToSend {
			channel.Send(message)
		}
	}()
	time.Sleep(time.Millisecond * 1)

	if err != nil {
		t.Fatalf("Error occurred: %s", err)
	}

	peerReceived := peer.getReceived()
	if !bytes.Equal(peerReceived, channelToSendBytes) {
		t.Fatalf("Peer received length: %d. Channel supposed to send length: %d", len(peerReceived), len(channelToSendBytes))
	}

	if !bytes.Equal(receivedBytes, mockPeerToSendBytes) {
		t.Fatalf("Channel received length: %d. Mock peer supposed to send length: %d", len(receivedBytes), len(mockPeerToSendBytes))
	}
}
