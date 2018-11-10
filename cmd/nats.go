package cmd

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"time"

	"github.com/dh1tw/remoteSwitch/hub"
	sw "github.com/dh1tw/remoteSwitch/switch"
	bsGPIO "github.com/dh1tw/remoteSwitch/switch/bandswitch-gpio"
	micro "github.com/micro/go-micro"
	"github.com/micro/go-micro/server"
	natsBroker "github.com/micro/go-plugins/broker/nats"
	natsReg "github.com/micro/go-plugins/registry/nats"
	natsTr "github.com/micro/go-plugins/transport/nats"
	"github.com/nats-io/go-nats"
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

	natsServerCmd.Flags().StringP("type", "t", "bandswitch", "Switch type (supported: bandswitch, stackmatch")
	natsServerCmd.Flags().StringP("name", "n", "mySwitch", "Name tag for the Switch")
	natsServerCmd.Flags().StringP("broker-url", "u", "localhost", "Broker URL")
	natsServerCmd.Flags().IntP("broker-port", "p", 4222, "Broker Port")
	natsServerCmd.Flags().StringP("password", "P", "", "NATS Password")
	natsServerCmd.Flags().StringP("username", "U", "", "NATS Username")
}

var configA = bsGPIO.PortConfig{
	Name: "A",
	ID:   0,
	OutPorts: []bsGPIO.PinConfig{
		bsGPIO.PinConfig{
			Name:     "160m",
			Pin:      "GPIO3",
			Inverted: true,
			ID:       0,
		},
		bsGPIO.PinConfig{
			Name:     "80m",
			Pin:      "GPIO19",
			Inverted: true,
			ID:       1,
		},
		bsGPIO.PinConfig{
			Name:     "40m",
			Pin:      "GPIO18",
			Inverted: true,
			ID:       2,
		},
		bsGPIO.PinConfig{
			Name:     "20m",
			Pin:      "GPIO15",
			Inverted: true,
			ID:       3,
		},
		bsGPIO.PinConfig{
			Name:     "15m",
			Pin:      "GPIO16",
			Inverted: true,
			ID:       4,
		},
		bsGPIO.PinConfig{
			Name:     "10m",
			Pin:      "GPIO2",
			Inverted: true,
			ID:       5,
		},
		bsGPIO.PinConfig{
			Name:     "6m",
			Pin:      "GPIO14",
			Inverted: true,
			ID:       6,
		},
		bsGPIO.PinConfig{
			Name:     "WARC",
			Pin:      "GPIO13",
			Inverted: true,
			ID:       7,
		},
	},
}

var configB = bsGPIO.PortConfig{
	Name: "B",
	ID:   1,
	OutPorts: []bsGPIO.PinConfig{
		bsGPIO.PinConfig{
			Name:     "160m",
			Pin:      "GPIO7",
			Inverted: true,
			ID:       0,
		},
		bsGPIO.PinConfig{
			Name:     "80m",
			Pin:      "GPIO0",
			Inverted: true,
			ID:       1,
		},
		bsGPIO.PinConfig{
			Name:     "40m",
			Pin:      "GPIO199",
			Inverted: true,
			ID:       2,
		},
		bsGPIO.PinConfig{
			Name:     "20m",
			Pin:      "GPIO1",
			Inverted: true,
			ID:       3,
		},
		bsGPIO.PinConfig{
			Name:     "15m",
			Pin:      "GPIO6",
			Inverted: true,
			ID:       4,
		},
		bsGPIO.PinConfig{
			Name:     "10m",
			Pin:      "GPIO198",
			Inverted: true,
			ID:       5,
		},
		bsGPIO.PinConfig{
			Name:     "6m",
			Pin:      "GPIO12",
			Inverted: true,
			ID:       6,
		},
		bsGPIO.PinConfig{
			Name:     "WARC",
			Pin:      "GPIO11",
			Inverted: true,
			ID:       7,
		},
	},
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

	viper.BindPFlag("switch.type", cmd.Flags().Lookup("type"))
	viper.BindPFlag("switch.name", cmd.Flags().Lookup("name"))
	viper.BindPFlag("nats.broker-url", cmd.Flags().Lookup("broker-url"))
	viper.BindPFlag("nats.broker-port", cmd.Flags().Lookup("broker-port"))
	viper.BindPFlag("nats.password", cmd.Flags().Lookup("password"))
	viper.BindPFlag("nats.username", cmd.Flags().Lookup("username"))

	// Profiling (uncomment if needed)
	// go func() {
	// 	log.Println(http.ListenAndServe("0.0.0.0:6060", http.DefaultServeMux))
	// }()

	// // struct which holds the rotator.Rotator instance, implements the
	// // RPC Service methods and publishes changes via the Broker
	// rpcRot := &rpcRotator{}

	// rotatorError := make(chan struct{})

	// // initialize our Rotator
	// r, err := initRotator(viper.GetString("rotator.type"), rpcRot.PublishState, rotatorError)
	// if err != nil {
	// 	fmt.Println("unable to initialize rotator:", err)
	// 	os.Exit(1)
	// }

	// better call this Addrs(?)
	serviceName := fmt.Sprintf("shackbus.switch.%s", viper.GetString("switch.name"))

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
	rs := micro.NewService(
		micro.Name(serviceName),
		micro.RegisterInterval(time.Second*10),
		micro.Broker(br),
		micro.Transport(tr),
		micro.Registry(reg),
		micro.Version(version),
		micro.Server(svr),
	)

	// initalize our service
	rs.Init()

	bcast := make(chan sw.Device, 10)

	var rEventHandler = func(r sw.Switcher, cfg sw.Device) {
		bcast <- cfg
	}

	sw := bsGPIO.NewSwitchGPIO(bsGPIO.Port(configA),
		bsGPIO.Port(configB), bsGPIO.Name(viper.GetString("switch.name")),
		bsGPIO.EventHandler(rEventHandler))
	if err := sw.Init(); err != nil {
		log.Fatal(err)
	}

	h, err := hub.NewHub(sw)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	errCh := make(chan struct{})

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)

	go func() {
		for {
			select {
			case <-errCh:
				log.Println("hub error")
				os.Exit(1)
			case msg := <-bcast:
				h.Broadcast(msg)
			case <-c:
				os.Exit(0)
			}
		}
	}()

	h.ListenHTTP("0.0.0.0", 6565, errCh)
}
