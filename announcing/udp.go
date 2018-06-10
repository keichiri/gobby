package announcing

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"gobby/logs"
	"math/rand"
	"net"
	"time"
)

const (
	_PROTOCOL_ID        int64 = 0x41727101980
	_MAX_TRANSACTION_ID int32 = 2147483647
	_READ_TIMEOUT             = time.Second * 5
)

type udpAdapter struct {
	socket net.Conn
}

func newUDPAdapter(host string, port string) (*udpAdapter, error) {
	conn, err := net.Dial("udp", host+":"+port)
	if err != nil {
		return nil, fmt.Errorf("Failed to open UDP socket: %s", err)
	}

	adapter := &udpAdapter{
		socket: conn,
	}

	return adapter, nil
}

func (a *udpAdapter) Announce(params map[string]interface{}) (*AnnounceResult, int, error) {
	connectionID, err := a.getConnectID()
	if err != nil {
		return nil, 0, fmt.Errorf("Failed to obtain connection id: %s", err)
	}

	tid := a.generateTID()
	announceData, numwant := a.prepareAnnounceData(params, connectionID, tid)

	// TODO - remove this, after first run
	if len(announceData) != 98 {
		logs.Critical("Announcer", "UDP announce data not 98 bytes long. Length: %d", len(announceData))
		panic("UDP announce data not 98 bytes long")
	}

	_, err = a.socket.Write(announceData)
	if err != nil {
		return nil, 0, fmt.Errorf("Failed to send data to tracker: %s", err)
	}

	err = a.socket.SetReadDeadline(time.Now().Add(_READ_TIMEOUT))
	if err != nil {
		return nil, 0, fmt.Errorf("Failed to set read deadline when connecting: %s", err)
	}

	responseData := make([]byte, 0, 20+numwant*6)
	_, err = a.socket.Read(responseData)
	if err != nil {
		return nil, 0, fmt.Errorf("Error while waiting for announce response: %s", err)
	}

	respTID, res, interval, err := a.parseAnnounceResponse(responseData)
	if err != nil {
		return nil, 0, err
	}
	if tid != respTID {
		return nil, 0, fmt.Errorf("Transaction ID missmath: %d and %d", tid, respTID)
	}

	return res, interval, nil
}

func (a *udpAdapter) getConnectID() (int64, error) {
	tid := a.generateTID()
	connectData := a.prepareConnectData(tid)
	_, err := a.socket.Write(connectData)
	if err != nil {
		return 0, fmt.Errorf("Failed to send data to tracker: %s", err)
	}

	err = a.socket.SetReadDeadline(time.Now().Add(_READ_TIMEOUT))
	if err != nil {
		return 0, fmt.Errorf("Failed to set read deadline when connecting: %s", err)
	}

	responseData := make([]byte, 16)
	rc, err := a.socket.Read(responseData)
	if err != nil {
		return 0, fmt.Errorf("Error while waiting for connect response: %s", err)
	}
	if rc != 16 {
		return 0, fmt.Errorf("Failed to get connect response. Read: %d", rc)
	}

	respTID, connectionID, err := a.parseConnectResponse(responseData)
	if err != nil {
		return 0, err
	}
	if tid != respTID {
		return 0, fmt.Errorf("Transaction ID missmath: %d and %d", tid, respTID)
	}

	return connectionID, nil
}

func (a *udpAdapter) generateTID() int32 {
	return rand.Int31n(_MAX_TRANSACTION_ID)
}

func (a *udpAdapter) prepareConnectData(tid int32) []byte {
	buf := new(bytes.Buffer)
	binary.Write(buf, binary.BigEndian, _PROTOCOL_ID)
	binary.Write(buf, binary.BigEndian, int32(0))
	binary.Write(buf, binary.BigEndian, tid)
	return buf.Bytes()
}

func (a *udpAdapter) parseConnectResponse(response []byte) (int32, int64, error) {
	var action int32
	buf := bytes.NewBuffer(response)
	binary.Read(buf, binary.BigEndian, &action)
	if action != 0 {
		return 0, 0, fmt.Errorf("Tracker returned action %d in connect response", action)
	}

	var tid int32
	binary.Read(buf, binary.BigEndian, &tid)

	var connectionID int64
	binary.Read(buf, binary.BigEndian, &connectionID)

	return tid, connectionID, nil
}

func (a *udpAdapter) prepareAnnounceData(params map[string]interface{}, connID int64, tid int32) ([]byte, int32) {
	event := params["event"].(string)
	var eventID int32
	switch event {
	case "":
		eventID = 0
	case "completed":
		eventID = 1
	case "started":
		eventID = 2
	case "stopped":
		eventID = 3
	}

	buf := new(bytes.Buffer)
	binary.Write(buf, binary.BigEndian, connID)
	binary.Write(buf, binary.BigEndian, int32(1))
	binary.Write(buf, binary.BigEndian, tid)
	buf.Write(params["infoHash"].([]byte))
	buf.Write(params["peerID"].([]byte))
	binary.Write(buf, binary.BigEndian, int64(params["downloaded"].(int)))
	binary.Write(buf, binary.BigEndian, int64(params["left"].(int)))
	binary.Write(buf, binary.BigEndian, int64(params["uploaded"].(int)))
	binary.Write(buf, binary.BigEndian, eventID)
	binary.Write(buf, binary.BigEndian, int32(0))
	binary.Write(buf, binary.BigEndian, params["key"].(int32))
	numwant := params["numwant"].(int32)
	binary.Write(buf, binary.BigEndian, numwant)
	binary.Write(buf, binary.BigEndian, params["port"].(int16))

	return buf.Bytes(), numwant
}

func (a *udpAdapter) parseAnnounceResponse(response []byte) (int32, *AnnounceResult, int, error) {
	if len(response) < 20 {
		return 0, nil, 0, errors.New("Response too short")
	}
	if (len(response)-20)%6 != 0 {
		return 0, nil, 0, errors.New("Response peer data not divisible by 6")
	}

	var action, tid, interval, complete, incomplete int32
	buf := bytes.NewBuffer(response)

	binary.Read(buf, binary.BigEndian, &action)
	if action != 1 {
		return 0, nil, 0, fmt.Errorf("Tracker returned action %d in announce response", action)
	}

	binary.Read(buf, binary.BigEndian, &tid)
	binary.Read(buf, binary.BigEndian, &interval)
	binary.Read(buf, binary.BigEndian, &incomplete)
	binary.Read(buf, binary.BigEndian, &complete)

	peerData := response[20:]

	announceResult := &AnnounceResult{
		Complete:   complete,
		Incomplete: incomplete,
		PeerData:   peerData,
	}

	return tid, announceResult, int(interval), nil
}

func (a *udpAdapter) Close() {
	a.socket.Close()
}
