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

	if !viper.IsSet(fmt.Sprintf("%s.ipaddress", switchName)) {
		return nil, fmt.Errorf("missing ipaddress parameter for switch %s", switchName)
	}

	name := rb.Name(viper.GetString(fmt.Sprintf("%s.name", switchName)))
	index := rb.Index(viper.GetInt(fmt.Sprintf("%s.index", switchName)))
	portname := rb.Portname(viper.GetString(fmt.Sprintf("%s.portname", switchName)))
	// new options for TCP connection
	rb_ipaddress := rb.Rb_IpAddress(viper.GetString(fmt.Sprintf("%s.ipaddress", switchName)))
	rb_ipport := rb.Rb_IpPort(viper.GetInt(fmt.Sprintf("%s.ipport", switchName)))
	rb_Connection := rb.Rb_Connection(viper.GetInt(fmt.Sprintf("%s.Connection", switchName)))

	opts := []func(*rb.Remotebox){name, index, portname, rb_ipaddress, rb_ipport,rb_Connection}

	return opts, nil
}
