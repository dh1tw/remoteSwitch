package configparser

import (
	"fmt"

	rb "github.com/dh1tw/remoteSwitch/switch/ea4tx_remotebox"
	"github.com/spf13/viper"
)

// GetEA4TXRemoteboxConfig tries to parse the config file via viper
// and returns on success an array of functional options.
func GetEA4TXRemoteboxConfig(switchName string) ([]func(*rb.Remotebox), error) {

	// let's check first if all necessary keys exist in the config file
	if !viper.IsSet(fmt.Sprintf("%s.name", switchName)) {
		return nil, fmt.Errorf("missing name parameter for switch %s", switchName)
	}

	if !viper.IsSet(fmt.Sprintf("%s.index", switchName)) {
		return nil, fmt.Errorf("missing index parameter for switch %s", switchName)
	}

	if !viper.IsSet(fmt.Sprintf("%s.portname", switchName)) {
		return nil, fmt.Errorf("missing portname parameter for switch %s", switchName)
	}

	name := rb.Name(viper.GetString(fmt.Sprintf("%s.name", switchName)))
	index := rb.Index(viper.GetInt(fmt.Sprintf("%s.index", switchName)))
	portname := rb.Portname(viper.GetString(fmt.Sprintf("%s.portname", switchName)))
	// new options for TCP connection
	Ipaddress := rb.Ipaddress(viper.GetString(fmt.Sprintf("%s.ipaddress", switchName)))
	Ipport := rb.Ipport(viper.GetInt(fmt.Sprintf("%s.ipport", switchName)))
	Connection := rb.Connection(viper.GetInt(fmt.Sprintf("%s.connection", switchName)))

	opts := []func(*rb.Remotebox){name, index, portname, Ipaddress, Ipport, Connection}

	return opts, nil
}
