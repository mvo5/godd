package udev

import (
	"bufio"
	"bytes"
	"fmt"
	"io/ioutil"
	"os/exec"
	"path/filepath"
	"strings"
)

// This code used to use a tiny C wrapper around libudev.
// However - this makes the packaging more complicated (now we need
// to ship libudev in the snap) and is also not really needed because
// we do not use any of the dynamic niceness of libudev. We just use
// it to detect removable devices which we can equally well do with
// the output of udevadm.

type Device struct {
	properties map[string]string
}

func (e *Device) GetSysfsAttr(attr string) string {
	p := filepath.Join("/sys", e.properties["DEVPATH"], attr)
	content, err := ioutil.ReadFile(p)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(content))
}

func (e *Device) GetProperty(name string) string {
	return e.properties[name]
}

func (e *Device) GetDeviceFile() string {
	return e.properties["DEVNAME"]
}

func parseDevice(block string) (*Device, error) {
	props := make(map[string]string)
	for i, line := range strings.Split(block, "\n") {
		if i == 0 && !strings.HasPrefix(line, "P: ") {
			return nil, fmt.Errorf("no device block marker found before %q", line)
		}
		if strings.HasPrefix(line, "E: ") {
			if kv := strings.SplitN(line[3:], "=", 2); len(kv) == 2 {
				props[kv[0]] = kv[1]
			} else {
				return nil, fmt.Errorf("failed to parse udevadm output %q", line)
			}
		}
	}
	return &Device{properties: props}, nil
}

func QueryBySubsystem(sub string) ([]*Device, error) {
	cmd := exec.Command("udevadm", "info", "-e")
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}

	scanner := bufio.NewScanner(stdout)
	scanner.Split(scanDoubleNewline)
	if err := cmd.Start(); err != nil {
		return nil, err
	}

	var res []*Device
	for scanner.Scan() {
		block := scanner.Text()
		env, err := parseDevice(block)
		if err != nil {
			return nil, err
		}
		if sub != "" && env.GetProperty("SUBSYSTEM") != sub {
			continue
		}
		res = append(res, env)
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("cannot read udevadm output: %s", err)
	}
	if err := cmd.Wait(); err != nil {
		return nil, fmt.Errorf("cannot run udevadm command: %s", err)
	}

	return res, nil
}

// helpers

// udevadm output scanner (all devices are separated via \n\n)
func scanDoubleNewline(data []byte, atEOF bool) (advance int, token []byte, err error) {
	if atEOF && len(data) == 0 {
		return 0, nil, nil
	}

	if i := bytes.Index(data, []byte("\n\n")); i >= 0 {
		// we found data
		return i + 2, data[0:i], nil
	}

	// If we're at EOF, return what is left.
	if atEOF {
		return len(data), data, nil
	}
	// Request more data.
	return 0, nil, nil
}
