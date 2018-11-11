package cmd

import (
	"fmt"

	mpGPIO "github.com/dh1tw/remoteSwitch/switch/multi-purpose-switch-gpio"
	"github.com/spf13/viper"
)

func getMPGPIOSwitches() ([]mpGPIO.SwitchConfig, error) {

	scs := []mpGPIO.SwitchConfig{}

	// let's check first if all necessary keys exist in the config file

	if !viper.IsSet("switches.multi_purpose_gpio") {
		return scs, nil
	}

	switches := viper.GetStringSlice(("switches.multi_purpose_gpio"))

	for _, switchName := range switches {
		s, err := getMPGPIOSwitch(switchName)
		if err != nil {
			return nil, err
		}
		scs = append(scs, s)
	}

	return scs, nil
}

func getMPGPIOSwitch(switchName string) (mpGPIO.SwitchConfig, error) {

	sc := mpGPIO.SwitchConfig{}

	// let's check first if all necessary keys exist in the config file
	if !viper.IsSet(fmt.Sprintf("%s.name", switchName)) {
		return sc, fmt.Errorf("missing name parameter for switch %s", switchName)
	}

	if !viper.IsSet(fmt.Sprintf("%s.id", switchName)) {
		return sc, fmt.Errorf("missing id parameter for switch %s", switchName)
	}

	if !viper.IsSet(fmt.Sprintf("%s.exclusive", switchName)) {
		return sc, fmt.Errorf("missing exclusive parameter for switch %s", switchName)
	}

	if !viper.IsSet(fmt.Sprintf("%s.ports", switchName)) {
		return sc, fmt.Errorf("missing ports parameter for switch %s", switchName)
	}

	// get the values
	name := viper.GetString(fmt.Sprintf("%s.name", switchName))
	if len(name) == 0 {
		return sc, fmt.Errorf("name parameter of switch %s must not be empty", switchName)
	}

	id := viper.GetInt(fmt.Sprintf("%s.id", switchName))
	exclusive := viper.GetBool(fmt.Sprintf("%s.exclusive", switchName))
	ports := viper.GetStringSlice(fmt.Sprintf("%s.ports", switchName))
	if len(ports) == 0 {
		return sc, fmt.Errorf("no ports found for switch %s", switchName)
	}

	sc.Name = name
	sc.ID = id
	sc.Exclusive = exclusive

	for _, port := range ports {
		p, err := getMPGPIOPort(port)
		if err != nil {
			return sc, err
		}
		sc.Ports = append(sc.Ports, p)
	}

	return sc, nil
}

func getMPGPIOPort(portName string) (mpGPIO.PortConfig, error) {

	pc := mpGPIO.PortConfig{}

	// let's check first if all necessary keys exist in the config file
	if !viper.IsSet(portName) {
		return pc, fmt.Errorf("no configuration found for port %s", portName)
	}

	if !viper.IsSet(fmt.Sprintf("%s.name", portName)) {
		return pc, fmt.Errorf("missing name parameter for port %s", portName)
	}

	if !viper.IsSet(fmt.Sprintf("%s.id", portName)) {
		return pc, fmt.Errorf("missing id parameter for port %s", portName)
	}

	if !viper.IsSet(fmt.Sprintf("%s.terminals", portName)) {
		return pc, fmt.Errorf("missing terminals parameter for port %s", portName)
	}

	// get the values
	name := viper.GetString(fmt.Sprintf("%s.name", portName))
	if len(name) == 0 {
		return pc, fmt.Errorf("name parameter of port %s must not be empty", portName)
	}

	id := viper.GetInt(fmt.Sprintf("%s.id", portName))
	exclusive := viper.GetBool(fmt.Sprintf("%s.exclusive", portName))
	terminals := viper.GetStringSlice(fmt.Sprintf("%s.terminals", portName))
	if len(terminals) == 0 {
		return pc, fmt.Errorf("no terminals found for port %s", portName)
	}

	pc.Name = name
	pc.ID = id
	pc.Exclusive = exclusive
	pc.Terminals = make([]mpGPIO.PinConfig, 0, len(terminals))

	for _, terminal := range terminals {
		t, err := getMPGPIOTerminal(terminal)
		if err != nil {
			return pc, err
		}
		pc.Terminals = append(pc.Terminals, t)
	}

	return pc, nil
}

func getMPGPIOTerminal(terminalName string) (mpGPIO.PinConfig, error) {

	pc := mpGPIO.PinConfig{}

	// let's check first if all necessary keys exist in the config file
	if !viper.IsSet(fmt.Sprintf("%s.name", terminalName)) {
		return pc, fmt.Errorf("missing name parameter for terminal %s", terminalName)
	}

	if !viper.IsSet(fmt.Sprintf("%s.id", terminalName)) {
		return pc, fmt.Errorf("missing id parameter for terminal %s", terminalName)
	}

	if !viper.IsSet(fmt.Sprintf("%s.pin", terminalName)) {
		return pc, fmt.Errorf("missing pin parameter for port %s", terminalName)
	}

	// get the values
	name := viper.GetString(fmt.Sprintf("%s.name", terminalName))
	if len(name) == 0 {
		return pc, fmt.Errorf("name parameter of terminal %s must not be empty", terminalName)
	}

	pin := viper.GetString(fmt.Sprintf("%s.pin", terminalName))
	if len(pin) == 0 {
		return pc, fmt.Errorf("pin parameter of terminal %s must not be empty", terminalName)
	}

	id := viper.GetInt(fmt.Sprintf("%s.id", terminalName))
	inverted := viper.GetBool(fmt.Sprintf("%s.inverted", terminalName))

	pc.Name = name
	pc.Pin = pin
	pc.ID = id
	pc.Inverted = inverted

	return pc, nil
}
