// Package tftp is a TFTP client (RFC 1350) with option negotiation (RFC 2347)
// and the blksize and tsize options (RFC 2348, RFC 2349): what the SLM-CPU-0
// bootloader serves. Octet mode only.
package tftp

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"
	"syscall"
	"time"
)

const (
	opRRQ   = 1
	opWRQ   = 2
	opDATA  = 3
	opACK   = 4
	opERROR = 5
	opOACK  = 6

	defaultBlksize = 512
	errUnknownTID  = 5
)

// ServerError is an ERROR packet from the server.
type ServerError struct {
	Code uint16
	Msg  string
}

func (e *ServerError) Error() string { return e.Msg }

var (
	// ErrTimeout means the server stopped answering.
	ErrTimeout = errors.New("no answer")
	// ErrUnreachable means the host answered that nothing listens on the
	// port (ICMP port unreachable, which only Windows reports to us).
	ErrUnreachable = errors.New("port unreachable")
	// ErrNoFinalAck means every block went out but the acknowledgement of
	// the last one never came back: the server may or may not have the file.
	ErrNoFinalAck = errors.New("the last block was not acknowledged")

	errRetry = errors.New("retry")
)

// Client talks to one server.
type Client struct {
	Server *net.UDPAddr
	// Retry is how long to wait for an answer before sending again.
	Retry time.Duration
	// Timeout is how long a transfer may go without progress.
	Timeout time.Duration
	// Logf, when set, receives protocol events.
	Logf func(format string, args ...any)
}

// Option is a request option (RFC 2347).
type Option struct{ Name, Value string }

// PutOptions tune a write request.
type PutOptions struct {
	// Blksize requests a block size; 0 leaves the default of 512.
	Blksize int
	// Tsize announces the file size.
	Tsize bool
	// Extra options must come back unchanged if the server acknowledges them.
	Extra []Option
	// Progress is called after each acknowledged block.
	Progress func(done, total int)
}

func (c *Client) logf(format string, args ...any) {
	if c.Logf != nil {
		c.Logf(format, args...)
	}
}

func (c *Client) retry() time.Duration {
	if c.Retry > 0 {
		return c.Retry
	}
	return 500 * time.Millisecond
}

func (c *Client) timeout() time.Duration {
	if c.Timeout > 0 {
		return c.Timeout
	}
	return 10 * time.Second
}

// Get reads a file of at most max bytes.
func (c *Client) Get(ctx context.Context, file string, max int) ([]byte, error) {
	s, err := c.open()
	if err != nil {
		return nil, err
	}
	defer s.close()

	var out []byte
	expect := uint16(1)
	if err := s.send(request(opRRQ, file, nil)); err != nil {
		return nil, err
	}
	progress := time.Now()
	for {
		p, err := s.recv(ctx)
		if err == errRetry {
			if time.Since(progress) >= c.timeout() {
				return nil, ErrTimeout
			}
			s.resend()
			continue
		}
		if err != nil {
			return nil, err
		}
		switch opcode(p) {
		case opERROR:
			return nil, parseError(p)
		case opDATA:
			blk := binary.BigEndian.Uint16(p[2:])
			if blk != expect {
				continue
			}
			data := p[4:]
			if len(out)+len(data) > max {
				s.abort("file too large")
				return nil, fmt.Errorf("%s is larger than %d bytes", file, max)
			}
			out = append(out, data...)
			if err := s.send(ack(blk)); err != nil {
				return nil, err
			}
			progress = time.Now()
			if len(data) < defaultBlksize {
				return out, nil
			}
			expect++
		}
	}
}

// Put writes a file.
func (c *Client) Put(ctx context.Context, file string, data []byte, o PutOptions) (err error) {
	var opts []Option
	if o.Blksize > 0 {
		opts = append(opts, Option{"blksize", strconv.Itoa(o.Blksize)})
	}
	if o.Tsize {
		opts = append(opts, Option{"tsize", strconv.Itoa(len(data))})
	}
	opts = append(opts, o.Extra...)

	s, err := c.open()
	if err != nil {
		return err
	}
	defer s.close()
	defer func() {
		// After the last block the server may be done, so there is nothing
		// to abandon.
		var se *ServerError
		if err != nil && !errors.As(err, &se) && !errors.Is(err, ErrNoFinalAck) {
			s.abort("transfer abandoned by the client")
		}
	}()

	if err := s.send(request(opWRQ, file, opts)); err != nil {
		return err
	}
	blksize, err := c.awaitAccept(ctx, s, opts, len(data), o.Blksize)
	if err != nil {
		return err
	}
	c.logf("transfer accepted, block size %d", blksize)

	total := len(data)
	nblocks := total/blksize + 1 // the last block is short, possibly empty
	blk := uint16(0)
	for i := 0; i < nblocks; i++ {
		blk++
		start := i * blksize
		end := min(start+blksize, total)
		if err := s.send(dataPacket(blk, data[start:end])); err != nil {
			return err
		}
		sent := time.Now()
		for acked := false; !acked; {
			p, err := s.recv(ctx)
			if err == errRetry {
				if time.Since(sent) >= c.timeout() {
					if i == nblocks-1 {
						return ErrNoFinalAck
					}
					return ErrTimeout
				}
				c.logf("block %d: no acknowledgement yet, sending it again", blk)
				s.resend()
				continue
			}
			if err != nil {
				return err
			}
			switch opcode(p) {
			case opERROR:
				return parseError(p)
			case opACK:
				// Acknowledgements of earlier blocks are duplicates; answering
				// them would double every later packet (RFC 1123 4.2.3.1).
				acked = binary.BigEndian.Uint16(p[2:]) == blk
			}
		}
		if o.Progress != nil {
			o.Progress(end, total)
		}
	}
	return nil
}

// awaitAccept waits for the answer to a write request and returns the block
// size to use.
func (c *Client) awaitAccept(ctx context.Context, s *session, sent []Option, size, blksize int) (int, error) {
	start := time.Now()
	for {
		p, err := s.recv(ctx)
		if err == errRetry {
			if time.Since(start) >= c.timeout() {
				return 0, ErrTimeout
			}
			c.logf("no answer to the write request yet, sending it again")
			s.resend()
			continue
		}
		if err != nil {
			return 0, err
		}
		switch opcode(p) {
		case opERROR:
			return 0, parseError(p)
		case opACK:
			if binary.BigEndian.Uint16(p[2:]) == 0 {
				return defaultBlksize, nil // the server ignored our options
			}
		case opOACK:
			got := parseOptions(p[2:])
			n := defaultBlksize
			if v, ok := got["blksize"]; ok {
				b, err := strconv.Atoi(v)
				if err != nil || b < 8 || (blksize > 0 && b > blksize) {
					return 0, fmt.Errorf("server answered with an invalid block size %q", v)
				}
				n = b
			}
			if v, ok := got["tsize"]; ok && v != strconv.Itoa(size) {
				return 0, fmt.Errorf("server answered tsize %s for a %d byte file", v, size)
			}
			for _, o := range sent {
				if o.Name == "blksize" || o.Name == "tsize" {
					continue
				}
				if v, ok := got[strings.ToLower(o.Name)]; ok && !strings.EqualFold(v, o.Value) {
					return 0, fmt.Errorf("server changed option %s from %s to %s", o.Name, o.Value, v)
				}
			}
			return n, nil
		}
	}
}

// ---- Transport -----------------------------------------------------------

type session struct {
	c    *Client
	conn *net.UDPConn
	peer *net.UDPAddr // the server's transfer ID, once known
	last []byte       // packet to repeat on timeout
	due  time.Time    // when to repeat it
	buf  []byte
}

func (c *Client) open() (*session, error) {
	conn, err := net.ListenUDP("udp4", nil)
	if err != nil {
		return nil, err
	}
	return &session{c: c, conn: conn, buf: make([]byte, 65536)}, nil
}

func (s *session) close() { s.conn.Close() }

// send transmits p to the server's transfer port, or to its request port
// before the server has answered.
func (s *session) send(p []byte) error {
	to := s.peer
	if to == nil {
		to = s.c.Server
	}
	s.last = p
	s.due = time.Now().Add(s.c.retry())
	_, err := s.conn.WriteToUDP(p, to)
	return err
}

func (s *session) resend() { s.send(s.last) }

func (s *session) abort(msg string) {
	if s.peer != nil {
		s.conn.WriteToUDP(errorPacket(0, msg), s.peer)
	}
}

// recv returns the next packet from the server, or errRetry once the packet
// last sent is due to be repeated.
func (s *session) recv(ctx context.Context) ([]byte, error) {
	for {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		deadline := s.due
		if d, ok := ctx.Deadline(); ok && d.Before(deadline) {
			deadline = d
		}
		s.conn.SetReadDeadline(deadline)
		n, from, err := s.conn.ReadFromUDP(s.buf)
		if err != nil {
			var ne net.Error
			if errors.As(err, &ne) && ne.Timeout() {
				if err := ctx.Err(); err != nil {
					return nil, err
				}
				return nil, errRetry
			}
			if unreachable(err) {
				if s.peer == nil {
					return nil, ErrUnreachable
				}
				continue
			}
			return nil, err
		}
		if !from.IP.Equal(s.c.Server.IP) || n < 4 {
			continue
		}
		p := s.buf[:n]
		switch {
		case s.peer == nil && opcode(p) == opERROR:
			// Refusals can come from the request port or a new one.
		case s.peer == nil:
			s.peer = &net.UDPAddr{IP: from.IP, Port: from.Port}
		case from.Port != s.peer.Port:
			s.conn.WriteToUDP(errorPacket(errUnknownTID, "unknown transfer ID"), from)
			continue
		}
		return p, nil
	}
}

// unreachable reports an ICMP error for an earlier datagram. Windows returns
// these (WSAECONNRESET) from reads on unconnected UDP sockets.
func unreachable(err error) bool {
	var errno syscall.Errno
	if !errors.As(err, &errno) {
		return false
	}
	return errno == syscall.ECONNREFUSED || errno == syscall.ECONNRESET || errno == 10054
}

// ---- Packets ---------------------------------------------------------------

func opcode(p []byte) uint16 { return binary.BigEndian.Uint16(p) }

func request(op uint16, file string, opts []Option) []byte {
	p := binary.BigEndian.AppendUint16(nil, op)
	p = append(append(p, file...), 0)
	p = append(append(p, "octet"...), 0)
	for _, o := range opts {
		p = append(append(p, o.Name...), 0)
		p = append(append(p, o.Value...), 0)
	}
	return p
}

func ack(blk uint16) []byte {
	return binary.BigEndian.AppendUint16(binary.BigEndian.AppendUint16(nil, opACK), blk)
}

func dataPacket(blk uint16, data []byte) []byte {
	p := binary.BigEndian.AppendUint16(nil, opDATA)
	p = binary.BigEndian.AppendUint16(p, blk)
	return append(p, data...)
}

func errorPacket(code uint16, msg string) []byte {
	p := binary.BigEndian.AppendUint16(nil, opERROR)
	p = binary.BigEndian.AppendUint16(p, code)
	return append(append(p, msg...), 0)
}

func parseError(p []byte) error {
	msg := string(p[4:])
	if i := strings.IndexByte(msg, 0); i >= 0 {
		msg = msg[:i]
	}
	return &ServerError{Code: binary.BigEndian.Uint16(p[2:]), Msg: msg}
}

func parseOptions(b []byte) map[string]string {
	fields := strings.Split(string(b), "\x00")
	out := map[string]string{}
	for i := 0; i+1 < len(fields); i += 2 {
		out[strings.ToLower(fields[i])] = fields[i+1]
	}
	return out
}
