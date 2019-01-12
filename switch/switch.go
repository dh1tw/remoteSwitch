package Switch

type Switcher interface {
	Name() string
	GetPort(portName string) (port Port, err error)
	SetPort(port Port) error
	Serialize() Device
	Close()
}

type Device struct {
	Name  string `json:"name,omitempty"`
	Index int    `json:"index,omitempty"`
	Ports []Port `json:"ports,omitempty"`
}

type Port struct {
	Name      string     `json:"name,omitempty"`
	Index     int        `json:"index,omitempty"`
	Terminals []Terminal `json:"terminals,omitempty"`
}

type Terminal struct {
	Name  string `json:"name,omitempty"`
	Index int    `json:"index,omitempty"`
	State bool   `json:"state,omitempty"`
}
