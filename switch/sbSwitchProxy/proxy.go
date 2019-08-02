package sbSwitchProxy

import (
	"context"
	"fmt"
	"sync"
	"time"

	sbSwitch "github.com/dh1tw/remoteSwitch/sb_switch"
	sw "github.com/dh1tw/remoteSwitch/switch"
	"github.com/gogo/protobuf/proto"
	"github.com/micro/go-micro/broker"
	"github.com/micro/go-micro/client"
)

type SbSwitchProxy struct {
	sync.RWMutex
	cli          client.Client
	scli         sbSwitch.SbSwitchService
	eventHandler func(sw.Switcher, sw.Device)
	device       sw.Device
	doneCh       chan struct{}
	doneOnce     sync.Once
	subscriber   broker.Subscriber
	serviceName  string
}

func New(opts ...func(*SbSwitchProxy)) (*SbSwitchProxy, error) {

	s := &SbSwitchProxy{
		device: sw.Device{
			Name:  "SwitchProxy",
			Index: 0,
			Ports: []sw.Port{},
		},
		serviceName: "shackbus.switch.mySwitch",
	}

	for _, opt := range opts {
		opt(s)
	}

	s.scli = sbSwitch.NewSbSwitchService(s.serviceName, s.cli)

	if err := s.getInfo(); err != nil {
		return nil, err
	}

	br := s.cli.Options().Broker
	if err := br.Connect(); err != nil {
		return nil, err
	}

	sub, err := br.Subscribe(s.serviceName+".state", s.updateHandler)
	if err != nil {
		return nil, err
	}
	s.subscriber = sub

	return s, nil
}

// the doneCh must be closed through this function to avoid
// multiple times closing this channel. Closing the doneCh signals the
// application that this object can be disposed
func (s *SbSwitchProxy) closeDone() {
	s.doneOnce.Do(func() { close(s.doneCh) })
}

func (s *SbSwitchProxy) updateHandler(p broker.Publication) error {

	sbDevice := sbSwitch.Device{}
	if err := proto.Unmarshal(p.Message().Body, &sbDevice); err != nil {
		return err
	}

	s.Lock()
	defer s.Unlock()

	s.device.Ports = []sw.Port{}

	for _, sbPort := range sbDevice.GetPorts() {

		port := sw.Port{
			Name:      sbPort.GetName(),
			Index:     int(sbPort.GetIndex()),
			Terminals: []sw.Terminal{},
		}

		for _, sbTerminal := range sbPort.GetTerminals() {
			t := sw.Terminal{
				Name:  sbTerminal.GetName(),
				Index: int(sbTerminal.GetIndex()),
				State: sbTerminal.GetState(),
			}
			port.Terminals = append(port.Terminals, t)
		}

		s.device.Ports = append(s.device.Ports, port)
	}

	if s.eventHandler != nil {
		go s.eventHandler(s, s.serialize())
	}

	return nil
}

func (s *SbSwitchProxy) getInfo() error {

	device, err := s.scli.GetDevice(context.Background(), &sbSwitch.None{})
	if err != nil {
		return err
	}

	s.device.Name = device.GetName()
	s.device.Index = int(device.GetIndex())
	ports := device.GetPorts()
	for _, port := range ports {
		p := sw.Port{
			Name:  port.GetName(),
			Index: int(port.GetIndex()),
		}

		terminals := port.GetTerminals()
		for _, terminal := range terminals {
			t := sw.Terminal{
				Name:  terminal.GetName(),
				Index: int(terminal.GetIndex()),
				State: terminal.GetState(),
			}
			p.Terminals = append(p.Terminals, t)
		}
		s.device.Ports = append(s.device.Ports, p)
	}

	return nil
}

func (s *SbSwitchProxy) Name() string {
	s.RLock()
	defer s.RUnlock()
	return s.device.Name
}

func (s *SbSwitchProxy) GetPort(portName string) (sw.Port, error) {
	s.RLock()
	defer s.RUnlock()

	for _, port := range s.device.Ports {
		if port.Name == portName {
			return port, nil
		}
	}
	return sw.Port{}, fmt.Errorf("unknown portname %s", portName)
}

func (s *SbSwitchProxy) SetPort(port sw.Port) error {
	s.Lock()
	defer s.Unlock()

	sbPortReq := &sbSwitch.PortRequest{
		Name:      port.Name,
		Terminals: []*sbSwitch.Terminal{},
	}

	for _, t := range port.Terminals {
		sbTerminal := &sbSwitch.Terminal{
			Name:  t.Name,
			State: t.State,
		}
		sbPortReq.Terminals = append(sbPortReq.Terminals, sbTerminal)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()
	_, err := s.scli.SetPort(ctx, sbPortReq)

	return err
}

func (s *SbSwitchProxy) Serialize() sw.Device {
	s.RLock()
	defer s.RUnlock()
	return s.serialize()
}

func (s *SbSwitchProxy) serialize() sw.Device {
	return s.device
}

func (s *SbSwitchProxy) Close() {
	if s.subscriber != nil {
		s.subscriber.Unsubscribe()
	}
	s.closeDone()
}
