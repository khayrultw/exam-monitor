package main

import (
	"bytes"
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
			println(serverAddress)
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

	resizedImg := resize.Resize(1080, 0, img, resize.NearestNeighbor)
	options := jpeg.Options{Quality: 50} // Reduce quality to 40 for better performance
	err = jpeg.Encode(&buf, resizedImg, &options)
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func (client *Client) SendStudentName(name string) {
	if client.socket != nil {
		nameBytes := []byte(name)
		client.sendData(packData(NAME, len(nameBytes)))
		client.sendData(nameBytes)
	}
}

func (client *Client) SendScreenshot(screenshot []byte) {
	if client.socket != nil {
		client.sendData(packData(PICTURE, len(screenshot)))
		client.sendData(screenshot)
		println("Image size:", len(screenshot))
	}
}

func (client *Client) SendMessage(msg string) {
	if client.socket != nil {
		msgBytes := []byte(msg)
		client.sendData(packData(MESSAGE, len(msgBytes)))
		client.sendData(msgBytes)
	}
}

func (client *Client) Stop() {
	client.isRunning.Store(false)
	client.isConnected.Store(false)
	if client.socket != nil {
		client.socket.Close()
	}
}

func (client *Client) sendData(data []byte) {
	_, err := client.socket.Write(data)

	if err != nil {
		println(err.Error())
		client.isConnected.Store(false)
	}
}

func packData(status byte, length int) []byte {
	data := make([]byte, 4)

	data[0] = status
	data[1] = byte((length >> 16) & 0xFF) // MSB
	data[2] = byte((length >> 8) & 0xFF)  // Middle byte
	data[3] = byte(length & 0xFF)         // LSB

	return data
}

func min(a, b time.Duration) time.Duration {
	if a < b {
		return a
	}
	return b
}
