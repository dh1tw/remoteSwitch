package remotebox

import sw "github.com/dh1tw/remoteSwitch/switch"

// Name is a functional option to set the name of this device.
func Name(name string) func(*Remotebox) {
	return func(r *Remotebox) {
		r.name = name
	}
}

// Index is a functional option to set the order in which it will
// be displayed on the graphical interface.
func Index(i int) func(*Remotebox) {
	return func(r *Remotebox) {
		r.index = i
	}
}

// Portname is a functional option to set the portname of the serial port.
// On Windows this will be "COMx", on Linux & MacOS "/dev/tty/xxx"
func Portname(pn string) func(*Remotebox) {
	return func(r *Remotebox) {
		r.spPortname = pn
	}
}


// IpAddress
func Ipaddress(pn string) func(*Remotebox) {
	return func(r *Remotebox) {
		r.Ipaddress = pn
	}
}

// ipport
func Ipport(i int) func(*Remotebox) {
	return func(r *Remotebox) {
		r.Ipport = i
	}
}

// typeConnection
func Connection(i int) func(*Remotebox) {
	return func(r *Remotebox) {
		r.Connection = i
	}
}

// EventHandler sets a callback function through which the bandswitch
// will report Events
func EventHandler(h func(sw.Switcher, sw.Device)) func(*Remotebox) {
	return func(r *Remotebox) {
		r.eventHandler = h
	}
}

// ErrorCh is a functional option allows you to pass a channel to the remotebox.
// The channel will be closed when an internal error occures.
func ErrorCh(ch chan struct{}) func(*Remotebox) {
	return func(r *Remotebox) {
		r.errorCh = ch
	}
}
