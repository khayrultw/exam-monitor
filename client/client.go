package main

import (
	"bytes"
	"encoding/binary"
	"image/jpeg"
	"net"
	"sync/atomic"
	"time"

	"github.com/kbinani/screenshot"
	"github.com/nfnt/resize"
)

const (
	UPDATE_INTERVAL = time.Second / 2
	NAME            = 0
	MESSAGE         = 1
	PICTURE         = 2
	HEADER_SIZE     = 8
)

const (
	CONNECTED int = iota
	RUNNING
	NOT_RUNNING
)

type Client struct {
	isRunning   atomic.Bool
	isConnected atomic.Bool
	socket      *net.TCPConn
}

func NewClient() *Client {
	client := Client{
		isRunning:   atomic.Bool{},
		isConnected: atomic.Bool{},
		socket:      nil,
	}
	client.isConnected.Store(false)
	client.isRunning.Store(false)
	return &client
}

func (client *Client) Start(studentName string, port int, updateUI func()) {
	client.isRunning.Store(true)
	go func() {
		retryDelay := 1 * time.Second
		for client.isRunning.Load() {
			client.isConnected.Store(false)
			updateUI()
			serverAddress, err := discoverServer(port)
			//println(serverAddress)
			if err != nil {
				time.Sleep(retryDelay)
				retryDelay = min(retryDelay*2, 8*time.Second)
				println(err)
				continue
			}
			client.isConnected.Store(true)
			updateUI()
			retryDelay = 1 * time.Second
			client.socket, err = net.DialTCP("tcp", nil, &net.TCPAddr{IP: net.ParseIP(serverAddress), Port: port})
			if err != nil {
				time.Sleep(retryDelay)
				retryDelay = min(retryDelay*2, 8*time.Second)
				println(err.Error())
				continue
			}

			client.SendStudentName(studentName)

			for client.isConnected.Load() {
				screenshot, err := client.captureScreen()
				if err != nil {
					break
				}
				client.SendScreenshot(screenshot)
				time.Sleep(UPDATE_INTERVAL)
			}

			client.socket.Close()
		}

	}()
}

func discoverServer(port int) (string, error) {
	address := net.UDPAddr{
		IP:   net.IPv4(0, 0, 0, 0),
		Port: port,
	}

	conn, err := net.ListenUDP("udp", &address)
	if err != nil {
		return "", err
	}
	defer conn.Close()

	buffer := make([]byte, 1024)
	for {
		n, addr, err := conn.ReadFromUDP(buffer)
		if err != nil {
			return "", err
		}
		if string(buffer[:n]) == "server" {
			return addr.IP.String(), nil
		}
	}
}

func (client *Client) captureScreen() ([]byte, error) {
	bounds := screenshot.GetDisplayBounds(0)
	img, err := screenshot.CaptureRect(bounds)
	if err != nil {
		return nil, err
	}

	var buf bytes.Buffer

	resizedImg := resize.Resize(720, 0, img, resize.NearestNeighbor)

	options := jpeg.Options{Quality: 60}
	err = jpeg.Encode(&buf, resizedImg, &options)
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func (client *Client) SendStudentName(name string) {
	if client.socket != nil {
		client.sendData(NAME, []byte(name))
	}
}

func (client *Client) SendScreenshot(screenshot []byte) {
	if client.socket != nil {
		client.sendData(PICTURE, screenshot)
	}
}

func (client *Client) SendMessage(msg string) {
	if client.socket != nil {
		client.sendData(MESSAGE, []byte(msg))
	}
}

func (client *Client) Stop() {
	client.isRunning.Store(false)
	client.isConnected.Store(false)
	if client.socket != nil {
		client.socket.Close()
	}
}

func (client *Client) sendData(dataType uint16, dataBytes []byte) {
	data := make([]byte, HEADER_SIZE+len(dataBytes))
	copy(data, client.packHeader(dataType, len(dataBytes)))
	copy(data[HEADER_SIZE:], dataBytes)
	_, err := client.socket.Write(data)

	if err != nil {
		println(err.Error())
		client.isConnected.Store(false)
	}

}

func (client *Client) packHeader(status uint16, length int) []byte {
	data := make([]byte, HEADER_SIZE)

	copy(data, []byte("HE"))
	binary.BigEndian.PutUint16(data[2:], status)
	binary.BigEndian.PutUint32(data[4:], uint32(length))

	return data
}

func min(a, b time.Duration) time.Duration {
	if a < b {
		return a
	}
	return b
}
