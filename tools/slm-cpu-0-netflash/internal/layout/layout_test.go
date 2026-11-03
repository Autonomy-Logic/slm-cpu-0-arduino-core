package layout

import (
	"encoding/binary"
	"os"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

// bootShared is the core's copy of the header the bootloader is built from.
const bootShared = "../../../../overlay/variants/SLM_CPU_0/boot_shared.h"

// defines returns the literal #define values in boot_shared.h.
func defines(t *testing.T) map[string]string {
	t.Helper()
	src, err := os.ReadFile(bootShared)
	if err != nil {
		t.Fatal(err)
	}
	re := regexp.MustCompile(`(?m)^#define\s+(SLM_CPU_0_\w+)\s+("[^"]*"|0x[0-9A-Fa-f]+u?|\d+u?)(?:\s|$)`)
	out := map[string]string{}
	for _, m := range re.FindAllStringSubmatch(string(src), -1) {
		out[m[1]] = m[2]
	}
	return out
}

func TestMatchesBootShared(t *testing.T) {
	defs := defines(t)
	nums := map[string]uint64{
		"SLM_CPU_0_FLASH_BASE":       FlashBase,
		"SLM_CPU_0_FLASH_END":        FlashEnd,
		"SLM_CPU_0_APP_BASE":         AppBase,
		"SLM_CPU_0_RAM_BASE":         RAMBase,
		"SLM_CPU_0_RAM_END":          RAMEnd,
		"SLM_CPU_0_TFTP_PORT":        TFTPPort,
		"SLM_CPU_0_TFTP_MAX_BLKSIZE": MaxBlksize,
	}
	for name, want := range nums {
		v, ok := defs[name]
		if !ok {
			t.Errorf("%s not found in boot_shared.h", name)
			continue
		}
		got, err := strconv.ParseUint(strings.TrimSuffix(v, "u"), 0, 64)
		if err != nil || got != want {
			t.Errorf("%s = %s in boot_shared.h, layout.go has 0x%X", name, v, want)
		}
	}
	strs := map[string]string{
		"SLM_CPU_0_TFTP_FILE_FIRMWARE": FileFirmware,
		"SLM_CPU_0_TFTP_FILE_INFO":     FileInfo,
		"SLM_CPU_0_TFTP_OPT_CRC32":     OptCRC32,
	}
	for name, want := range strs {
		if got := strings.Trim(defs[name], `"`); got != want {
			t.Errorf("%s = %q in boot_shared.h, layout.go has %q", name, got, want)
		}
	}
}

func image(sp, reset uint32, n int) []byte {
	b := make([]byte, n)
	binary.LittleEndian.PutUint32(b, sp)
	binary.LittleEndian.PutUint32(b[4:], reset)
	return b
}

func TestCheckImage(t *testing.T) {
	cases := []struct {
		name string
		img  []byte
		ok   bool
	}{
		{"valid", image(0x20020000, AppBase+0x189, 4096), true},
		{"ccm stack", image(0x10010000, AppBase+0x189, 4096), true},
		{"too small", []byte{1, 2, 3}, false},
		{"too large", image(0x20020000, AppBase+0x189, AppMaxSize+1), false},
		{"stack not in RAM", image(0x08010000, AppBase+0x189, 4096), false},
		{"unaligned stack", image(0x2001FFFE, AppBase+0x189, 4096), false},
		{"arm reset handler", image(0x20020000, AppBase+0x188, 4096), false},
		{"built for address 0", image(0x20020000, 0x08000189, 4096), false},
		{"reset beyond image", image(0x20020000, AppBase+4096+1, 4096), false},
	}
	for _, c := range cases {
		if err := CheckImage(c.img); (err == nil) != c.ok {
			t.Errorf("%s: CheckImage = %v", c.name, err)
		}
	}
}
