package main

import (
	"net"
	"io"
	"strings"
	"time"
	"strconv"
)

type Connection struct {
	amount     int
	id         string
	localAddr  string
	remoteAddr []Channel
	outputC    *chan string
	record     bool
}

func NewConnection(id, laddr string, raddr []Channel, outputC *chan string) *Connection {
	return &Connection{
		amount:     1000,
		id:         id,
		localAddr:  laddr,
		remoteAddr: raddr,
		outputC:    outputC,
		record:     false,
	}
}

func (th *Connection) SendMarker() {
	th.record = true
	*th.outputC <- "START RECORD"
	time.Sleep(2 * time.Second)
	for _, adress := range th.remoteAddr {
		conn, err := net.Dial("tcp", adress.RemoteAddress)
		if err != nil {
			*th.outputC <- "ERROR: " + err.Error()
			return
		}
		_, err = conn.Write([]byte(th.id + "|marker"))
		if err != nil {
			*th.outputC <- "ERROR: " + err.Error()
			return
		}
	}
}

func (th *Connection) receiveFromAllChannels() bool {
	for _, channel := range th.remoteAddr {
		if channel.buffer == "" {
			return false
		}
	}
	return true
}

func (th *Connection) ReceiveMessage() {
	listener, err := net.Listen("tcp", th.localAddr)
	if err != nil {
		*th.outputC <- "ERROR: " + err.Error()
		return
	}
	for {
		_, buffer, err := th.receive(listener)
		if err != nil {
			*th.outputC <- "ERROR: " + err.Error()
			return
		}
		*th.outputC <- string(buffer)
		split := strings.SplitN(string(buffer), "|", -1)
		if len(split) < 2 {
			continue
		}
		channelId := split[0]
		message := split[1]
		if message != "marker" {
			amount, err := strconv.Atoi(message)
			if err != nil {
				*th.outputC <- "ERROR: " + err.Error()
				continue
			}
			th.amount = th.amount+amount
		}
		*th.outputC <- "Received " + message + " From " + channelId + " Current Amount " + strconv.Itoa(th.amount)
		if message == "marker" && !th.record {
			th.SendMarker()
			th.bufferMessage(channelId, message)
			th.checkChannels()
			continue
		}
		if th.record {
			th.bufferMessage(channelId, message)
			th.checkChannels()
		}
	}
}

func (th *Connection) bufferMessage(channelId, message string) {
	for i, channel := range th.remoteAddr {
		if channel.ChannelID == channelId && channel.buffer != "marker" {
			if message == "marker" {
				th.remoteAddr[i].buffer = th.remoteAddr[i].buffer + " " + "clear"
			} else {
				th.remoteAddr[i].buffer = th.remoteAddr[i].buffer + " " + message
			}
		}
	}
}


func (th *Connection) checkChannels() {
	if th.receiveFromAllChannels() {
		*th.outputC <- "FINISH RECORD"
		*th.outputC <- "AMOUNT: " + strconv.Itoa(th.amount)
		for _, channel := range th.remoteAddr {
			*th.outputC <- "CHANNEL (" + th.id + "," + channel.ChannelID + ") = " + channel.buffer
		}
		th.record = false
	}
}

func (th *Connection) SendMessage(buffer int) {
	for _, address := range th.remoteAddr {
		conn, err := net.Dial("tcp", address.RemoteAddress)
		if err != nil {
			*th.outputC <- "DIAL ERROR: " + err.Error()
			return
		}
		th.amount = th.amount - buffer
		*th.outputC <- "Send " + strconv.Itoa(buffer) + " To " + address.ChannelID + " Current Amount " + strconv.Itoa(th.amount)
		_, err = conn.Write([]byte(th.id + "|" + strconv.Itoa(buffer)))
		if err != nil {
			*th.outputC <- "ERROR: " + err.Error()
			return
		}
	}
}

func (th *Connection) receive(listener net.Listener) (string, []byte, error) {
	conn, err := listener.Accept()
	if err != nil {
		*th.outputC <- "ERROR: " + err.Error()
		return "", []byte{}, err
	}
	buffer := make([]byte, 1024)
	n, err := conn.Read(buffer)
	if err != nil && err != io.EOF {
		*th.outputC <- "ERROR: " + err.Error()
		return "", []byte{}, err
	}
	return conn.RemoteAddr().String(), buffer[:n], nil
}
