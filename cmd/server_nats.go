package cmd

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/dh1tw/remoteSwitch/configparser"
	sbSwitch "github.com/dh1tw/remoteSwitch/sb_switch"
	sw "github.com/dh1tw/remoteSwitch/switch"
	ds "github.com/dh1tw/remoteSwitch/switch/dummy_switch"
	rb "github.com/dh1tw/remoteSwitch/switch/ea4tx_remotebox"
	mpGPIO "github.com/dh1tw/remoteSwitch/switch/multi-purpose-switch-gpio"
	smGPIO "github.com/dh1tw/remoteSwitch/switch/stackmatch_gpio"
	"github.com/gogo/protobuf/proto"
	micro "github.com/micro/go-micro"
	"github.com/micro/go-micro/broker"
	"github.com/micro/go-micro/server"
	natsBroker "github.com/micro/go-plugins/broker/nats"
	natsReg "github.com/micro/go-plugins/registry/nats"
	natsTr "github.com/micro/go-plugins/transport/nats"
	nats "github.com/nats-io/nats.go"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// natsCmd represents the nats command
var natsServerCmd = &cobra.Command{
	Use:   "nats",
	Short: "expose your Switch via a nats broker",
	Long: `
The nats server allows you to expose an Switch on a nats.io broker. The broker
can be located within your local lan or somewhere on the internet.`,
	Run: natsServer,
}

func init() {
	serverCmd.AddCommand(natsServerCmd)
	natsServerCmd.Flags().StringP("broker-url", "u", "localhost", "Broker URL")
	natsServerCmd.Flags().IntP("broker-port", "p", 4222, "Broker Port")
	natsServerCmd.Flags().StringP("password", "P", "", "NATS Password")
	natsServerCmd.Flags().StringP("username", "U", "", "NATS Username")
}

func natsServer(cmd *cobra.Command, args []string) {

	// Try to read config file
	if err := viper.ReadInConfig(); err == nil {
		fmt.Println("Using config file:", viper.ConfigFileUsed())
	} else {
		if strings.Contains(err.Error(), "Not Found in") {
			fmt.Println("no config file found")
		} else {
			fmt.Println("Error parsing config file", viper.ConfigFileUsed())
			fmt.Println(err)
			os.Exit(1)
		}
	}

	viper.BindPFlag("nats.broker-url", cmd.Flags().Lookup("broker-url"))
	viper.BindPFlag("nats.broker-port", cmd.Flags().Lookup("broker-port"))
	viper.BindPFlag("nats.password", cmd.Flags().Lookup("password"))
	viper.BindPFlag("nats.username", cmd.Flags().Lookup("username"))

	// Profiling (uncomment if needed)
	// go func() {
	// 	log.Println(http.ListenAndServe("0.0.0.0:6060", http.DefaultServeMux))
	// }()

	// // struct which holds the switch.Switcher instance, implements the
	// // RPC Service methods and publishes changes via the Broker
	rpcSwitch := &rpcSwitch{}

	switchError := make(chan struct{})

	if !viper.IsSet("switch.type") {
		log.Fatal("missing configuration for switch (switch.type)")
	}

	if !viper.IsSet("switch.name") {
		log.Fatal("missing configuration for switch (switch.name)")
	}

	switchType := viper.GetString("switch.type")
	switchName := viper.GetString("switch.name")

	switch switchType {
	case "multi_purpose_gpio":
		sc, err := configparser.GetMPGPIOSwitchConfig(switchName)
		if err != nil {
			log.Fatal(err)
		}
		sw := mpGPIO.NewMPSwitchGPIO(mpGPIO.Switch(sc),
			mpGPIO.EventHandler(rpcSwitch.PublishDeviceUpdate))

		if err := sw.Init(); err != nil {
			log.Fatal(err)
		}
		rpcSwitch.sw = sw

	case "dummy_switch":
		sc, err := configparser.GetDummySwitchConfig(switchName)
		if err != nil {
			log.Fatal(err)
		}
		sw := ds.NewDummySwitch(ds.Switch(sc), ds.EventHandler(rpcSwitch.PublishDeviceUpdate))

		if err := sw.Init(); err != nil {
			log.Fatal(err)
		}
		rpcSwitch.sw = sw

	case "stackmatch_gpio":
		sc, err := configparser.GetSmGPIOConfig(switchName)
		if err != nil {
			log.Fatal(err)
		}
		sw := smGPIO.NewStackmatchGPIO(smGPIO.Config(sc), smGPIO.EventHandler(rpcSwitch.PublishDeviceUpdate))
		if err := sw.Init(); err != nil {
			log.Fatal(err)
		}
		rpcSwitch.sw = sw

	case "ea4tx-remotebox":
		opts, err := configparser.GetEA4TXRemoteboxConfig(switchName)
		if err != nil {
			log.Fatal(err)
		}
		opts = append(opts, rb.EventHandler(rpcSwitch.PublishDeviceUpdate))
		sw := rb.New(opts...)
		if err := sw.Init(); err != nil {
			log.Fatal(err)
		}
		rpcSwitch.sw = sw
		fmt.Println(sw.Name())
	default:
		log.Fatalf("unknown switch type %s", switchType)
	}

	// better call this Addrs(?)
	serviceName := fmt.Sprintf("shackbus.switch.%s", rpcSwitch.sw.Name())

	username := viper.GetString("nats.username")
	password := viper.GetString("nats.password")
	url := viper.GetString("nats.broker-url")
	port := viper.GetInt("nats.broker-port")
	addr := fmt.Sprintf("nats://%s:%v", url, port)

	// start from default nats config and add the common options
	nopts := nats.GetDefaultOptions()
	nopts.Servers = []string{addr}
	nopts.User = username
	nopts.Password = password

	regNatsOpts := nopts
	brNatsOpts := nopts
	trNatsOpts := nopts
	// we want to set the nats.Options.Name so that we can distinguish
	// them when monitoring the nats server with nats-top
	regNatsOpts.Name = serviceName + ":registry"
	brNatsOpts.Name = serviceName + ":broker"
	trNatsOpts.Name = serviceName + ":transport"

	// create instances of our nats Registry, Broker and Transport
	reg := natsReg.NewRegistry(natsReg.Options(regNatsOpts))
	br := natsBroker.NewBroker(natsBroker.Options(brNatsOpts))
	tr := natsTr.NewTransport(natsTr.Options(trNatsOpts))

	// this is a workaround since we must set server.Address with the
	// sanitized version of our service name. The server.Address will be
	// used in nats as the topic on which the server (transport) will be
	// listening on.
	svr := server.NewServer(
		server.Name(serviceName),
		server.Address(validateSubject(serviceName)),
		server.RegisterInterval(time.Second*10),
		server.Transport(tr),
		server.Registry(reg),
		server.Broker(br),
	)

	// version is typically defined through a git tag and injected during
	// compilation; if not, just set it to "dev"
	if version == "" {
		version = "dev"
	}

	// let's create the new rotator service
	ss := micro.NewService(
		micro.Name(serviceName),
		micro.Broker(br),
		micro.Transport(tr),
		micro.Registry(reg),
		micro.Version(version),
		micro.Server(svr),
	)

	// initialize our service
	ss.Init()

	// before we annouce this service, we have to ensure that no other
	// service with the same name exists. Therefore we query the
	// registry for all other existing services.
	services, err := reg.ListServices()
	if err != nil {
		log.Fatal(err)
	}

	// if a service with this name already exists, then exit
	for _, service := range services {
		if service.Name == serviceName {
			log.Fatalf("service '%s' already exists", service.Name)
		}
	}

	rpcSwitch.Lock()
	rpcSwitch.service = ss
	rpcSwitch.pubSubTopic = fmt.Sprintf("%s.state", strings.Replace(serviceName, " ", "_", -1))

	// register our Rotator RPC handler
	sbSwitch.RegisterSbSwitchHandler(ss.Server(), rpcSwitch)

	rpcSwitch.initialized = true
	rpcSwitch.Unlock()

	go func() {
		for {
			select {
			case <-switchError:
				ss.Server().Stop()
				os.Exit(1)
			}
		}
	}()

	if err := ss.Run(); err != nil {
		log.Println(err)
		os.Exit(1)
	}
}

type rpcSwitch struct {
	sync.Mutex
	initialized bool
	service     micro.Service
	sw          sw.Switcher
	pubSubTopic string
}

func (s *rpcSwitch) PublishDeviceUpdate(swi sw.Switcher, d sw.Device) {

	s.Lock()
	defer s.Unlock()
	if !s.initialized {
		return
	}

	data, err := proto.Marshal(deviceToSbDevice(swi.Serialize()))
	if err != nil {
		log.Println(err)
		return
	}

	msg := broker.Message{
		Body: data,
	}

	if err := s.service.Options().Broker.Publish(s.pubSubTopic, &msg); err != nil {
		log.Println(err)
	}
}

func (s *rpcSwitch) GetPort(ctx context.Context, portName *sbSwitch.PortName, port *sbSwitch.Port) error {
	p, err := s.sw.GetPort(portName.GetName())
	if err != nil {
		return err
	}

	myPort := portToSbPort(p)
	port.Name = myPort.GetName()
	port.Index = myPort.GetIndex()
	port.Exclusive = myPort.GetExclusive()
	port.Terminals = myPort.GetTerminals()

	return nil
}

func (s *rpcSwitch) SetPort(ctx context.Context, portReq *sbSwitch.PortRequest, out *sbSwitch.None) error {

	port := sw.Port{
		Name:      portReq.GetName(),
		Terminals: []sw.Terminal{},
	}

	for _, t := range portReq.GetTerminals() {
		terminal := sw.Terminal{
			Name:  t.GetName(),
			State: t.GetState(),
		}
		port.Terminals = append(port.Terminals, terminal)
	}

	return s.sw.SetPort(port)
}

func (s *rpcSwitch) GetDevice(ctx context.Context, in *sbSwitch.None, sbDevice *sbSwitch.Device) error {

	myDevice := deviceToSbDevice(s.sw.Serialize())

	sbDevice.Name = myDevice.GetName()
	sbDevice.Index = myDevice.GetIndex()
	sbDevice.Exclusive = myDevice.GetExclusive()
	sbDevice.Ports = myDevice.GetPorts()

	return nil
}

func deviceToSbDevice(device sw.Device) *sbSwitch.Device {

	sbDevice := &sbSwitch.Device{
		Name:  device.Name,
		Index: int32(device.Index),
		// Exclusive: device.Exclusive,
		Ports: []*sbSwitch.Port{},
	}

	for _, p := range device.Ports {
		sbDevice.Ports = append(sbDevice.Ports, portToSbPort(p))
	}
	return sbDevice
}

func portToSbPort(port sw.Port) *sbSwitch.Port {

	sbPort := &sbSwitch.Port{
		Name:  port.Name,
		Index: int32(port.Index),
		// Exclusive: port.Exclusive,
		Terminals: []*sbSwitch.Terminal{},
	}

	for _, t := range port.Terminals {
		sbTerminal := &sbSwitch.Terminal{
			Name:  t.Name,
			Index: int32(t.Index),
			State: t.State,
		}
		sbPort.Terminals = append(sbPort.Terminals, sbTerminal)
	}

	return sbPort
}
