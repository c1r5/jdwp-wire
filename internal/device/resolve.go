package device

import (
	"fmt"
	"strings"
)

func resolve(devs []Device, serial string) (Device, error) {
	serial = strings.TrimSpace(serial)

	var match *Device
	for i := range devs {
		if devs[i].Serial == serial {
			match = &devs[i]
			break
		}
	}

	if serial != "" {
		if match == nil {
			if usableCount(devs) == 0 {
				return Device{}, noDeviceError(devs)
			}
			return Device{}, fmt.Errorf("%w: %s", ErrDeviceNotFound, serial)
		}
		if !match.Usable() {
			return Device{}, fmt.Errorf("%w: %s (%s)", ErrDeviceUnusable, match.Serial, match.State)
		}
		return *match, nil
	}

	var usable []Device
	for _, d := range devs {
		if d.Usable() {
			usable = append(usable, d)
		}
	}
	switch len(usable) {
	case 0:
		return Device{}, noDeviceError(devs)
	case 1:
		return usable[0], nil
	default:
		serials := make([]string, len(usable))
		for i, d := range usable {
			serials[i] = d.Serial
		}
		return Device{}, fmt.Errorf("%w: %s", ErrAmbiguousDevice, strings.Join(serials, ", "))
	}
}

func usableCount(devs []Device) int {
	n := 0
	for _, d := range devs {
		if d.Usable() {
			n++
		}
	}
	return n
}

func noDeviceError(devs []Device) error {
	if len(devs) == 0 {
		return ErrNoDevice
	}
	parts := make([]string, 0, len(devs))
	for _, d := range devs {
		parts = append(parts, fmt.Sprintf("%s:%s", d.Serial, d.State))
	}
	return fmt.Errorf("%w: %s", ErrNoDevice, strings.Join(parts, ", "))
}
