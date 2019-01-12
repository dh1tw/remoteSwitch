package cmd

import (
	"fmt"

	smGPIO "github.com/dh1tw/remoteSwitch/switch/stackmatch_gpio"
	"github.com/spf13/viper"
)

func getSmGPIOConfig(smName string) (smGPIO.SmConfig, error) {

	sc := smGPIO.SmConfig{}

	// let's check first if all necessary keys exist in the config file
	if !viper.IsSet(fmt.Sprintf("%s.name", smName)) {
		return sc, fmt.Errorf("missing name parameter for stackmatch %s", smName)
	}

	if !viper.IsSet(fmt.Sprintf("%s.index", smName)) {
		return sc, fmt.Errorf("missing index parameter for stackmatch %s", smName)
	}

	if !viper.IsSet(fmt.Sprintf("%s.combinations", smName)) {
		return sc, fmt.Errorf("missing combinations parameter for stackmatch %s", smName)
	}

	// get the value
	name := viper.GetString(fmt.Sprintf("%s.name", smName))
	if len(name) == 0 {
		return sc, fmt.Errorf("name parameter of stackmatch %s must not be empty", smName)
	}

	index := viper.GetInt(fmt.Sprintf("%s.index", smName))

	combinations := viper.GetStringSlice(fmt.Sprintf("%s.combinations", smName))
	if len(combinations) == 0 {
		return sc, fmt.Errorf("no combinations found for stackmatch %s", smName)
	}

	sc.Name = name
	sc.Index = index

	for _, combination := range combinations {
		c, err := getSmGPIOCombinationConfig(combination)
		if err != nil {
			return sc, err
		}
		sc.Combinations = append(sc.Combinations, c)
	}

	return sc, nil
}

func getSmGPIOCombinationConfig(cName string) (smGPIO.CombinationConfig, error) {

	cc := smGPIO.CombinationConfig{}

	// let's check first if all necessary keys exist in the config file
	if !viper.IsSet(cName) {
		return cc, fmt.Errorf("no configuration found for combination %s", cName)
	}

	if !viper.IsSet(fmt.Sprintf("%s.terminals", cName)) {
		return cc, fmt.Errorf("missing terminals parameter for combination %s", cName)
	}

	if !viper.IsSet(fmt.Sprintf("%s.pins", cName)) {
		return cc, fmt.Errorf("missing pins parameter for combination %s", cName)
	}

	// get the values
	terminals := viper.GetStringSlice(fmt.Sprintf("%s.terminals", cName))
	if len(terminals) == 0 {
		return cc, fmt.Errorf("no terminals found for combination %s", cName)
	}

	pins := viper.GetStringSlice(fmt.Sprintf("%s.pins", cName))
	if len(pins) == 0 {
		return cc, fmt.Errorf("no pins found for combination %s", cName)
	}

	cc.Terminals = make([]smGPIO.TerminalConfig, 0, len(terminals))
	cc.Pins = make([]smGPIO.PinConfig, 0, len(pins))

	for _, terminal := range terminals {
		t, err := getSmGPIOTerminalConfig(terminal)
		if err != nil {
			return cc, err
		}
		cc.Terminals = append(cc.Terminals, t)
	}

	for _, pin := range pins {
		p, err := getSmGPIOPinConfig(pin)
		if err != nil {
			return cc, err
		}
		cc.Pins = append(cc.Pins, p)
	}

	return cc, nil
}

func getSmGPIOTerminalConfig(terminalName string) (smGPIO.TerminalConfig, error) {

	tc := smGPIO.TerminalConfig{}

	// let's check first if all necessary keys exist in the config file
	if !viper.IsSet(fmt.Sprintf("%s.name", terminalName)) {
		return tc, fmt.Errorf("missing name parameter for terminal %s", terminalName)
	}

	if !viper.IsSet(fmt.Sprintf("%s.index", terminalName)) {
		return tc, fmt.Errorf("missing index parameter for terminal %s", terminalName)
	}

	// get the values
	name := viper.GetString(fmt.Sprintf("%s.name", terminalName))
	if len(name) == 0 {
		return tc, fmt.Errorf("name parameter of terminal %s must not be empty", terminalName)
	}

	index := viper.GetInt(fmt.Sprintf("%s.index", terminalName))

	tc.Name = name
	tc.Index = index

	return tc, nil
}

func getSmGPIOPinConfig(pinName string) (smGPIO.PinConfig, error) {

	pc := smGPIO.PinConfig{}

	// let's check first if all necessary keys exist in the config file
	if !viper.IsSet(fmt.Sprintf("%s.pin", pinName)) {
		return pc, fmt.Errorf("missing pin parameter for pin %s", pinName)
	}

	// get the values
	pin := viper.GetString(fmt.Sprintf("%s.pin", pinName))
	if len(pin) == 0 {
		return pc, fmt.Errorf("pin parameter of pin %s must not be empty", pinName)
	}

	pc.Name = pinName
	pc.Inverted = viper.GetBool(fmt.Sprintf("%s.inverted", pinName))
	pc.Pin = pin

	return pc, nil
}
