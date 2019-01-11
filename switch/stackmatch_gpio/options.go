package StackmatchGPIO

import sw "github.com/dh1tw/remoteSwitch/switch"

// Switch is a functional option to set the switch's configuration.
func Config(sc SmConfig) func(*SmGPIO) {
	return func(g *SmGPIO) {
		g.config = sc
	}
}

type SmConfig struct {
	Name         string
	ID           int
	Combinations []CombinationConfig
}

type CombinationConfig struct {
	Name      string
	Terminals []TerminalConfig
	Pins      []PinConfig
}

type TerminalConfig struct {
	Name string
	ID   int
}

// PinConfig describes a gpio pin.
type PinConfig struct {
	Name     string
	Pin      string
	Inverted bool
}

// EventHandler sets a callback function through which the bandswitch
// will report Events
func EventHandler(h func(sw.Switcher, sw.Device)) func(*SmGPIO) {
	return func(s *SmGPIO) {
		s.eventHandler = h
	}
}
