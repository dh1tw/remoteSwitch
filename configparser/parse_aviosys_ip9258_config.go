package configparser

import (
	"fmt"

	ip9258 "github.com/dh1tw/remoteSwitch/switch/aviosys_ip9258"
	"github.com/spf13/viper"
)

// GetIP9258Config tries to parse the config file via viper
// and returns on success a ip9258.SwitchConfig object.
func GetIP9258Config(switchName string) ([]func(*ip9258.IP9258), error) {

	// let's check first if all necessary keys exist in the config file
	if !viper.IsSet(fmt.Sprintf("%s.name", switchName)) {
		return nil, fmt.Errorf("missing name parameter for switch %s", switchName)
	}

	if !viper.IsSet(fmt.Sprintf("%s.index", switchName)) {
		return nil, fmt.Errorf("missing index parameter for switch %s", switchName)
	}

	if !viper.IsSet(fmt.Sprintf("%s.username", switchName)) {
		return nil, fmt.Errorf("missing username for switch %s", switchName)
	}

	if !viper.IsSet(fmt.Sprintf("%s.password", switchName)) {
		return nil, fmt.Errorf("missing password for switch %s", switchName)
	}

	if !viper.IsSet(fmt.Sprintf("%s.url", switchName)) {
		return nil, fmt.Errorf("missing url or ip address for switch %s", switchName)
	}

	terminalNames := viper.GetStringSlice(fmt.Sprintf("%s.terminals", switchName))
	if len(terminalNames) == 0 {
		return nil, fmt.Errorf("no terminals found for device %s", switchName)
	}

	name := ip9258.Name(viper.GetString(fmt.Sprintf("%s.name", switchName)))
	index := ip9258.Index(viper.GetInt(fmt.Sprintf("%s.index", switchName)))
	username := ip9258.Username(viper.GetString(fmt.Sprintf("%s.username", switchName)))
	password := ip9258.Password(viper.GetString(fmt.Sprintf("%s.password", switchName)))
	url := ip9258.URL(viper.GetString(fmt.Sprintf("%s.url", switchName)))

	terms := []ip9258.Terminal{}

	for _, tName := range terminalNames {
		t, err := getIP9258TerminalConfig(tName)
		if err != nil {
			return nil, err
		}
		terms = append(terms, t)
	}

	terminals := ip9258.Terminals(terms)

	opts := []func(*ip9258.IP9258){name, index, username, password, url, terminals}
	return opts, nil

}

func getIP9258TerminalConfig(terminalName string) (ip9258.Terminal, error) {

	t := ip9258.Terminal{}

	// let's check first if all necessary keys exist in the config file
	if !viper.IsSet(fmt.Sprintf("%s.name", terminalName)) {
		return t, fmt.Errorf("missing name parameter for terminal %s", terminalName)
	}

	if !viper.IsSet(fmt.Sprintf("%s.index", terminalName)) {
		return t, fmt.Errorf("missing index parameter for terminal %s", terminalName)
	}

	if !viper.IsSet(fmt.Sprintf("%s.outlet", terminalName)) {
		return t, fmt.Errorf("missing outlet parameter for terminal %s", terminalName)
	}

	// get the values
	name := viper.GetString(fmt.Sprintf("%s.name", terminalName))
	if len(name) == 0 {
		return t, fmt.Errorf("name parameter of terminal %s must not be empty", terminalName)
	}

	index := viper.GetInt(fmt.Sprintf("%s.index", terminalName))
	outlet := viper.GetInt(fmt.Sprintf("%s.outlet", terminalName))

	t.Name = name
	t.Index = index
	t.Outlet = outlet

	return t, nil
}
