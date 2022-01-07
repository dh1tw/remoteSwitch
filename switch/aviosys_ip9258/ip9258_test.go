package ip9258

import (
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"

	sw "github.com/dh1tw/remoteSwitch/switch"
)

func TestMain(m *testing.M) {
	log.SetOutput(os.Stdout)
	log.Println("starting IP9258 unit tests")
	os.Exit(m.Run())
}

func Test_parseTerminalState(t *testing.T) {
	type args struct {
		input string
	}
	tests := []struct {
		name    string
		args    args
		want    int
		want1   bool
		wantErr bool
	}{
		{"valid terminal (1) and valid state (true)", args{"p61=1"}, 1, true, false},
		{"valid terminal (4) and valid state (false)", args{"p64=0"}, 4, false, false},
		{"invalid terminal (x) and valid state (false)", args{"p6x=0"}, 0, false, true},
		{"invalid state ('2')", args{"p61=2"}, 1, false, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1, err := parseTerminalState(tt.args.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("decodeState() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("decodeState() got = %v, want %v", got, tt.want)
			}
			if got1 != tt.want1 {
				t.Errorf("decodeState() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}

func Test_createDeviceURL(t *testing.T) {
	type args struct {
		url      string
		username string
		password string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{"create valid device URL", args{
			url:      "192.168.0.10",
			username: "testAccount",
			password: "12345",
		}, "http://testAccount:12345@192.168.0.10"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := createDeviceURL(tt.args.username, tt.args.password, tt.args.url); got != tt.want {
				t.Errorf("createDeviceURL() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIP9258_updateTerminal(t *testing.T) {
	type fields struct {
		terminals map[int]*Terminal
	}
	type args struct {
		terminal int
		newState bool
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    bool
		wantErr bool
	}{
		{
			"update state of valid terminal",
			fields{
				terminals: map[int]*Terminal{
					0: &Terminal{},
				},
			},
			args{0, true},
			true, false,
		},
		{
			"call method with valid terminal, but same state",
			fields{
				terminals: map[int]*Terminal{
					0: &Terminal{},
					1: &Terminal{state: true},
				},
			},
			args{0, true},
			true, false,
		},
		{
			"try to update state of not existing terminal",
			fields{
				terminals: map[int]*Terminal{
					0: &Terminal{},
				},
			},
			args{1, true},
			false, true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &IP9258{
				terminals: tt.fields.terminals,
			}
			got, err := s.updateTerminal(tt.args.terminal, tt.args.newState)
			if (err != nil) != tt.wantErr {
				t.Errorf("IP9258.updateTerminal() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("IP9258.updateTerminal() = %v, want %v", got, tt.want)
			}
		})
	}
}

func getTerminalStatePattern() *regexp.Regexp {

	tsp, _ := regexp.Compile("p6[1-4]=[0-1]")
	return tsp
}

func TestIP9258_updateTerminals(t *testing.T) {

	type fields struct {
		RWMutex              sync.RWMutex
		terminals            map[int]*Terminal
		terminalStatePattern *regexp.Regexp
		eventHandler         func(sw.Switcher, sw.Device)
	}
	type args struct {
		text string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{"update 3 of 4 terminals",
			fields{
				terminals: map[int]*Terminal{
					1: &Terminal{state: false},
					2: &Terminal{state: false},
					3: &Terminal{state: false},
					4: &Terminal{state: false},
				},
			},
			args{"<html>p61=1,p62=1,p63=0,p64=1</html>"},
			false},
		{"malformatted string must throw error",
			fields{
				terminals: map[int]*Terminal{
					1: &Terminal{state: false},
					2: &Terminal{state: false},
					3: &Terminal{state: false},
					4: &Terminal{state: false},
				},
			},
			args{"<html>p61=xx,p62=Y,p63=Z,p64=s</html>"},
			true},
		{"trying to update not existing terminal must throw error",
			fields{
				terminals: map[int]*Terminal{},
			},
			args{"<html>p65=1</html>"},
			true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &IP9258{
				terminals:            tt.fields.terminals,
				terminalStatePattern: getTerminalStatePattern(),
				eventHandler:         tt.fields.eventHandler,
			}
			if err := s.updateTerminals(tt.args.text); (err != nil) != tt.wantErr {
				t.Errorf("IP9258.updateTerminals() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// handlerFunc for httptest.Server trying to cover the most common
// happy path cases
func handleDefaultRequest(rw http.ResponseWriter, req *http.Request) {
	url := req.URL.String()

	if strings.Contains(url, "set.cmd?cmd=getpower") {
		rw.Write([]byte("<html>p61=1,p62=1,p63=1,p64=1</html>"))
		return
		// setting terminal 1 to 0
	} else if strings.Contains(url, "set.cmd?cmd=setpower+p61=0") {
		rw.Write([]byte("<html>p61=1</html>"))
		return
		// setting terminal 1 to 1
	} else if strings.Contains(url, "set.cmd?cmd=setpower+p61=1") {
		rw.Write([]byte("<html>p61=1</html>"))
		return
		// invalid state '2'
	} else if strings.Contains(url, "set.cmd?cmd=setpower+p62=2") {
		rw.Write([]byte("<html>HTTPCMD_BADPARAM</html>"))
		return
		// for setting terminal 6, which doesn't exist ip9258 returns
		// the state of terminal 1
	} else if strings.Contains(url, "set.cmd?cmd=setpower+p66=1") {
		rw.Write([]byte("<html>p61=1</html>"))
		return
	} else {
		rw.WriteHeader(401) // unauthorized
	}
}

// handlerfunc for httptest.Server returning a 401 when either the
// credentials are wrong or the URL is invalid
func handleRequest401(rw http.ResponseWriter, req *http.Request) {
	rw.WriteHeader(401) // unauthorized
}

// ip9258Mock returns an httptest.Server to test the HTTP Get commands
// excecuted by some functions. By providing the param the behaviour
// of the server can be changed:
// - any string (happy path testing)
// - '401' -> returns always 401
func ip9258Mock(param string) *httptest.Server {

	switch param {
	case "401":
		return httptest.NewServer(http.HandlerFunc(handleRequest401))
	default:
		return httptest.NewServer(http.HandlerFunc(handleDefaultRequest))
	}
}

func TestIP9258_setTerminal(t *testing.T) {
	type fields struct {
		terminals            map[int]*Terminal
		terminalStatePattern *regexp.Regexp
	}
	type args struct {
		terminal    int
		newstate    bool
		serverParms string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{"set terminal 1 to true",
			fields{
				terminals: map[int]*Terminal{
					1: &Terminal{state: false},
				},
			},
			args{1, true, "default"},
			false},
		{"set terminal 1 to false",
			fields{
				terminals: map[int]*Terminal{
					1: &Terminal{state: false},
				},
			},
			args{1, false, "default"},
			false},
		{"use wrong credentials resulting in a 401 status reply",
			fields{
				terminals: map[int]*Terminal{
					1: &Terminal{state: false},
				},
			},
			args{1, false, "401"},
			true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := ip9258Mock(tt.args.serverParms)
			defer server.Close()

			s := &IP9258{
				terminals:            tt.fields.terminals,
				url:                  server.URL,
				terminalStatePattern: getTerminalStatePattern(),
			}

			if err := s.setTerminal(tt.args.terminal, tt.args.newstate); (err != nil) != tt.wantErr {
				t.Errorf("IP9258.setTerminal() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestIP9258_queryTerminalStatus(t *testing.T) {
	type args struct {
		mockServer  bool
		serverParms string
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{"get state of all terminals",
			args{true, "default"},
			"<html>p61=1,p62=1,p63=1,p64=1</html>", false,
		},
		{"use wrong credentials resulting in a 401 status reply",
			args{true, "401"},
			"", true,
		},
		{"query a non-existing IP Address must result in a timeout",
			args{false, ""},
			"", true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			server := ip9258Mock(tt.args.serverParms)
			defer server.Close()

			s := &IP9258{
				url: server.URL,
			}

			if !tt.args.mockServer {
				s.url = "http://192.0.2.0"
			}

			got, err := s.queryTerminalStatus()
			if (err != nil) != tt.wantErr {
				t.Errorf("IP9258.queryTerminalStatus() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("IP9258.queryTerminalStatus() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIP9258_getTerminal(t *testing.T) {
	t1 := &Terminal{Name: "term1", state: false}
	t2 := &Terminal{Name: "term2", state: true}

	type fields struct {
		RWMutex   sync.RWMutex
		terminals map[int]*Terminal
	}
	type args struct {
		name string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *Terminal
		wantErr bool
	}{
		{"get existing terminal",
			fields{
				terminals: map[int]*Terminal{
					1: t1,
					2: t2,
				},
			},
			args{"term2"},
			t2, false,
		},
		{"try to get non-existing terminal",
			fields{
				terminals: map[int]*Terminal{
					1: t1,
					2: t2,
				},
			},
			args{"term3"},
			nil, true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := &IP9258{
				terminals: tt.fields.terminals,
			}
			got, err := d.getTerminal(tt.args.name)
			if (err != nil) != tt.wantErr {
				t.Errorf("IP9258.getTerminal() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("IP9258.getTerminal() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIP9258_serialize(t *testing.T) {

	dev := sw.Device{
		Name:  "IP9258 Web Switch",
		Index: 5,
		Ports: []sw.Port{
			sw.Port{
				Name:  "PS",
				Index: 0,
				Terminals: []sw.Terminal{
					// returned list must be ordered by index
					sw.Terminal{Name: "term1", Index: 1, State: true},
					sw.Terminal{Name: "term2", Index: 2, State: false},
					sw.Terminal{Name: "term3", Index: 3, State: false},
					sw.Terminal{Name: "term4", Index: 4, State: true},
				},
			},
		},
	}

	type fields struct {
		name      string
		index     int
		portName  string
		terminals map[int]*Terminal
	}
	tests := []struct {
		name   string
		fields fields
		want   sw.Device
	}{
		{
			name: "get a valid device",
			fields: fields{
				name:     "IP9258 Web Switch",
				index:    5,
				portName: "PS",
				terminals: map[int]*Terminal{
					1: &Terminal{Name: "term1", Outlet: 1, Index: 1, state: true},
					2: &Terminal{Name: "term2", Outlet: 2, Index: 2, state: false},
					4: &Terminal{Name: "term4", Outlet: 4, Index: 4, state: true},
					3: &Terminal{Name: "term3", Outlet: 3, Index: 3, state: false},
				},
			},
			want: dev,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := &IP9258{
				name:      tt.fields.name,
				index:     tt.fields.index,
				portName:  tt.fields.portName,
				terminals: tt.fields.terminals,
			}
			if got := d.serialize(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("IP9258.serialize() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIP9258_getPort(t *testing.T) {

	t1 := &Terminal{Name: "term1", Index: 1, Outlet: 1, state: true}
	t2 := &Terminal{Name: "term2", Index: 2, Outlet: 2, state: false}
	// term4 and term3 are switched; we want to make sure that
	// the sorting algorithm works properly (sorted by index)
	t3 := &Terminal{Name: "term4", Index: 4, Outlet: 4, state: true}
	t4 := &Terminal{Name: "term3", Index: 3, Outlet: 3, state: false}

	p := sw.Port{
		Name:  "PS",
		Index: 0,
		Terminals: []sw.Terminal{
			// returned list must be ordered by index
			sw.Terminal{Name: "term1", Index: 1, State: true},
			sw.Terminal{Name: "term2", Index: 2, State: false},
			sw.Terminal{Name: "term3", Index: 3, State: false},
			sw.Terminal{Name: "term4", Index: 4, State: true},
		},
	}

	type fields struct {
		portName  string
		terminals map[int]*Terminal
	}
	tests := []struct {
		name   string
		fields fields
		want   sw.Port
	}{
		{"get port with an arbitrary name",
			fields{
				portName: "PS",
				terminals: map[int]*Terminal{
					1: t1,
					2: t2,
					3: t3,
					4: t4,
				},
			},
			p},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := &IP9258{
				terminals: tt.fields.terminals,
				portName:  tt.fields.portName,
			}
			if got := d.getPort(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("IP9258.getPort() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIP9258_SetPort(t *testing.T) {
	type fields struct {
		terminals map[int]*Terminal
	}
	type args struct {
		portRequest sw.Port
		serverParms string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "set a port with a single valid Terminal",
			fields: fields{
				terminals: map[int]*Terminal{
					1: &Terminal{Name: "term1", Outlet: 1, Index: 1, state: true},
					2: &Terminal{Name: "term2", Outlet: 2, Index: 2, state: false},
					4: &Terminal{Name: "term4", Outlet: 4, Index: 4, state: true},
					3: &Terminal{Name: "term3", Outlet: 3, Index: 3, state: false},
				},
			},
			args: args{
				sw.Port{
					Terminals: []sw.Terminal{
						sw.Terminal{
							Name:  "term1",
							State: true,
						},
					},
				},
				"default",
			},
			wantErr: false,
		},
		{
			name: "set a port with one valid and one non-existing Terminal",
			fields: fields{
				terminals: map[int]*Terminal{
					1: &Terminal{Name: "term1", Outlet: 1, Index: 1, state: true},
					2: &Terminal{Name: "term2", Outlet: 2, Index: 2, state: false},
					4: &Terminal{Name: "term4", Outlet: 4, Index: 4, state: true},
					3: &Terminal{Name: "term3", Outlet: 3, Index: 3, state: false},
				},
			},
			args: args{
				sw.Port{
					Terminals: []sw.Terminal{
						sw.Terminal{
							Name:  "term1",
							State: true,
						},
						sw.Terminal{
							Name:  "term5",
							State: true,
						},
					},
				},
				"default",
			},
			wantErr: false,
		},
		{
			name: "try to set the port with invalid credentials",
			fields: fields{
				terminals: map[int]*Terminal{
					1: &Terminal{Name: "term1", Outlet: 1, Index: 1, state: true},
					2: &Terminal{Name: "term2", Outlet: 2, Index: 2, state: false},
					4: &Terminal{Name: "term4", Outlet: 4, Index: 4, state: true},
					3: &Terminal{Name: "term3", Outlet: 3, Index: 3, state: false},
				},
			},
			args: args{
				sw.Port{
					Terminals: []sw.Terminal{
						sw.Terminal{
							Name:  "term1",
							State: true,
						},
					},
				},
				"401",
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			server := ip9258Mock(tt.args.serverParms)
			defer server.Close()

			d := &IP9258{
				terminals:            tt.fields.terminals,
				url:                  server.URL,
				terminalStatePattern: getTerminalStatePattern(),
			}

			if err := d.SetPort(tt.args.portRequest); (err != nil) != tt.wantErr {
				t.Errorf("IP9258.SetPort() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestIP9258_Close(t *testing.T) {
	d := &IP9258{
		pollingTicker: time.NewTicker(time.Second * 5),
		stopPolling:   make(chan struct{}),
	}

	t.Log("launch")

	closed := 0

	go func() {
		t.Log("launched go routine")
		<-d.stopPolling
		closed += 1
	}()

	t.Log("closing 1st time")
	d.Close()
	// // try to close a second time
	d.Close()

	// if closed != 1 {
	// 	t.Fatalf("close method executed twice")
	// }
}
