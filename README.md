# Go support for the Linux evdev interface

This is a pure Go package for the Linux evdev interface, without cgo dependencies.

The Linux evdev interface is the userspace interface to interact with input devices such as
keyboard, mice, joysticks, touchscreens, rotary encoders etc.

The implementation in this package has the following features:

* Query device information such as the name, the physical location, the unique ID,
  the vendor/product/bus/version IDs
* Query supported event types, codes and device properties
* Query the current status of bit-field based input types (such as keyboard, switches etc)
  as well as information on absolute types (`ABS_X`, ...) including their min/max values and
  current state
* Grab/Ungrab/Revoke support for exclusive claiming of devices
* Auto-generated `const` definitions and maps for types and codes from the kernel include headers

# Install

```
go get github.com/holoplot/go-evdev
```

And then use it in your source code.

```
import "github.com/holoplot/go-evdev"
```

# Re-generating codes.go

To re-generate `codes.go` from the latest kernel headers, use the following command.

```
go run build/gen-codes/main.go 
gofmt codes.go
```

You can optionally validate generated string mappings with `go run build/gen-codes-validate/main.go` 

# Example

See the code in `cmd/evtest` for an example.

# MIT License

See file `LICENSE` for details.
