package evdev

import (
	"fmt"
	"os"
)

// InputPath contains information about an InputDevice Name & Path
type InputPath struct {
	Name string
	Path string
}

// ListDevicePaths lists all available input devices, returning their
// filename path, and the name as reported by the kernel.
func ListDevicePaths() ([]InputPath, error) {
	var list []InputPath

	basePath := "/dev/input"

	files, err := os.ReadDir(basePath)
	if err != nil {
		return list, err
	}

	for _, fileName := range files {
		if fileName.IsDir() {
			continue
		}

		full := fmt.Sprintf("%s/%s", basePath, fileName.Name())
		if d, err := OpenWithFlags(full, os.O_RDONLY); err == nil {
			name, _ := d.Name()
			list = append(list, InputPath{Name: name, Path: d.Path()})
			d.Close()
		}
	}
	return list, nil
}
