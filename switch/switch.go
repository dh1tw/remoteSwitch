package Switch

type Switcher interface {
	Name() string
	ID() int
	GetPort(portName string) (port Port, err error)
	SetPort(port Port) error
	Serialize() Device
	Close()
}

type Device struct {
	Name  string `json:"name,omitempty"`
	ID    int    `json:"id,omitempty"`
	Ports []Port `json:"ports,omitempty"`
}

type Port struct {
	Name      string     `json:"name,omitempty"`
	ID        int        `json:"id,omitempty"`
	Terminals []Terminal `json:"terminals,omitempty"`
}

type Terminal struct {
	Name  string `json:"name,omitempty"`
	ID    int    `json:"id,omitempty"`
	State bool   `json:"state,omitempty"`
}
