package main

import (
	"bytes"
	"encoding/binary"
	"errors"
	"image"
	"image/draw"
	"image/jpeg"
	"io"
	"log"
	"net"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

const (
	HEADER_SIZE          = 8
	READ_TIMEOUT         = 10 * time.Second
	REMOVAL_GRACE_PERIOD = 5 * time.Second // Keep student visible for a few seconds after disconnect
	FrameTypeKey   byte = 0x01
	FrameTypeDirty byte = 0x02
)

type StudentDecoder struct {
	canvas *image.RGBA
	mu     sync.Mutex
}

type Server struct {
	listener    *net.TCPListener
	isRunning   atomic.Bool
	studentUtil StudentUtil
	activeConns   map[string]int64 // studentID -> connection timestamp
	activeConnsMu sync.Mutex

	decoders   map[string]*StudentDecoder
	decodersMu sync.Mutex
}

type StudentUtil interface {
	AddStudent(id, name string)
	RemoveStudent(id string)
	UpdateImage(id string, img image.Image)
	UpdateName(id string, name string)
	isExists(id string) bool
}

func NewServer() *Server {
	server := Server{
		isRunning:   atomic.Bool{},
		activeConns: make(map[string]int64),
		decoders:    make(map[string]*StudentDecoder),
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

func (s *Server) registerConnection(id string) int64 {
	s.activeConnsMu.Lock()
	defer s.activeConnsMu.Unlock()
	timestamp := time.Now().UnixNano()
	s.activeConns[id] = timestamp
	return timestamp
}

func (s *Server) scheduleStudentRemoval(id string, connTimestamp int64) {
	time.Sleep(REMOVAL_GRACE_PERIOD)

	s.activeConnsMu.Lock()
	defer s.activeConnsMu.Unlock()

	if currentTimestamp, exists := s.activeConns[id]; exists {
		if currentTimestamp == connTimestamp {
			delete(s.activeConns, id)
			s.studentUtil.RemoveStudent(id)
			s.removeDecoder(id)
		}
	}
}

func (s *Server) getOrCreateDecoder(id string) *StudentDecoder {
	s.decodersMu.Lock()
	defer s.decodersMu.Unlock()

	if dec, ok := s.decoders[id]; ok {
		return dec
	}

	dec := &StudentDecoder{}
	s.decoders[id] = dec
	return dec
}

// removeDecoder removes a student's decoder.
func (s *Server) removeDecoder(id string) {
	s.decodersMu.Lock()
	defer s.decodersMu.Unlock()
	delete(s.decoders, id)
}

func (s *Server) handleStudent(socket *net.TCPConn) {
	defer socket.Close()
	id := ""
	var connTimestamp int64 = 0
	header := make([]byte, HEADER_SIZE)

	// Reusable data buffer with larger initial capacity
	data := make([]byte, 64*1024)

	for s.isRunning.Load() {
		socket.SetReadDeadline(time.Now().Add(READ_TIMEOUT))

		_, err := io.ReadFull(socket, header)

		if err != nil {
			break
		}

		dataType, dataSize, err := unpackHeader(header)

		if err != nil || dataSize <= 0 || dataSize > 5*1024*1024 {
			break
		}

		if cap(data) < dataSize {
			data = make([]byte, dataSize)
		}
		data = data[:dataSize]

		socket.SetReadDeadline(time.Now().Add(READ_TIMEOUT))
		_, err = io.ReadFull(socket, data)

		if err != nil {
			break
		}

		switch dataType {
		case 0: // NAME
			info := string(data)
			parts := strings.SplitN(info, "###", 2)
			if len(parts) != 2 {
				break
			}
			id = strings.TrimSpace(parts[0])
			name := strings.TrimSpace(parts[1])

			connTimestamp = s.registerConnection(id)

			if !s.studentUtil.isExists(id) {
				s.studentUtil.AddStudent(id, name)
			} else {
				s.studentUtil.UpdateName(id, name)
			}
		case 1: // MESSAGE
			msg := string(data)
			println(msg)
		default: // PICTURE
			if id == "" {
				continue
			}
			// Decode frame with dirty rect support
			log.Printf("[%s] PICTURE received: %d bytes, first byte: 0x%02x", id, len(data), data[0])
			img := s.decodeFrame(id, data)
			if img != nil {
				log.Printf("[%s] Frame decoded: %dx%d, type=%T", id, img.Bounds().Dx(), img.Bounds().Dy(), img)
				s.studentUtil.UpdateImage(id, img)
			} else {
				log.Printf("[%s] Frame decode returned nil!", id)
			}
		}
	}

	// Schedule student removal with grace period
	// If client reconnects within the grace period, they won't be removed
	if id != "" && connTimestamp != 0 {
		go s.scheduleStudentRemoval(id, connTimestamp)
	}
}

func (s *Server) broadcastHost(port int) {
	address := net.UDPAddr{
		IP:   net.IPv4(255, 255, 255, 255),
		Port: port,
	}

	message := []byte("server")
	for s.isRunning.Load() {
		conn, err := net.DialUDP("udp", nil, &address)
		if err != nil {
			time.Sleep(1 * time.Second)
			continue
		}

		_, err = conn.Write(message)
		conn.Close()

		if err != nil {
			time.Sleep(1 * time.Second)
			continue
		}
		time.Sleep(1 * time.Second)
	}
}

func (s *Server) Stop() {
	s.isRunning.Store(false)
	if s.listener != nil {
		s.listener.Close()
	}
}

func (s *Server) decodeFrame(id string, data []byte) image.Image {
	if len(data) < 1 {
		return nil
	}

	frameType := data[0]

	switch frameType {
	case FrameTypeKey:
		return s.decodeKeyFrame(id, data[1:])
	case FrameTypeDirty:
		return s.decodeDirtyRects(id, data[1:])
	default:
		return s.decodeLegacyFrame(id, data)
	}
}

func (s *Server) decodeKeyFrame(id string, data []byte) image.Image {
	img, err := jpeg.Decode(bytes.NewReader(data))
	if err != nil {
		log.Printf("[%s] keyframe jpeg.Decode failed: %v (data len=%d, first bytes: %x)", id, err, len(data), data[:min(16, len(data))])
		return nil
	}
	log.Printf("[%s] keyframe decoded: %dx%d", id, img.Bounds().Dx(), img.Bounds().Dy())

	dec := s.getOrCreateDecoder(id)
	dec.mu.Lock()
	defer dec.mu.Unlock()

	bounds := img.Bounds()
	rgba := image.NewRGBA(bounds)
	draw.Draw(rgba, bounds, img, bounds.Min, draw.Src)
	dec.canvas = rgba

	return rgba
}

func (s *Server) decodeDirtyRects(id string, data []byte) image.Image {
	if len(data) < 2 {
		return nil
	}

	dec := s.getOrCreateDecoder(id)
	dec.mu.Lock()
	defer dec.mu.Unlock()

	if dec.canvas == nil {
		return nil
	}

	rectCount := int(binary.BigEndian.Uint16(data[:2]))
	offset := 2

	for i := 0; i < rectCount; i++ {
		if offset+12 > len(data) {
			break
		}

		x := int(binary.BigEndian.Uint16(data[offset:]))
		y := int(binary.BigEndian.Uint16(data[offset+2:]))
		w := int(binary.BigEndian.Uint16(data[offset+4:]))
		h := int(binary.BigEndian.Uint16(data[offset+6:]))
		dataLen := int(binary.BigEndian.Uint32(data[offset+8:]))
		offset += 12

		if offset+dataLen > len(data) {
			break
		}

		rectImg, err := jpeg.Decode(bytes.NewReader(data[offset : offset+dataLen]))
		offset += dataLen

		if err != nil {
			continue
		}

		destRect := image.Rect(x, y, x+w, y+h)
		canvasBounds := dec.canvas.Bounds()

		if destRect.Max.X > canvasBounds.Max.X || destRect.Max.Y > canvasBounds.Max.Y {
			continue
		}

		draw.Draw(dec.canvas, destRect, rectImg, image.Point{}, draw.Src)
	}

	return dec.canvas
}

func (s *Server) decodeLegacyFrame(id string, data []byte) image.Image {
	img, err := jpeg.Decode(bytes.NewReader(data))
	if err != nil {
		log.Printf("[%s] legacy jpeg.Decode failed: %v, trying image.Decode", id, err)
		img, _, err = image.Decode(bytes.NewReader(data))
		if err != nil {
			log.Printf("[%s] legacy image.Decode also failed: %v", id, err)
			return nil
		}
	}
	log.Printf("[%s] legacy frame decoded: %dx%d", id, img.Bounds().Dx(), img.Bounds().Dy())

	dec := s.getOrCreateDecoder(id)
	dec.mu.Lock()
	defer dec.mu.Unlock()

	bounds := img.Bounds()
	rgba := image.NewRGBA(bounds)
	draw.Draw(rgba, bounds, img, bounds.Min, draw.Src)
	dec.canvas = rgba

	return rgba
}

func unpackHeader(data []byte) (uint16, int, error) {
	if len(data) < HEADER_SIZE || string(data[:2]) != "HE" {
		return 0, 0, errors.New("invalid header")
	}

	status := uint16(binary.BigEndian.Uint16(data[2:4]))
	length := int(binary.BigEndian.Uint32(data[4:8]))

	return status, length, nil
}
