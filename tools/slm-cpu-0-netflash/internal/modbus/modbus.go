// Package modbus sends the OpenPLC runtime's editor function codes over Modbus
// TCP: enough to hand an SLM-CPU-0 to its bootloader and to see that a new
// program runs. Frames are the ones the OpenPLC Editor sends (unit id 0).
package modbus

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"time"
)

const (
	FCGetVersion       = 0x47
	FCGetDeviceID      = 0x48
	FCRebootBootloader = 0x4C

	StatusSuccess       = 0x7E
	StatusOutOfBounds   = 0x81
	StatusRefusedLocked = 0x6C

	ExIllegalFunction = 0x01
)

// RebootMagic must follow function code 0x4C.
var RebootMagic = []byte{0xB0, 0x07, 0x10, 0xAD}

// ErrLocked means the runtime refused because its programming lock is on.
var ErrLocked = errors.New("the PLC's programming lock is engaged; release it at the device and try again")

// ExceptionError is a Modbus exception reply.
type ExceptionError struct {
	FC, Code byte
}

func (e *ExceptionError) Error() string {
	return fmt.Sprintf("function code 0x%02X answered with Modbus exception %d", e.FC, e.Code)
}

// Conn is a Modbus TCP connection to one device.
type Conn struct {
	conn    net.Conn
	tid     uint16
	timeout time.Duration
}

// Dial connects to addr (host:port). timeout bounds the connection and each
// request.
func Dial(ctx context.Context, addr string, timeout time.Duration) (*Conn, error) {
	d := net.Dialer{Timeout: timeout}
	c, err := d.DialContext(ctx, "tcp4", addr)
	if err != nil {
		return nil, err
	}
	return &Conn{conn: c, timeout: timeout}, nil
}

func (c *Conn) Close() error { return c.conn.Close() }

// Request sends one PDU and returns the reply PDU, function code first.
func (c *Conn) Request(fc byte, payload []byte) ([]byte, error) {
	c.tid++
	req := make([]byte, 8+len(payload))
	binary.BigEndian.PutUint16(req[0:], c.tid)
	binary.BigEndian.PutUint16(req[4:], uint16(2+len(payload)))
	req[7] = fc
	copy(req[8:], payload)
	c.conn.SetDeadline(time.Now().Add(c.timeout))
	// One write, so the request arrives in one segment: the runtime reads
	// the header and the frame from whatever has arrived.
	if _, err := c.conn.Write(req); err != nil {
		return nil, err
	}
	for {
		var hdr [7]byte
		if _, err := io.ReadFull(c.conn, hdr[:]); err != nil {
			return nil, err
		}
		n := int(binary.BigEndian.Uint16(hdr[4:]))
		if binary.BigEndian.Uint16(hdr[2:]) != 0 || n < 2 || n > 260 {
			return nil, errors.New("malformed Modbus TCP reply")
		}
		pdu := make([]byte, n-1)
		if _, err := io.ReadFull(c.conn, pdu); err != nil {
			return nil, err
		}
		if binary.BigEndian.Uint16(hdr[0:]) != c.tid {
			continue // reply to an earlier request
		}
		switch pdu[0] {
		case fc:
			return pdu, nil
		case fc | 0x80:
			code := byte(0)
			if len(pdu) > 1 {
				code = pdu[1]
			}
			return nil, &ExceptionError{FC: fc, Code: code}
		default:
			return nil, fmt.Errorf("function code 0x%02X answered a 0x%02X request", pdu[0], fc)
		}
	}
}

// debugRequest sends one of the runtime's debug function codes and checks the
// status byte that starts every answer.
func (c *Conn) debugRequest(fc byte, payload []byte) ([]byte, error) {
	pdu, err := c.Request(fc, payload)
	if err != nil {
		return nil, err
	}
	if len(pdu) < 2 {
		return nil, fmt.Errorf("function code 0x%02X: reply too short", fc)
	}
	switch pdu[1] {
	case StatusSuccess:
		return pdu[2:], nil
	case StatusRefusedLocked:
		return nil, ErrLocked
	default:
		return nil, fmt.Errorf("function code 0x%02X answered with status 0x%02X", fc, pdu[1])
	}
}

// RebootToBootloader asks the runtime to reset into the board's bootloader.
// The runtime answers first and resets afterwards.
func (c *Conn) RebootToBootloader() error {
	_, err := c.debugRequest(FCRebootBootloader, RebootMagic)
	return err
}

// RuntimeVersion returns the OpenPLC runtime version, e.g. "4.2.7".
func (c *Conn) RuntimeVersion() (string, error) {
	v, err := c.debugRequest(FCGetVersion, nil)
	return string(v), err
}

// DeviceID returns the board's licence identity; empty on boards without one.
// A successful answer is what the editor takes as proof that firmware runs.
func (c *Conn) DeviceID() ([]byte, error) {
	v, err := c.debugRequest(FCGetDeviceID, nil)
	if err != nil {
		return nil, err
	}
	if len(v) < 1 || len(v) < 1+int(v[0]) {
		return nil, errors.New("function code 0x48: malformed reply")
	}
	return v[1 : 1+int(v[0])], nil
}
