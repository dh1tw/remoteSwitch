module github.com/dh1tw/remoteSwitch

go 1.16

require (
	github.com/GeertJohan/go.rice v1.0.2
	github.com/asim/go-micro/plugins/broker/nats/v3 v3.0.0-20210408173139-0d57213d3f5c
	github.com/asim/go-micro/plugins/registry/nats/v3 v3.0.0-20210408173139-0d57213d3f5c
	github.com/asim/go-micro/plugins/transport/nats/v3 v3.0.0-20210408173139-0d57213d3f5c
	github.com/asim/go-micro/v3 v3.5.0
	github.com/golang/protobuf v1.5.2
	github.com/gorilla/mux v1.8.0
	github.com/gorilla/websocket v1.4.2
	github.com/nats-io/nats.go v1.10.0
	github.com/spf13/cobra v1.1.3
	github.com/spf13/viper v1.7.1
	github.com/tarm/serial v0.0.0-20180830185346-98f6abe2eb07
	google.golang.org/protobuf v1.26.0
	periph.io/x/periph v3.6.7+incompatible
)
