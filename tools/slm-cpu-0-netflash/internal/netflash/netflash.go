// Package netflash is the slm-cpu-0-netflash upload: hand the board over to its
// bootloader, write the image over TFTP and check that the new program runs.
package netflash

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"strconv"
	"time"

	"github.com/Autonomy-Logic/slm-cpu-0-arduino-core/tools/slm-cpu-0-netflash/internal/boot"
	"github.com/Autonomy-Logic/slm-cpu-0-arduino-core/tools/slm-cpu-0-netflash/internal/handover"
	"github.com/Autonomy-Logic/slm-cpu-0-arduino-core/tools/slm-cpu-0-netflash/internal/layout"
	"github.com/Autonomy-Logic/slm-cpu-0-arduino-core/tools/slm-cpu-0-netflash/internal/modbus"
	"github.com/Autonomy-Logic/slm-cpu-0-arduino-core/tools/slm-cpu-0-netflash/internal/tftp"
)

// Options for one upload.
type Options struct {
	IP         net.IP
	TFTPPort   int
	ModbusPort int
	Image      []byte
	// Verify waits for the new program's runtime to answer.
	Verify bool
	// ExpectRuntime is the runtime version the program should report; a
	// different one gives a warning. Empty skips the comparison.
	ExpectRuntime string
	// BootWait bounds the wait for the bootloader after the hand-over.
	BootWait time.Duration
	// VerifyWait bounds the wait for the new program.
	VerifyWait time.Duration
	Out        io.Writer
	Verbose    bool
}

type run struct {
	Options
	target boot.Target
	modbus string
}

func (r *run) say(format string, args ...any) { fmt.Fprintf(r.Out, format+"\n", args...) }

func (r *run) debug(format string, args ...any) {
	if r.Verbose {
		fmt.Fprintf(r.Out, "  "+format+"\n", args...)
	}
}

// Run uploads o.Image to the board at o.IP.
func Run(ctx context.Context, o Options) error {
	if err := layout.CheckImage(o.Image); err != nil {
		return err
	}
	r := &run{Options: o, modbus: net.JoinHostPort(o.IP.String(), strconv.Itoa(o.ModbusPort))}
	r.target = boot.Target{Addr: &net.UDPAddr{IP: o.IP, Port: o.TFTPPort}, Logf: r.debug}
	r.say("Image: %d bytes (%d%% of the application area), CRC-32 %08x",
		len(o.Image), len(o.Image)*100/layout.AppMaxSize, layout.CRC32(o.Image))

	info, err := handover.Enter(ctx, handover.Options{
		Target:     r.target,
		IP:         o.IP,
		ModbusPort: o.ModbusPort,
		BootWait:   o.BootWait,
		Say:        r.say,
		Debug:      r.debug,
	})
	if err != nil {
		return err
	}
	r.say("Bootloader %s ready (MAC %s, entered: %s).", info["bootloader"], info["mac"], info["entry"])

	r.say("Writing %d bytes...", len(o.Image))
	start := time.Now()
	next := 20
	err = r.target.WriteFirmware(ctx, o.Image, func(done, total int) {
		for pct := done * 100 / total; next <= 100 && pct >= next; next += 20 {
			r.say("  %d%%", next)
		}
	})
	unconfirmed := errors.Is(err, tftp.ErrNoFinalAck)
	switch {
	case unconfirmed:
		r.say("The bootloader did not confirm the last block; checking whether the new program started.")
	case err != nil:
		return writeError(err)
	default:
		r.say("Written and checked by the bootloader in %.1f s.", time.Since(start).Seconds())
	}
	if !o.Verify {
		if unconfirmed {
			return errors.New("the bootloader did not confirm the upload")
		}
		return nil
	}
	return r.verify(ctx, unconfirmed)
}

// verify waits for the runtime of the new program to answer.
func (r *run) verify(ctx context.Context, unconfirmed bool) error {
	r.say("Waiting for the new program to start...")
	// The bootloader waits a second for a repeated last block, then resets.
	sleep(ctx, 1500*time.Millisecond)
	deadline := time.Now().Add(r.VerifyWait)
	var probed time.Time
	for time.Now().Before(deadline) && ctx.Err() == nil {
		if c, err := modbus.Dial(ctx, r.modbus, time.Second); err == nil {
			version, verr := c.RuntimeVersion()
			_, derr := c.DeviceID()
			c.Close()
			if verr == nil && derr == nil {
				if r.ExpectRuntime != "" && version != r.ExpectRuntime {
					r.say("Warning: the program reports OpenPLC runtime %s; this board package expects %s.",
						version, r.ExpectRuntime)
				}
				r.say("OpenPLC runtime %s is running at %s.", version, r.IP)
				return nil
			}
			r.debug("runtime not ready: %v %v", verr, derr)
		}
		if time.Since(probed) >= 3*time.Second {
			probed = time.Now()
			if info, err := r.target.ReadInfo(ctx, 700*time.Millisecond); err == nil {
				return fmt.Errorf("the new program did not start: the board is still in its bootloader "+
					"(entered: %s, app: %s)", info["entry"], info["app"])
			}
		}
		sleep(ctx, 500*time.Millisecond)
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if unconfirmed {
		return fmt.Errorf("could not confirm the upload: %s answered neither as the bootloader nor as the new program", r.IP)
	}
	r.say("Warning: the image was written, but no OpenPLC runtime answered at %s within %s. "+
		"If the new program uses a different IP address, it is running there.", r.IP, r.VerifyWait)
	return nil
}

func writeError(err error) error {
	var se *tftp.ServerError
	switch {
	case errors.As(err, &se):
		return fmt.Errorf("the bootloader refused the image: %s", se.Msg)
	case errors.Is(err, tftp.ErrTimeout):
		return errors.New("the bootloader stopped answering during the upload; the board keeps waiting " +
			"in its bootloader, so run the upload again")
	default:
		return fmt.Errorf("upload failed: %w", err)
	}
}

func sleep(ctx context.Context, d time.Duration) { handover.Sleep(ctx, d) }
