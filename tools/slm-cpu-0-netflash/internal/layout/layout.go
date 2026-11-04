// Package layout mirrors the parts of the core's boot_shared.h
// (overlay/variants/SLM_CPU_0) that the uploader uses: the flash map and the
// bootloader's TFTP service. layout_test.go checks the constants against it.
package layout

import (
	"encoding/binary"
	"errors"
	"fmt"
	"hash/crc32"
)

const (
	FlashBase  = 0x08000000
	FlashEnd   = 0x08080000
	AppBase    = 0x08010000
	AppMaxSize = FlashEnd - AppBase
	RAMBase    = 0x20000000
	RAMEnd     = 0x20020000

	// The bootloader also accepts an initial stack pointer in CCM RAM.
	CCMRAMBase = 0x10000000
	CCMRAMEnd  = 0x10010000

	TFTPPort     = 69
	FileFirmware = "firmware.bin"
	FileInfo     = "info"
	OptCRC32     = "crc32"
	MaxBlksize   = 1468
)

// CRC32 is the bootloader's CRC-32 (zlib, Ethernet).
func CRC32(b []byte) uint32 { return crc32.ChecksumIEEE(b) }

// CheckImage returns why b cannot be an SLM-CPU-0 application image, or nil.
// It makes the vector table check the bootloader makes before it commits an
// upload, so a wrong file is caught before the board leaves its program.
func CheckImage(b []byte) error {
	if len(b) < 8 {
		return errors.New("the image is too small to be an application")
	}
	if len(b) > AppMaxSize {
		return fmt.Errorf("the image is %d bytes, larger than the %d KB application area", len(b), AppMaxSize/1024)
	}
	sp := binary.LittleEndian.Uint32(b)
	reset := binary.LittleEndian.Uint32(b[4:])
	inRAM := (sp >= RAMBase && sp <= RAMEnd) || (sp >= CCMRAMBase && sp <= CCMRAMEnd)
	if !inRAM || sp&3 != 0 {
		return fmt.Errorf("not an SLM-CPU-0 application: its initial stack pointer 0x%08X is not in RAM", sp)
	}
	entry := reset &^ 1
	if reset&1 == 0 || entry < AppBase+8 || entry >= AppBase+uint32(len(b)) {
		return fmt.Errorf("not an SLM-CPU-0 application: its reset handler 0x%08X is outside an image at 0x%08X "+
			"(was it built for another board?)", reset, AppBase)
	}
	return nil
}
