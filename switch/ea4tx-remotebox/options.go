package remotebox

import sw "github.com/dh1tw/remoteSwitch/switch"

// Baudrate is a functional option to set the baurate of the serial port.
func Baudrate(baudrate int) func(*Remotebox) {
	return func(r *Remotebox) {
		r.spBaudrate = baudrate
	}
}

// Portname is a functional option to set the portname of the serial port.
// On Windows this will be "COMx", on Linux & MacOS "/dev/tty/xxx"
func Portname(pn string) func(*Remotebox) {
	return func(r *Remotebox) {
		r.spPortname = pn
	}
}

// EventHandler sets a callback function through which the bandswitch
// will report Events
func EventHandler(h func(sw.Switcher, sw.Device)) func(*Remotebox) {
	return func(r *Remotebox) {
		r.eventHandler = h
	}
}
