package boot

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/Autonomy-Logic/slm-cpu-0-arduino-core/tools/slm-cpu-0-netflash/internal/fakeboard"
)

func TestParseInfo(t *testing.T) {
	info := ParseInfo([]byte("device=SLM-MX\nmode=bootloader\r\nmac=02:00:00:00:00:01\nnot a pair\n\nip=1.2.3.4"))
	want := map[string]string{"device": "SLM-MX", "mode": "bootloader", "mac": "02:00:00:00:00:01", "ip": "1.2.3.4"}
	if len(info) != len(want) {
		t.Fatalf("parsed %v", info)
	}
	for k, v := range want {
		if info[k] != v {
			t.Errorf("%s = %q, want %q", k, info[k], v)
		}
	}
}

func TestFromBootloader(t *testing.T) {
	cases := []struct {
		device, mode string
		ok           bool
	}{
		{"SLM-MX", "bootloader", true},
		{"SLM-CPU-0", "bootloader", true},
		{"SLM-CPU-0", "runtime", false},
		{"SLM-MX", "", false},
		{"SLM-RP4", "bootloader", false},
		{"slm-cpu-0", "bootloader", false},
		{"", "bootloader", false},
	}
	for _, c := range cases {
		if got := (Info{"device": c.device, "mode": c.mode}).FromBootloader(); got != c.ok {
			t.Errorf("device=%q mode=%q: %v, want %v", c.device, c.mode, got, c.ok)
		}
	}
}

func TestReadInfo(t *testing.T) {
	for _, c := range []struct {
		device string
		ok     bool
	}{{"SLM-MX", true}, {"SLM-CPU-0", true}, {"SLM-XX", false}} {
		t.Run(c.device, func(t *testing.T) {
			b := fakeboard.Start(t, fakeboard.Options{Device: c.device, StartInBootloader: true})
			target := Target{Addr: &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: b.TFTPPort}}
			info, err := target.ReadInfo(context.Background(), time.Second)
			if (err == nil) != c.ok {
				t.Fatalf("ReadInfo = %v, %v", info, err)
			}
			if c.ok && info["device"] != c.device {
				t.Errorf("device %q", info["device"])
			}
		})
	}
}
