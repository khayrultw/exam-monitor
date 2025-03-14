package main

import (
	"bytes"
	"image"
	"io"
	"net"
	"sync/atomic"
	"time"
)

type Server struct {
	listener      *net.TCPListener
	isRunning     atomic.Bool
	AddStudent    func() *Student
	RemoveStudent func(id int)
}

func NewServer() *Server {
	server := Server{
		isRunning: atomic.Bool{},
	}

	server.isRunning.Store(false)
	return &server
}

func (s *Server) Start(port int) {
	s.isRunning.Store(true)
	go s.broadcastHost(port)
	println("Server started")
	go func() {
		listener, err := net.ListenTCP("tcp", &net.TCPAddr{Port: port})
		if err != nil {
			s.isRunning.Store(false)
			return
		}
		defer listener.Close()
		s.listener = listener
		for s.isRunning.Load() {
			println("Waiting for connection", port)
			conn, err := listener.AcceptTCP()
			if err != nil {
				continue
			}
			conn.SetKeepAlive(true)
			conn.SetKeepAlivePeriod(5 * time.Second)
			conn.SetNoDelay(true)

			go s.handleStudent(conn)
		}

		println("Server stopped")
	}()
}

func (s *Server) handleStudent(socket *net.TCPConn) {
	student := s.AddStudent()

	go func() {
		defer socket.Close()
		header := make([]byte, 4)
		for s.isRunning.Load() {
			_, err := socket.Read(header)

			if err != nil {
				s.RemoveStudent(student.Id)
				break
			}

			dataType, dataSize := unpackData(header)
			data := make([]byte, dataSize)
			println("Reading data", dataSize)
			_, err = io.ReadFull(socket, data)

			if err != nil {
				continue
			}

			switch dataType {
			case 0:
				name := string(data)
				student.Name = name
			case 1:
				msg := string(data)
				println(msg)
			default:
				img, _, err := image.Decode(bytes.NewReader(data))
				if err == nil {
					student.Image = img // Check if zero array can be formed

				}
			}

			time.Sleep(time.Millisecond * 100)
		}
		s.RemoveStudent(student.Id)
	}()
}

func (s *Server) broadcastHost(port int) {
	address := net.UDPAddr{
		IP:   net.IPv4(255, 255, 255, 255),
		Port: port,
	}
	conn, err := net.DialUDP("udp", nil, &address)
	if err != nil {
		return
	}
	defer conn.Close()

	message := []byte("server")
	for s.isRunning.Load() {
		_, err := conn.Write(message)
		if err != nil {
			return
		}
		time.Sleep(2 * time.Second)
	}
}

func (s *Server) Stop() {
	s.isRunning.Store(false)
	if s.listener != nil {
		s.listener.Close()
	}
}

func unpackData(data []byte) (byte, int) {
	if len(data) != 4 {
		panic("Invalid data length")
	}

	status := data[0]
	length := (int(data[1]) << 16) | (int(data[2]) << 8) | int(data[3])

	return status, length
}
