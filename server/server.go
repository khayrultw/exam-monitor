package main

import (
	"bytes"
	"encoding/binary"
	"errors"
	"image"
	"io"
	"net"
	"sync/atomic"
	"time"
)

const (
	HEADER_SIZE = 8
)

type Server struct {
	listener    *net.TCPListener
	isRunning   atomic.Bool
	studentUtil StudentUtil
}

type StudentUtil interface {
	AddStudent(name string) int
	RemoveStudent(id int)
	UpdateImage(id int, img image.Image)
	UpdateName(id int, name string)
	isExists(id int) bool
}

func NewServer() *Server {
	server := Server{
		isRunning: atomic.Bool{},
	}

	server.isRunning.Store(false)
	return &server
}

func (s *Server) Start(port int) {
	if s.studentUtil == nil {
		return
	}
	s.isRunning.Store(true)
	go s.broadcastHost(port)
	go func() {
		listener, err := net.ListenTCP("tcp", &net.TCPAddr{Port: port})
		if err != nil {
			s.isRunning.Store(false)
			return
		}
		defer listener.Close()
		s.listener = listener
		for s.isRunning.Load() {
			conn, err := listener.AcceptTCP()
			if err != nil {
				continue
			}
			conn.SetKeepAlive(true)
			conn.SetKeepAlivePeriod(5 * time.Second)
			conn.SetNoDelay(true)

			go s.handleStudent(conn)
		}

	}()
}

func (s *Server) handleStudent(socket *net.TCPConn) {
	id := s.studentUtil.AddStudent("Unknown")

	go func() {
		defer socket.Close()
		header := make([]byte, HEADER_SIZE)
		data := make([]byte, 0)
		for s.isRunning.Load() {
			if !s.studentUtil.isExists(id) {
				socket.Close()
			}
			_, err := io.ReadFull(socket, header)

			if err != nil {
				s.studentUtil.RemoveStudent(id)
				break
			}

			dataType, dataSize, err := unpackHeader(header)

			if err != nil || dataSize <= 0 || dataSize > 5*1024*1024 {
				continue
			}

			if len(data) < dataSize {
				data = make([]byte, dataSize)
			}
			_, err = io.ReadFull(socket, data[:dataSize])

			if err != nil {
				continue
			}

			switch dataType {
			case 0:
				name := string(data[:dataSize])
				s.studentUtil.UpdateName(id, name)
			case 1:
				msg := string(data[:dataSize])
				println(msg)
			default:
				img, _, err := image.Decode(bytes.NewReader(data[:dataSize]))
				if err == nil {
					s.studentUtil.UpdateImage(id, img) // Check if zero array can be formed

				}
			}
		}
		s.studentUtil.RemoveStudent(id)
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

func unpackHeader(data []byte) (uint16, int, error) {
	if len(data) < HEADER_SIZE || string(data[:2]) != "HE" {
		return 0, 0, errors.New("invalid header")
	}

	status := uint16(binary.BigEndian.Uint16(data[2:4]))
	length := int(binary.BigEndian.Uint32(data[4:8]))

	return status, length, nil
}
