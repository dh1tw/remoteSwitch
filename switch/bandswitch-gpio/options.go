package BandswitchGPIO

import sw "github.com/dh1tw/remoteSwitch/switch"

// Name is a functional option to set the name of the rotator
func Name(name string) func(*BandswitchGPIO) {
	return func(g *BandswitchGPIO) {
		g.name = name
	}
}

// Port is a functional option to set a Port configuration.
func Port(pc PortConfig) func(*BandswitchGPIO) {
	return func(g *BandswitchGPIO) {
		g.portConfig[pc.Name] = pc
	}
}

// ID is a functional option to set the display sequence of this element.
// The ID is a reference which is used in GUIs.
func ID(id int) func(*BandswitchGPIO) {
	return func(g *BandswitchGPIO) {
		g.id = id
	}
}

// PortConfig describes a port which is a collection of gpio pins. This struct
// is injected through the functional option "Port" during construction of BandswitchGPIO.
type PortConfig struct {
	Name     string
	ID       int
	OutPorts []PinConfig
}

// PinConfig describes a gpio pin.
type PinConfig struct {
	Name     string
	Pin      string
	Inverted bool
	ID       int
}

// EventHandler sets a callback function through which the bandswitch
// will report Events
func EventHandler(h func(sw.Switcher, sw.Device)) func(*BandswitchGPIO) {
	return func(g *BandswitchGPIO) {
		g.eventHandler = h
	}
}
