package modbus

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"io"
	"net"
	"testing"
	"time"
)

// serve answers requests on one connection like modbus_tcp.cpp: the MBAP
// header is echoed with the reply length.
func serve(t *testing.T, answer func(pdu []byte) []byte) (string, <-chan []byte) {
	t.Helper()
	ln, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { ln.Close() })
	seen := make(chan []byte, 8)
	go func() {
		c, err := ln.Accept()
		if err != nil {
			return
		}
		defer c.Close()
		for {
			var hdr [7]byte
			if _, err := io.ReadFull(c, hdr[:]); err != nil {
				return
			}
			pdu := make([]byte, int(binary.BigEndian.Uint16(hdr[4:]))-1)
			if _, err := io.ReadFull(c, pdu); err != nil {
				return
			}
			seen <- append(hdr[:], pdu...)
			reply := answer(pdu)
			binary.BigEndian.PutUint16(hdr[4:], uint16(1+len(reply)))
			c.Write(append(hdr[:], reply...))
		}
	}()
	return ln.Addr().String(), seen
}

func dial(t *testing.T, addr string) *Conn {
	t.Helper()
	c, err := Dial(context.Background(), addr, 2*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { c.Close() })
	return c
}

func TestRebootFrame(t *testing.T) {
	addr, seen := serve(t, func(pdu []byte) []byte { return []byte{pdu[0], StatusSuccess} })
	c := dial(t, addr)
	if err := c.RebootToBootloader(); err != nil {
		t.Fatal(err)
	}
	// MBAP: transaction 1, protocol 0, length 6, unit 0; then 4C and the magic.
	want := []byte{0, 1, 0, 0, 0, 6, 0, 0x4C, 0xB0, 0x07, 0x10, 0xAD}
	if got := <-seen; !bytes.Equal(got, want) {
		t.Fatalf("request % X, want % X", got, want)
	}
}

func TestRebootRefusals(t *testing.T) {
	status := []byte{StatusRefusedLocked, StatusOutOfBounds}
	addr, _ := serve(t, func(pdu []byte) []byte {
		s := status[0]
		status = status[1:]
		return []byte{pdu[0], s}
	})
	c := dial(t, addr)
	if err := c.RebootToBootloader(); !errors.Is(err, ErrLocked) {
		t.Errorf("locked: %v", err)
	}
	if err := c.RebootToBootloader(); err == nil {
		t.Error("out of bounds status accepted")
	}
}

func TestVersionAndDeviceID(t *testing.T) {
	addr, _ := serve(t, func(pdu []byte) []byte {
		switch pdu[0] {
		case FCGetVersion:
			return append([]byte{FCGetVersion, StatusSuccess}, "4.2.7"...)
		case FCGetDeviceID:
			return []byte{FCGetDeviceID, StatusSuccess, 0}
		}
		return []byte{pdu[0] | 0x80, ExIllegalFunction}
	})
	c := dial(t, addr)
	v, err := c.RuntimeVersion()
	if err != nil || v != "4.2.7" {
		t.Errorf("version %q, %v", v, err)
	}
	id, err := c.DeviceID()
	if err != nil || len(id) != 0 {
		t.Errorf("device id % X, %v", id, err)
	}
	_, err = c.Request(0x41, nil)
	var ex *ExceptionError
	if !errors.As(err, &ex) || ex.Code != ExIllegalFunction {
		t.Errorf("exception: %v", err)
	}
}
