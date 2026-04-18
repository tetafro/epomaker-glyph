// Send the current date and time to an Epomaker Glyph keyboard.
package main

import (
	"context"
	"encoding/hex"
	"errors"
	"flag"
	"fmt"
	"math"
	"os"
	"os/signal"
	"syscall"
	"time"

	hid "github.com/sstallion/go-hid"
)

type Flags struct {
	List bool
	Run  bool
	VID  uint16
	PID  uint16
	Time time.Time
}

func main() {
	ctx, cancel := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer cancel()

	flags, err := parseFlags()
	if err != nil {
		failf("Failed to parse flags: %v", err)
	}

	switch {
	case flags.List:
		if err := listDevices(); err != nil {
			failf("Failed to list devices: %v", err)
		}
	case flags.Run:
		if err := runLoop(ctx, flags.VID, flags.PID); err != nil {
			failf("Failed to run: %v", err)
		}
	default:
		fmt.Printf("Setting time to %s...\n", flags.Time.Format("2006-01-02T15:04:05"))
		if err := setTime(flags.VID, flags.PID, flags.Time); err != nil {
			failf("Failed to set time: %v", err)
		}
		fmt.Println("Done")
	}
}

func parseFlags() (Flags, error) {
	var flags Flags

	flag.BoolVar(&flags.List, "list", false, "List all HID devices and exit")
	flag.BoolVar(&flags.Run, "run", false, "Run in a loop, set time every hour")
	vid := flag.Uint("vid", 0x3151, "Vendor ID in hex, e.g. 0x3151") // Default is ROYUAN (Epomaker's USB vendor)
	pid := flag.Uint("pid", 0, "Product ID in hex, e.g. 0x5002")
	timeStr := flag.String("time", "", `Time to send, e.g. "2006-01-02T15:04:05" (default: now)`)
	flag.Parse()

	if flags.List {
		return flags, nil
	}

	if *pid == 0 {
		return Flags{}, fmt.Errorf("empty -pid")
	}
	if *pid > math.MaxUint16 {
		return Flags{}, fmt.Errorf("product ID overflows uint16, max: 0x%04x", math.MaxUint16)
	}
	flags.PID = uint16(*pid)

	if *vid > math.MaxUint16 {
		return Flags{}, fmt.Errorf("vendor ID overflows uint16, max: 0x%04x", math.MaxUint16)
	}
	flags.VID = uint16(*vid)

	flags.Time = time.Now()
	if *timeStr != "" {
		var err error
		flags.Time, err = time.Parse("2006-01-02T15:04:05", *timeStr)
		if err != nil {
			return Flags{}, fmt.Errorf("invalid -time value: '%s'", *timeStr)
		}
	}

	return flags, nil
}

func runLoop(ctx context.Context, vid, pid uint16) error {
	// Don't wait for the ticker for the first run
	t := time.Now()
	fmt.Printf("Setting time to %s...\n", t.Format("2006-01-02T15:04:05"))
	if err := setTime(vid, pid, t); err != nil {
		return fmt.Errorf("set time: %w", err)
	}
	fmt.Println("Done")

	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			t := time.Now()
			fmt.Printf("Setting time to %s...\n", t.Format("2006-01-02T15:04:05"))
			if err := setTime(vid, pid, t); err != nil {
				return fmt.Errorf("set time: %w", err)
			}
			fmt.Println("Done")
		case <-ctx.Done():
			return nil
		}
	}
}

func listDevices() error {
	// Collect devices and group by vid/pid
	devices := map[string]hid.DeviceInfo{}
	err := hid.Enumerate(hid.VendorIDAny, hid.ProductIDAny, func(dev *hid.DeviceInfo) error {
		devices[fmt.Sprintf("0x%04x:0x%04x", dev.VendorID, dev.ProductID)] = *dev
		return nil
	})
	if err != nil {
		return fmt.Errorf("enumerate devices: %w", err)
	}

	// Print found devices
	for _, dev := range devices {
		name := dev.ProductStr
		if dev.MfrStr != "" {
			name = dev.MfrStr + "/" + name
		}
		fmt.Printf("vid=0x%04x pid=0x%04x %s\n",
			dev.VendorID, dev.ProductID, name)
	}
	return nil
}

func setTime(vid, pid uint16, t time.Time) error {
	packet := buildTimePacket(t)
	stopEnum := fmt.Errorf("stop")

	var errs []error
	err := hid.Enumerate(vid, pid, func(info *hid.DeviceInfo) error {
		dev, err := hid.OpenPath(info.Path)
		if err != nil {
			errs = append(errs, fmt.Errorf("open path %s: %w", info.Path, err))
			return nil
		}
		_, err = dev.SendFeatureReport(packet)
		dev.Close() //nolint:errcheck,gosec
		if err != nil {
			errs = append(errs, fmt.Errorf("send packet %s: %w", info.Path, err))
			return nil
		}
		return stopEnum // stop enumeration
	})
	if !errors.Is(err, stopEnum) {
		if len(errs) == 0 {
			return fmt.Errorf("device not found: vid=0x%04x pid=0x%04x", vid, pid)
		}
		return errors.Join(errs...)
	}
	return nil
}

// buildTimePacket constructs the 64-byte feature report for setting the time.
// The format is copied from https://github.com/strodgers/epomaker-controller.
func buildTimePacket(t time.Time) []byte {
	packet := make([]byte, 64)
	header, _ := hex.DecodeString("28000000000000d7")
	copy(packet, header)
	y := t.Year()
	packet[8] = byte(y >> 8)
	packet[9] = byte(y)
	packet[10] = byte(t.Month())
	packet[11] = byte(t.Day())
	packet[12] = byte(t.Hour())
	packet[13] = byte(t.Minute())
	packet[14] = byte(t.Second())
	return packet
}

func failf(format string, args ...any) {
	fmt.Printf(format+"\n", args...)
	os.Exit(1)
}
