package sbSwitchProxy

import (
	sw "github.com/dh1tw/remoteSwitch/switch"
	"github.com/micro/go-micro/client"
)

func Client(cli client.Client) func(*SbSwitchProxy) {
	return func(s *SbSwitchProxy) {
		s.cli = cli
	}
}

// DoneCh is a functional option allows you to pass a channel to the proxy object.
// This channel will be closed by this object. It serves as a notification that
// the object can be deleted.
func DoneCh(ch chan struct{}) func(*SbSwitchProxy) {
	return func(s *SbSwitchProxy) {
		s.doneCh = ch
	}
}

func Name(name string) func(*SbSwitchProxy) {
	return func(s *SbSwitchProxy) {
		s.device.Name = name
	}
}

func ServiceName(name string) func(*SbSwitchProxy) {
	return func(s *SbSwitchProxy) {
		s.serviceName = name
	}
}

// EventHandler sets a callback function through which the proxy rotator
// will report Events
func EventHandler(h func(sw.Switcher, sw.Device)) func(*SbSwitchProxy) {
	return func(s *SbSwitchProxy) {
		s.eventHandler = h
	}
}
