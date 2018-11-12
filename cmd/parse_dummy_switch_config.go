package cmd

import (
	"fmt"

	ds "github.com/dh1tw/remoteSwitch/switch/dummy_switch"
	"github.com/spf13/viper"
)

func getDummySwitchConfig(switchName string) (ds.SwitchConfig, error) {

	sc := ds.SwitchConfig{}

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
		p, err := getDummySwitchPortConfig(port)
		if err != nil {
			return sc, err
		}
		sc.Ports = append(sc.Ports, p)
	}

	return sc, nil
}

func getDummySwitchPortConfig(portName string) (ds.PortConfig, error) {

	pc := ds.PortConfig{}

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
	pc.Terminals = make([]ds.PinConfig, 0, len(terminals))

	for _, terminal := range terminals {
		t, err := getDummySwitchTerminalConfig(terminal)
		if err != nil {
			return pc, err
		}
		pc.Terminals = append(pc.Terminals, t)
	}

	return pc, nil
}

func getDummySwitchTerminalConfig(terminalName string) (ds.PinConfig, error) {

	pc := ds.PinConfig{}

	// let's check first if all necessary keys exist in the config file
	if !viper.IsSet(fmt.Sprintf("%s.name", terminalName)) {
		return pc, fmt.Errorf("missing name parameter for terminal %s", terminalName)
	}

	if !viper.IsSet(fmt.Sprintf("%s.id", terminalName)) {
		return pc, fmt.Errorf("missing id parameter for terminal %s", terminalName)
	}

	// get the values
	name := viper.GetString(fmt.Sprintf("%s.name", terminalName))
	if len(name) == 0 {
		return pc, fmt.Errorf("name parameter of terminal %s must not be empty", terminalName)
	}

	id := viper.GetInt(fmt.Sprintf("%s.id", terminalName))

	pc.Name = name
	pc.ID = id

	return pc, nil
}
