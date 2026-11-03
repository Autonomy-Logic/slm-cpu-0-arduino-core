// Package boot talks to the SLM-CPU-0 network bootloader over TFTP. The
// contract is described in docs/bootloader-contract.md.
package boot

import (
	"context"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/Autonomy-Logic/slm-cpu-0-arduino-core/tools/slm-cpu-0-netflash/internal/layout"
	"github.com/Autonomy-Logic/slm-cpu-0-arduino-core/tools/slm-cpu-0-netflash/internal/tftp"
)

// Devices are the "device" values an SLM-CPU-0 bootloader reports in its info
// file. Current factory bootloaders report "SLM-MX".
var Devices = []string{"SLM-CPU-0", "SLM-MX"}

// Target is a bootloader's TFTP request port.
type Target struct {
	Addr *net.UDPAddr
	// Logf, when set, receives protocol events.
	Logf func(format string, args ...any)
}

// Info is the bootloader's "info" file: device, mode, bootloader version,
// entry reason, mac, ip, app state and so on.
type Info map[string]string

// ParseInfo reads key=value lines.
func ParseInfo(b []byte) Info {
	info := Info{}
	for _, line := range strings.Split(string(b), "\n") {
		if k, v, ok := strings.Cut(strings.TrimSpace(line), "="); ok {
			info[k] = v
		}
	}
	return info
}

// FromBootloader reports whether info was written by an SLM-CPU-0 bootloader.
func (info Info) FromBootloader() bool {
	if info["mode"] != "bootloader" {
		return false
	}
	for _, d := range Devices {
		if info["device"] == d {
			return true
		}
	}
	return false
}

// ReadInfo asks the bootloader for its info file, waiting at most wait.
func (t Target) ReadInfo(ctx context.Context, wait time.Duration) (Info, error) {
	c := &tftp.Client{Server: t.Addr, Retry: min(wait/2, 500*time.Millisecond), Timeout: wait, Logf: t.Logf}
	b, err := c.Get(ctx, layout.FileInfo, 4096)
	if err != nil {
		return nil, err
	}
	info := ParseInfo(b)
	if !info.FromBootloader() {
		return nil, fmt.Errorf("the info file is not from an SLM-CPU-0 bootloader (device=%q, mode=%q)",
			info["device"], info["mode"])
	}
	return info, nil
}

// WriteFirmware uploads an application image. The bootloader checks its size
// and CRC-32, writes the vector table last and then starts it.
func (t Target) WriteFirmware(ctx context.Context, image []byte, progress func(done, total int)) error {
	c := &tftp.Client{Server: t.Addr, Retry: 500 * time.Millisecond, Timeout: 10 * time.Second, Logf: t.Logf}
	return c.Put(ctx, layout.FileFirmware, image, tftp.PutOptions{
		Blksize:  layout.MaxBlksize,
		Tsize:    true,
		Extra:    []tftp.Option{{Name: layout.OptCRC32, Value: fmt.Sprintf("%08x", layout.CRC32(image))}},
		Progress: progress,
	})
}
