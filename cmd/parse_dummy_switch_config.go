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

	if !viper.IsSet(fmt.Sprintf("%s.index", switchName)) {
		return sc, fmt.Errorf("missing index parameter for switch %s", switchName)
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

	index := viper.GetInt(fmt.Sprintf("%s.index", switchName))
	exclusive := viper.GetBool(fmt.Sprintf("%s.exclusive", switchName))
	ports := viper.GetStringSlice(fmt.Sprintf("%s.ports", switchName))
	if len(ports) == 0 {
		return sc, fmt.Errorf("no ports found for switch %s", switchName)
	}

	sc.Name = name
	sc.Index = index
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

	if !viper.IsSet(fmt.Sprintf("%s.index", portName)) {
		return pc, fmt.Errorf("missing index parameter for port %s", portName)
	}

	if !viper.IsSet(fmt.Sprintf("%s.terminals", portName)) {
		return pc, fmt.Errorf("missing terminals parameter for port %s", portName)
	}

	// get the values
	name := viper.GetString(fmt.Sprintf("%s.name", portName))
	if len(name) == 0 {
		return pc, fmt.Errorf("name parameter of port %s must not be empty", portName)
	}

	index := viper.GetInt(fmt.Sprintf("%s.index", portName))
	exclusive := viper.GetBool(fmt.Sprintf("%s.exclusive", portName))
	terminals := viper.GetStringSlice(fmt.Sprintf("%s.terminals", portName))
	if len(terminals) == 0 {
		return pc, fmt.Errorf("no terminals found for port %s", portName)
	}

	pc.Name = name
	pc.Index = index
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

	if !viper.IsSet(fmt.Sprintf("%s.index", terminalName)) {
		return pc, fmt.Errorf("missing index parameter for terminal %s", terminalName)
	}

	// get the values
	name := viper.GetString(fmt.Sprintf("%s.name", terminalName))
	if len(name) == 0 {
		return pc, fmt.Errorf("name parameter of terminal %s must not be empty", terminalName)
	}

	index := viper.GetInt(fmt.Sprintf("%s.index", terminalName))

	pc.Name = name
	pc.Index = index

	return pc, nil
}
