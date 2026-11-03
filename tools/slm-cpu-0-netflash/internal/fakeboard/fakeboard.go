// Package fakeboard is a test double for an SLM-CPU-0 on local sockets: the
// running OpenPLC program on Modbus TCP and the network bootloader on TFTP,
// following the contract in docs/bootloader-contract.md.
package fakeboard

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Autonomy-Logic/slm-cpu-0-arduino-core/tools/slm-cpu-0-netflash/internal/layout"
)

// Options shape the board.
type Options struct {
	// Device is the info file's device value; empty means "SLM-MX".
	Device string
	// StartInBootloader starts with no program running.
	StartInBootloader bool
	// DropEvery drops every n-th TFTP packet the board receives; 0 drops none.
	DropEvery int
	// Runtime is the version the program reports; empty means "4.2.7".
	Runtime string
}

// Commit is an image the bootloader accepted.
type Commit struct {
	Size  int
	CRC32 uint32
}

// Board is a running fake.
type Board struct {
	TFTPPort, ModbusPort int
	// Commits receives every accepted image.
	Commits chan Commit

	opt        Options
	inBoot     atomic.Bool
	udp        *net.UDPConn
	tcp        net.Listener
	mu         sync.Mutex
	received   int
	peer       *net.UDPAddr
	blksize    int
	block      uint16
	want       int
	wantCRC    uint32
	haveCRC    bool
	image      []byte
	lastAnswer []byte
}

// Start runs a board until the test ends.
func Start(t *testing.T, opt Options) *Board {
	t.Helper()
	if opt.Device == "" {
		opt.Device = "SLM-MX"
	}
	if opt.Runtime == "" {
		opt.Runtime = "4.2.7"
	}
	udp, err := net.ListenUDP("udp4", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1)})
	if err != nil {
		t.Fatal(err)
	}
	tcp, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	b := &Board{
		TFTPPort:   udp.LocalAddr().(*net.UDPAddr).Port,
		ModbusPort: tcp.Addr().(*net.TCPAddr).Port,
		Commits:    make(chan Commit, 4),
		opt:        opt,
		udp:        udp,
		tcp:        tcp,
	}
	b.inBoot.Store(opt.StartInBootloader)
	t.Cleanup(func() {
		udp.Close()
		tcp.Close()
	})
	go b.serveTFTP()
	go b.serveModbus()
	return b
}

// ---- Bootloader (TFTP) ------------------------------------------------------

const (
	opRRQ, opWRQ, opDATA, opACK, opERROR, opOACK = 1, 2, 3, 4, 5, 6
)

func (b *Board) serveTFTP() {
	buf := make([]byte, 65536)
	for {
		n, from, err := b.udp.ReadFromUDP(buf)
		if err != nil {
			return
		}
		if !b.inBoot.Load() || n < 4 {
			continue
		}
		b.mu.Lock()
		b.received++
		drop := b.opt.DropEvery > 0 && b.received%b.opt.DropEvery == 0
		if !drop {
			b.handle(from, append([]byte(nil), buf[:n]...))
		}
		b.mu.Unlock()
	}
}

func (b *Board) send(to *net.UDPAddr, p []byte) {
	b.lastAnswer = p
	b.udp.WriteToUDP(p, to)
}

func (b *Board) fail(to *net.UDPAddr, code uint16, msg string) {
	p := binary.BigEndian.AppendUint16(nil, opERROR)
	p = binary.BigEndian.AppendUint16(p, code)
	b.udp.WriteToUDP(append(append(p, msg...), 0), to)
	b.peer = nil
}

func (b *Board) handle(from *net.UDPAddr, p []byte) {
	op := binary.BigEndian.Uint16(p)
	switch op {
	case opRRQ, opWRQ:
		fields := strings.Split(string(p[2:]), "\x00")
		if len(fields) < 2 || !strings.EqualFold(fields[1], "octet") {
			b.fail(from, 4, "only octet mode is supported")
			return
		}
		opts := map[string]string{}
		for i := 2; i+1 < len(fields); i += 2 {
			opts[strings.ToLower(fields[i])] = fields[i+1]
		}
		file := strings.ToLower(fields[0])
		switch {
		case op == opRRQ && file == layout.FileInfo:
			info := fmt.Sprintf("device=%s\nmode=bootloader\nbootloader=test\nentry=request\nmac=02:00:00:00:00:01\napp=valid\n",
				b.opt.Device)
			b.send(from, append([]byte{0, opDATA, 0, 1}, info...))
		case op == opWRQ && file == layout.FileFirmware:
			b.acceptWrite(from, opts)
		case op == opRRQ:
			b.fail(from, 1, "no such file")
		default:
			b.fail(from, 2, "only firmware.bin and identity.bin can be written")
		}
	case opDATA:
		if b.peer == nil || from.Port != b.peer.Port {
			return
		}
		b.data(from, binary.BigEndian.Uint16(p[2:]), p[4:])
	}
}

func (b *Board) acceptWrite(from *net.UDPAddr, opts map[string]string) {
	b.peer, b.block, b.image, b.blksize = from, 0, nil, 512
	b.want, b.haveCRC = -1, false
	if v, ok := opts["blksize"]; ok {
		n, _ := strconv.Atoi(v)
		b.blksize = min(max(n, 8), layout.MaxBlksize)
	}
	if v, ok := opts["tsize"]; ok {
		b.want, _ = strconv.Atoi(v)
	}
	if v, ok := opts[layout.OptCRC32]; ok {
		c, err := strconv.ParseUint(v, 16, 32)
		b.wantCRC, b.haveCRC = uint32(c), err == nil
	}
	if len(opts) == 0 {
		b.send(from, []byte{0, opACK, 0, 0})
		return
	}
	oack := []byte{0, opOACK}
	put := func(k, v string) { oack = append(append(append(append(oack, k...), 0), v...), 0) }
	if _, ok := opts["blksize"]; ok {
		put("blksize", strconv.Itoa(b.blksize))
	}
	if b.want >= 0 {
		put("tsize", strconv.Itoa(b.want))
	}
	if b.haveCRC {
		put(layout.OptCRC32, fmt.Sprintf("%08x", b.wantCRC))
	}
	b.send(from, oack)
}

func (b *Board) data(from *net.UDPAddr, block uint16, d []byte) {
	if block == b.block {
		b.udp.WriteToUDP(b.lastAnswer, from) // our acknowledgement was lost
		return
	}
	if block != b.block+1 {
		return
	}
	if len(d) > b.blksize {
		b.fail(from, 4, "block larger than negotiated blksize")
		return
	}
	b.image = append(b.image, d...)
	b.block = block
	ack := binary.BigEndian.AppendUint16([]byte{0, opACK}, block)
	if len(d) == b.blksize {
		b.send(from, ack)
		return
	}
	switch {
	case b.want >= 0 && len(b.image) != b.want:
		b.fail(from, 0, "image size does not match tsize")
	case b.haveCRC && layout.CRC32(b.image) != b.wantCRC:
		b.fail(from, 0, "image CRC does not match the crc32 option")
	case layout.CheckImage(b.image) != nil:
		b.fail(from, 0, "not an SLM-CPU-0 application: bad vector table")
	default:
		b.send(from, ack)
		b.Commits <- Commit{Size: len(b.image), CRC32: layout.CRC32(b.image)}
		go func() {
			time.Sleep(200 * time.Millisecond) // dally, then reset into the new program
			b.inBoot.Store(false)
		}()
	}
}

// ---- Program and bootloader (Modbus TCP) ------------------------------------

var rebootMagic = []byte{0xB0, 0x07, 0x10, 0xAD}

func (b *Board) serveModbus() {
	for {
		c, err := b.tcp.Accept()
		if err != nil {
			return
		}
		go b.modbusConn(c)
	}
}

func (b *Board) modbusConn(c net.Conn) {
	defer c.Close()
	for {
		var hdr [7]byte
		if _, err := io.ReadFull(c, hdr[:]); err != nil {
			return
		}
		pdu := make([]byte, int(binary.BigEndian.Uint16(hdr[4:]))-1)
		if _, err := io.ReadFull(c, pdu); err != nil || len(pdu) == 0 {
			return
		}
		reply, reboot := b.answer(pdu)
		binary.BigEndian.PutUint16(hdr[4:], uint16(1+len(reply)))
		c.Write(append(hdr[:], reply...))
		if reboot {
			time.Sleep(100 * time.Millisecond)
			b.inBoot.Store(true)
			return
		}
	}
}

func (b *Board) answer(pdu []byte) (reply []byte, reboot bool) {
	fc := pdu[0]
	if fc == 0x4C {
		if !bytes.Equal(pdu[1:], rebootMagic) {
			return []byte{fc, 0x81}, false
		}
		return []byte{fc, 0x7E}, !b.inBoot.Load()
	}
	if b.inBoot.Load() {
		return []byte{fc | 0x80, 0x01}, false
	}
	switch fc {
	case 0x47:
		return append([]byte{fc, 0x7E}, b.opt.Runtime...), false
	case 0x48:
		return []byte{fc, 0x7E, 0}, false
	}
	return []byte{fc | 0x80, 0x01}, false
}
