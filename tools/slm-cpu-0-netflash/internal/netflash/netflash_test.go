package netflash

import (
	"bytes"
	"context"
	"encoding/binary"
	"math/rand"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/Autonomy-Logic/slm-cpu-0-arduino-core/tools/slm-cpu-0-netflash/internal/fakeboard"
	"github.com/Autonomy-Logic/slm-cpu-0-arduino-core/tools/slm-cpu-0-netflash/internal/layout"
)

func makeImage(n int) []byte {
	b := make([]byte, n)
	rand.New(rand.NewSource(int64(n))).Read(b)
	binary.LittleEndian.PutUint32(b, layout.RAMEnd)
	binary.LittleEndian.PutUint32(b[4:], layout.AppBase+0x189)
	return b
}

func freePort(t *testing.T, network string) int {
	t.Helper()
	if network == "udp4" {
		c, err := net.ListenPacket("udp4", "127.0.0.1:0")
		if err != nil {
			t.Fatal(err)
		}
		defer c.Close()
		return c.LocalAddr().(*net.UDPAddr).Port
	}
	l, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port
}

func upload(t *testing.T, b *fakeboard.Board, img []byte, bootWait time.Duration) (string, error) {
	t.Helper()
	var out bytes.Buffer
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	err := Run(ctx, Options{
		IP:         net.IPv4(127, 0, 0, 1),
		TFTPPort:   b.TFTPPort,
		ModbusPort: b.ModbusPort,
		Image:      img,
		Verify:     true,
		BootWait:   bootWait,
		VerifyWait: 5 * time.Second,
		Out:        &out,
		Verbose:    true,
	})
	return out.String(), err
}

func TestUpload(t *testing.T) {
	cases := []struct {
		name string
		opt  fakeboard.Options
		size int
	}{
		{"from the running program, bootloader reports SLM-MX", fakeboard.Options{Device: "SLM-MX"}, 150001},
		{"from the running program, bootloader reports SLM-CPU-0", fakeboard.Options{Device: "SLM-CPU-0"}, 150001},
		{"board waiting in its bootloader, empty last block", fakeboard.Options{StartInBootloader: true}, layout.MaxBlksize * 40},
		{"lossy link", fakeboard.Options{DropEvery: 9}, 40000},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			b := fakeboard.Start(t, c.opt)
			img := makeImage(c.size)
			out, err := upload(t, b, img, 5*time.Second)
			if err != nil {
				t.Fatalf("%v\n%s", err, out)
			}
			select {
			case got := <-b.Commits:
				if want := (fakeboard.Commit{Size: len(img), CRC32: layout.CRC32(img)}); got != want {
					t.Errorf("committed %+v, want %+v", got, want)
				}
			default:
				t.Fatalf("the board committed nothing\n%s", out)
			}
			if !strings.Contains(out, "OpenPLC runtime 4.2.7 is running") {
				t.Errorf("output:\n%s", out)
			}
		})
	}
}

func TestRejectsUnknownBootloader(t *testing.T) {
	b := fakeboard.Start(t, fakeboard.Options{Device: "SLM-XX", StartInBootloader: true})
	out, err := upload(t, b, makeImage(4096), 2*time.Second)
	if err == nil {
		t.Fatalf("upload to a foreign bootloader succeeded\n%s", out)
	}
	select {
	case c := <-b.Commits:
		t.Fatalf("the board committed %+v", c)
	default:
	}
}

func TestNothingAnswers(t *testing.T) {
	var out bytes.Buffer
	err := Run(context.Background(), Options{
		IP:         net.IPv4(127, 0, 0, 1),
		TFTPPort:   freePort(t, "udp4"),
		ModbusPort: freePort(t, "tcp4"),
		Image:      makeImage(4096),
		Out:        &out,
	})
	if err == nil || !strings.Contains(err.Error(), "no SLM-CPU-0 answered") {
		t.Fatalf("err = %v", err)
	}
}

func TestRejectsForeignImage(t *testing.T) {
	img := makeImage(4096)
	binary.LittleEndian.PutUint32(img[4:], 0x08000189) // linked for address 0
	var out bytes.Buffer
	if err := Run(context.Background(), Options{IP: net.IPv4(127, 0, 0, 1), Image: img, Out: &out}); err == nil {
		t.Fatal("image for another flash offset accepted")
	}
	if out.Len() != 0 {
		t.Errorf("contacted the board before checking the image:\n%s", out.String())
	}
}
