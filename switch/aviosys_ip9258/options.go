package ip9258

import sw "github.com/dh1tw/remoteSwitch/switch"

// Name is a functional option to set the name of this device.
func Name(name string) func(*IP9258) {
	return func(r *IP9258) {
		r.name = name
	}
}

// Index is a functional option to set the order in which it will
// be displayed on the graphical interface.
func Index(i int) func(*IP9258) {
	return func(r *IP9258) {
		r.index = i
	}
}

// Username is a functional option to set the username required to access the IP9258.
func Username(username string) func(*IP9258) {
	return func(r *IP9258) {
		r.username = username
	}
}

// Password is a functional option to set the password required to access the IP9258.
func Password(password string) func(*IP9258) {
	return func(r *IP9258) {
		r.password = password
	}
}

// URL is a functional option to set the IP address or url under which the
// IP9258 web switch can be found.
func URL(url string) func(*IP9258) {
	return func(r *IP9258) {
		r.rawurl = url
	}
}

func Terminals(ts []Terminal) func(*IP9258) {
	return func(r *IP9258) {
		// better make a copy
		for _, t := range ts {
			r.terminals[t.Outlet] = &Terminal{
				Name:   t.Name,
				Outlet: t.Outlet,
				Index:  t.Index,
				state:  false,
			}
		}
	}
}

// EventHandler sets a callback function through which the bandswitch
// will report Events
func EventHandler(h func(sw.Switcher, sw.Device)) func(*IP9258) {
	return func(r *IP9258) {
		r.eventHandler = h
	}
}

// ErrorCh is a functional option allows you to pass a channel to the IP9258.
// The channel will be closed when an internal error occures.
func ErrorCh(ch chan struct{}) func(*IP9258) {
	return func(r *IP9258) {
		r.errorCh = ch
	}
}
