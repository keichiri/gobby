package peers

import (
	"fmt"
	"gobby/logs"
	"gobby/pwp"
	"net"
	"sync"
)

type PeerServer struct {
	peerID         []byte
	port           string
	socket         net.Listener
	closed         bool
	coordinatorsMx sync.Mutex
	coordinators   map[string]*PeerCoordinator
}

func NewPeerServer(peerID []byte, port string) *PeerServer {
	return &PeerServer{
		peerID:       peerID,
		port:         port,
		coordinators: make(map[string]*PeerCoordinator),
	}
}

func (s *PeerServer) Register(infoHash []byte, coordinator *PeerCoordinator) {
	s.coordinatorsMx.Lock()
	s.coordinators[string(infoHash)] = coordinator
	s.coordinatorsMx.Unlock()
}

func (s *PeerServer) Deregister(infoHash []byte) {
	s.coordinatorsMx.Lock()
	delete(s.coordinators, string(infoHash))
	s.coordinatorsMx.Unlock()
}

func (s *PeerServer) getCoordinator(infoHash []byte) *PeerCoordinator {
	s.coordinatorsMx.Lock()
	coordinator := s.coordinators[string(infoHash)]
	s.coordinatorsMx.Unlock()
	return coordinator
}

func (s *PeerServer) Stop() {
	if !s.closed && s.socket != nil {
		s.closed = true
		s.socket.Close()
	}
}

func (s *PeerServer) Serve() error {
	logs.Info("PeerServer", "Opening server on port %s", s.port)
	socket, err := net.Listen("tcp", ":"+s.port)
	if err != nil {
		return fmt.Errorf("Failed to open server socket. Error: %s", err)
	}

	s.socket = socket
	for {
		clientSocket, err := s.socket.Accept()
		if err != nil {
			if s.closed {
				logs.Info("PeerServer", "Terminating server on port %s", s.port)
				return nil
			} else {
				return fmt.Errorf("Error while accepting connections on port %s: %s", s.port, err)
			}
		}

		go s.handleIncomingPeer(clientSocket)
	}
}

func (s *PeerServer) handleIncomingPeer(socket net.Conn) {
	peerAddress := socket.RemoteAddr().String()
	logs.Debug("PeerServer", "Incoming connection at port %s from %s", s.port, peerAddress)

	peerHandshakeData := make([]byte, 0, 68)
	readCount, err := socket.Read(peerHandshakeData)
	if err != nil {
		logs.Warn("PeerServer", "Error while receiving handshake from %s", peerAddress)
		socket.Close()
		return
	}
	if readCount != 68 {
		logs.Warn("PeerServer", "Read only %d bytes of handshake data from %s", readCount, peerAddress)
		socket.Close()
		return
	}

	infoHash, peerID, err := pwp.ParseHandshake(peerHandshakeData)
	if err != nil {
		logs.Warn("PeerServer", "Invalid handshake from %s: %s", peerAddress, err)
		socket.Close()
		return
	}

	coordinator := s.getCoordinator(infoHash)
	if coordinator == nil {
		logs.Warn("PeerServer", "Nonexistant info hash from %s", peerAddress)
		socket.Close()
		return
	}

	if !coordinator.CanAcceptMore() {
		logs.Debug("PeerServer", "Refusing connection from %s. Cannot accept more", peerAddress)
		socket.Close()
		return
	}

	responseHandshake := pwp.EncodeHandshake(infoHash, s.peerID)
	_, err = socket.Write(responseHandshake)
	if err != nil {
		logs.Warn("PeerServer", "Failed to send response handshake to %s: %s", peerAddress, err)
		socket.Close()
		return
	}

	logs.Debug("PeerServer", "Exchanged handshake with %s. Handing off to coordinator", peerAddress)
	coordinator.HandleIncomingConnection(socket, peerID)
}
