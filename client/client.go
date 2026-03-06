package main

import (
	"encoding/binary"
	"net"
	"sync/atomic"
	"time"

	"github.com/exam-gaurd/client/capture"
	"github.com/exam-gaurd/client/encoder"
)

// Protocol constants
const (
	UPDATE_INTERVAL = time.Second / 6 // 6 FPS for better performance
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

// Client handles screen capture and streaming to the server.
type Client struct {
	isRunning      atomic.Bool
	isConnected    atomic.Bool
	socket         *net.TCPConn
	lastSentTime   atomic.Value
	onConnected    func()
	onError        func(error)
	cachedServerIP string

	// New capture system
	capturer capture.Capturer
	enc      *encoder.Encoder

	// Statistics
	framesSent    atomic.Int64
	framesDropped atomic.Int64
}

// NewClient creates a new streaming client.
func NewClient() *Client {
	client := &Client{
		isRunning:   atomic.Bool{},
		isConnected: atomic.Bool{},
		socket:      nil,
	}
	client.isConnected.Store(false)
	client.isRunning.Store(false)
	client.lastSentTime.Store(time.Time{})
	return client
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

func (client *Client) Start(studentId, studentName string, port int, serverIP string, updateUI func()) {
	client.isRunning.Store(true)
	go func() {
		retryDelay := 1 * time.Second
		timeout := 10 * time.Second

		for client.isRunning.Load() {
			client.isConnected.Store(false)
			updateUI()

			var serverAddress string
			var err error

			if serverIP != "" {
				serverAddress = serverIP
			} else if client.cachedServerIP != "" {
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

			// Optimize TCP settings
			client.socket.SetKeepAlive(true)
			client.socket.SetKeepAlivePeriod(5 * time.Second)
			client.socket.SetNoDelay(true)
			client.socket.SetWriteBuffer(256 * 1024) // Larger write buffer

			client.isConnected.Store(true)
			if client.onConnected != nil {
				client.onConnected()
			}
			updateUI()
			retryDelay = 1 * time.Second

			client.SendStudentName(studentId + "###" + studentName)

			// Run the new streaming loop with compositor-based capture
			client.runStreamingLoop(updateUI)

			client.socket.Close()
		}
	}()
}

// runStreamingLoop runs the main capture-encode-send loop using the new capture system.
func (client *Client) runStreamingLoop(updateUI func()) {
	// Initialize platform-specific capturer (auto-detected via build tags)
	client.capturer = capture.NewPlatformCapturer()
	if err := client.capturer.Start(); err != nil {
		if client.onError != nil {
			client.onError(err)
		}
		client.isConnected.Store(false)
		return
	}
	defer client.capturer.Stop()

	// Create encoder with optimized settings
	client.enc = encoder.NewEncoder(encoder.EncoderConfig{
		Quality:  45, // Lower quality for bandwidth efficiency
		MaxWidth: 720,
	})

	// Frame timing at 6 FPS
	ticker := time.NewTicker(UPDATE_INTERVAL)
	defer ticker.Stop()

	frameCount := 0
	keyFrameInterval := 30 // Force keyframe every 5 seconds at 6 FPS

	// Send queue with frame dropping to prevent memory growth
	sendQueue := make(chan []byte, 2)
	sendDone := make(chan struct{})

	// Start background send worker
	go func() {
		defer close(sendDone)
		for data := range sendQueue {
			if !client.isConnected.Load() {
				return
			}
			err := client.SendScreenshot(data)
			if err != nil {
				client.isConnected.Store(false)
				return
			}
			client.lastSentTime.Store(time.Now())
			client.framesSent.Add(1)
		}
	}()

	defer func() {
		close(sendQueue)
		<-sendDone
	}()

	for client.isConnected.Load() && client.isRunning.Load() {
		// Wait for next frame interval
		<-ticker.C

		// Capture frame using compositor-based capture
		frameData, err := client.capturer.ReadFrame()
		if err != nil {
			client.isConnected.Store(false)
			return
		}

		if frameData == nil {
			continue // No new frame available
		}

		// Force keyframe periodically for reliability
		if frameCount%keyFrameInterval == 0 {
			frameData.IsKeyFrame = true
		}
		frameCount++

		// Encode frame (handles both keyframes and dirty rects)
		encoded, err := client.enc.Encode(frameData)
		if err != nil || encoded == nil {
			continue
		}

		// Try to send, drop if queue full (prevents memory growth)
		select {
		case sendQueue <- encoded.Data:
			// Successfully queued
		default:
			// Queue full, drop frame to maintain responsiveness
			client.framesDropped.Add(1)
		}

		updateUI()
	}
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

	client.socket.SetWriteDeadline(time.Now().Add(5 * time.Second))
	_, err := client.socket.Write(data)

	if err != nil {
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

// Stats returns frame transmission statistics.
func (client *Client) Stats() (sent, dropped int64) {
	return client.framesSent.Load(), client.framesDropped.Load()
}

func min(a, b time.Duration) time.Duration {
	if a < b {
		return a
	}
	return b
}
