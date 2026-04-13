// Send the current date and time to an Epomaker Glyph keyboard.
package main

import (
	"encoding/hex"
	"errors"
	"flag"
	"fmt"
	"math"
	"os"
	"time"

	hid "github.com/sstallion/go-hid"
)

func main() {
	list := flag.Bool("list", false, "List all HID devices and exit")
	vid := flag.Uint("vid", 0x3151, "Vendor ID in hex, e.g. 0x3151") // Default is ROYUAN (Epomaker's USB vendor)
	pid := flag.Uint("pid", 0, "Product ID in hex, e.g. 0x5002")
	timeStr := flag.String("time", "", `Time to send, e.g. "2006-01-02T15:04:05" (default: now)`)
	flag.Parse()

	if *list {
		if err := listDevices(); err != nil {
			failf("Failed to list devices: %v", err)
		}
		return
	}

	if *pid == 0 {
		failf("Empty -pid")
	}
	if *vid > math.MaxUint16 {
		failf("Vendor ID overflows uint16, max: 0x%04x", math.MaxUint16)
	}
	if *pid > math.MaxUint16 {
		failf("Product ID overflows uint16, max: 0x%04x", math.MaxUint16)
	}

	t := time.Now()
	if *timeStr != "" {
		var err error
		t, err = time.Parse("2006-01-02T15:04:05", *timeStr)
		if err != nil {
			failf("Invalid -time value: '%s'", *timeStr)
		}
	}
	fmt.Printf("Setting time to %s...\n", t.Format("2006-01-02T15:04:05"))
	if err := setTime(uint16(*vid), uint16(*pid), t); err != nil {
		failf("Failed to set time: %v", err)
	}
	fmt.Println("Done")
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
