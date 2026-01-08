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
	isRunning      atomic.Bool
	isConnected    atomic.Bool
	socket         *net.TCPConn
	lastSentTime   atomic.Value
	onConnected    func()
	onError        func(error)
	cachedServerIP string
}

func NewClient() *Client {
	client := Client{
		isRunning:   atomic.Bool{},
		isConnected: atomic.Bool{},
		socket:      nil,
	}
	client.isConnected.Store(false)
	client.isRunning.Store(false)
	client.lastSentTime.Store(time.Time{})
	return &client
}

func (client *Client) SetCallbacks(onConnected func(), onError func(error)) {
	client.onConnected = onConnected
	client.onError = onError
}

func (client *Client) GetLastSentTime() time.Time {
	if t := client.lastSentTime.Load(); t != nil {
		return t.(time.Time)
	}
	return time.Time{}
}

func (client *Client) Start(studentId, studentName string, port int, updateUI func()) {
	client.isRunning.Store(true)
	go func() {
		retryDelay := 1 * time.Second
		timeout := 10 * time.Second

		for client.isRunning.Load() {
			client.isConnected.Store(false)
			updateUI()

			var serverAddress string
			var err error

			if client.cachedServerIP != "" {
				serverAddress = client.cachedServerIP
			} else {
				serverAddress, err = discoverServerWithTimeout(port, timeout)
				if err != nil {
					if client.onError != nil {
						client.onError(err)
					}
					client.cachedServerIP = ""
					time.Sleep(retryDelay)
					retryDelay = min(retryDelay*2, 8*time.Second)
					continue
				}

				client.cachedServerIP = serverAddress
			}

			client.socket, err = net.DialTCP("tcp", nil, &net.TCPAddr{IP: net.ParseIP(serverAddress), Port: port})
			if err != nil {
				if client.onError != nil {
					client.onError(err)
				}
				client.cachedServerIP = ""
				time.Sleep(retryDelay)
				retryDelay = min(retryDelay*2, 8*time.Second)
				continue
			}

			client.socket.SetKeepAlive(true)
			client.socket.SetKeepAlivePeriod(5 * time.Second)
			client.socket.SetNoDelay(true)

			client.isConnected.Store(true)
			if client.onConnected != nil {
				client.onConnected()
			}
			updateUI()
			retryDelay = 1 * time.Second

			client.SendStudentName(studentId + "###" + studentName)

			for client.isConnected.Load() && client.isRunning.Load() {
				screenshot, err := client.captureScreen()
				if err != nil {
					client.isConnected.Store(false)
					break
				}
				err = client.SendScreenshot(screenshot)
				if err != nil {
					break
				}
				client.lastSentTime.Store(time.Now())
				updateUI()
				time.Sleep(UPDATE_INTERVAL)
			}

			client.socket.Close()
		}

	}()
}

func discoverServerWithTimeout(port int, timeout time.Duration) (string, error) {
	address := net.UDPAddr{
		IP:   net.IPv4(0, 0, 0, 0),
		Port: port,
	}

	conn, err := net.ListenUDP("udp", &address)
	if err != nil {
		return "", err
	}
	defer conn.Close()

	conn.SetReadDeadline(time.Now().Add(timeout))

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

func (client *Client) SendStudentName(name string) error {
	if client.socket != nil {
		return client.sendData(NAME, []byte(name))
	}
	return nil
}

func (client *Client) SendScreenshot(screenshot []byte) error {
	if client.socket != nil {
		return client.sendData(PICTURE, screenshot)
	}
	return nil
}

func (client *Client) SendMessage(msg string) error {
	if client.socket != nil {
		return client.sendData(MESSAGE, []byte(msg))
	}
	return nil
}

func (client *Client) Stop() {
	client.isRunning.Store(false)
	client.isConnected.Store(false)
	if client.socket != nil {
		client.socket.Close()
	}
}

func (client *Client) sendData(dataType uint16, dataBytes []byte) error {
	data := make([]byte, HEADER_SIZE+len(dataBytes))
	copy(data, client.packHeader(dataType, len(dataBytes)))
	copy(data[HEADER_SIZE:], dataBytes)
	_, err := client.socket.Write(data)

	if err != nil {
		println(err.Error())
		client.isConnected.Store(false)
		return err
	}
	return nil
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
