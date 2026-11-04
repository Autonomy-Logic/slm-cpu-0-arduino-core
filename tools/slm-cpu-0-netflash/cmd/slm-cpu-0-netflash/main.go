// Command slm-cpu-0-netflash uploads an OpenPLC program to an SLM-CPU-0 over
// Ethernet. arduino-cli runs it as the default upload tool of FACTS:stm32:
//
//	slm-cpu-0-netflash upload [-v] --ip <address> --bin <image.bin>
//	slm-cpu-0-netflash info --ip <address>
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"net"
	"os"
	"os/signal"
	"sort"
	"time"

	"github.com/Autonomy-Logic/slm-cpu-0-arduino-core/tools/slm-cpu-0-netflash/internal/boot"
	"github.com/Autonomy-Logic/slm-cpu-0-arduino-core/tools/slm-cpu-0-netflash/internal/layout"
	"github.com/Autonomy-Logic/slm-cpu-0-arduino-core/tools/slm-cpu-0-netflash/internal/modbus"
	"github.com/Autonomy-Logic/slm-cpu-0-arduino-core/tools/slm-cpu-0-netflash/internal/netflash"
)

const name = "slm-cpu-0-netflash"

// version is set at build time with -ldflags "-X main.version=...".
var version = "dev"

const usage = `Usage:
  slm-cpu-0-netflash upload [-v] --ip <address> --bin <image.bin>
      Start the board's bootloader, write the image and check that it runs.
  slm-cpu-0-netflash info --ip <address>
      Show what the board at <address> is running.
  slm-cpu-0-netflash version
`

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	os.Exit(run(ctx, os.Args[1:], os.Stdout, os.Stderr))
}

func run(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprint(stderr, usage)
		return 2
	}
	var err error
	switch args[0] {
	case "upload":
		err = upload(ctx, args[1:], stdout, stderr)
	case "info":
		err = info(ctx, args[1:], stdout, stderr)
	case "version", "--version":
		fmt.Fprintln(stdout, name, version)
	case "help", "-h", "--help":
		fmt.Fprint(stdout, usage)
	default:
		fmt.Fprintf(stderr, "%s: unknown command %q\n%s", name, args[0], usage)
		return 2
	}
	var usageErr *usageError
	switch {
	case errors.As(err, &usageErr):
		fmt.Fprintf(stderr, "%s: %v\n%s", name, err, usage)
		return 2
	case err != nil:
		fmt.Fprintf(stderr, "%s: error: %v\n", name, err)
		return 1
	}
	return 0
}

type usageError struct{ msg string }

func (e *usageError) Error() string { return e.msg }

type common struct {
	ip         *string
	tftpPort   *int
	modbusPort *int
}

func newFlags(cmd string, stderr io.Writer) (*flag.FlagSet, common) {
	fs := flag.NewFlagSet(cmd, flag.ContinueOnError)
	fs.SetOutput(stderr)
	return fs, common{
		ip:         fs.String("ip", "", "board IP address"),
		tftpPort:   fs.Int("tftp-port", layout.TFTPPort, "bootloader TFTP port"),
		modbusPort: fs.Int("modbus-port", 502, "OpenPLC runtime Modbus TCP port"),
	}
}

func (c common) resolve() (net.IP, error) {
	if *c.ip == "" {
		return nil, &usageError{"--ip is required"}
	}
	a, err := net.ResolveIPAddr("ip4", *c.ip)
	if err != nil {
		return nil, fmt.Errorf("%s is not an IPv4 address or known host name", *c.ip)
	}
	return a.IP, nil
}

func upload(ctx context.Context, args []string, stdout, stderr io.Writer) error {
	fs, c := newFlags("upload", stderr)
	bin := fs.String("bin", "", "application image (.bin)")
	verbose := fs.Bool("v", false, "show protocol details")
	noVerify := fs.Bool("no-verify", false, "don't wait for the new program to answer")
	expect := fs.String("expect-runtime", "", "OpenPLC runtime version the program should report; empty skips the check")
	bootWait := fs.Duration("boot-wait", 15*time.Second, "how long to wait for the bootloader")
	verifyWait := fs.Duration("verify-wait", 30*time.Second, "how long to wait for the new program")
	if err := fs.Parse(args); err != nil {
		return &usageError{err.Error()}
	}
	ip, err := c.resolve()
	if err != nil {
		return err
	}
	if *bin == "" {
		return &usageError{"--bin is required"}
	}
	image, err := os.ReadFile(*bin)
	if err != nil {
		return err
	}
	fmt.Fprintf(stdout, "%s %s: uploading to %s\n", name, version, ip)
	return netflash.Run(ctx, netflash.Options{
		IP:            ip,
		TFTPPort:      *c.tftpPort,
		ModbusPort:    *c.modbusPort,
		Image:         image,
		Verify:        !*noVerify,
		ExpectRuntime: *expect,
		BootWait:      *bootWait,
		VerifyWait:    *verifyWait,
		Out:           stdout,
		Verbose:       *verbose,
	})
}

// info only reads: a TFTP read of the bootloader's info file, else a Modbus
// read of the runtime version.
func info(ctx context.Context, args []string, stdout, stderr io.Writer) error {
	fs, c := newFlags("info", stderr)
	if err := fs.Parse(args); err != nil {
		return &usageError{err.Error()}
	}
	ip, err := c.resolve()
	if err != nil {
		return err
	}
	target := boot.Target{Addr: &net.UDPAddr{IP: ip, Port: *c.tftpPort}}
	if bi, err := target.ReadInfo(ctx, 1500*time.Millisecond); err == nil {
		keys := make([]string, 0, len(bi))
		for k := range bi {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			fmt.Fprintf(stdout, "%s=%s\n", k, bi[k])
		}
		return nil
	}
	mb, err := modbus.Dial(ctx, net.JoinHostPort(ip.String(), fmt.Sprint(*c.modbusPort)), 3*time.Second)
	if err != nil {
		return fmt.Errorf("nothing answered at %s: no bootloader (TFTP) and no OpenPLC runtime (Modbus TCP)", ip)
	}
	defer mb.Close()
	v, err := mb.RuntimeVersion()
	if err != nil {
		return fmt.Errorf("the device at %s did not report an OpenPLC runtime version: %w", ip, err)
	}
	fmt.Fprintf(stdout, "mode=runtime\nruntime=%s\n", v)
	return nil
}
