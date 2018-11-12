package DummySwitch

import sw "github.com/dh1tw/remoteSwitch/switch"

// Switch is a functional option to set the switch's configuration.
func Switch(sc SwitchConfig) func(*DummySwitch) {
	return func(g *DummySwitch) {
		g.switchConfig = sc
	}
}

// SwitchConfig describes a switch which is a collection of ports.
type SwitchConfig struct {
	Name      string
	ID        int
	Exclusive bool
	Ports     []PortConfig
}

// PortConfig describes a port which is a collection of gpio pins. This struct
// is injected through the functional option "Port" during construction of BandswitchGPIO.
type PortConfig struct {
	Name      string
	ID        int
	Exclusive bool
	Terminals []PinConfig
}

// PinConfig describes a gpio pin.
type PinConfig struct {
	Name string
	ID   int
}

// EventHandler sets a callback function through which the bandswitch
// will report Events
func EventHandler(h func(sw.Switcher, sw.Device)) func(*DummySwitch) {
	return func(g *DummySwitch) {
		g.eventHandler = h
	}
}
