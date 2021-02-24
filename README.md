# RemoteSwitch

[![Go Report Card](https://goreportcard.com/badge/github.com/dh1tw/remoteSwitch)](https://goreportcard.com/report/github.com/dh1tw/remoteSwitch)
[![Build Status](https://travis-ci.com/dh1tw/remoteSwitch.svg?branch=master)](https://github.com/dh1tw/remoteSwitch/workflows/Cross%20Platform%20build/badge.svg?branch=master)
[![MIT licensed](https://img.shields.io/badge/license-MIT-blue.svg)](https://img.shields.io/badge/license-MIT-blue.svg)
[![Coverage Status](https://coveralls.io/repos/github/dh1tw/remoteSwitch/badge.svg?branch=travis-setup)](https://coveralls.io/github/dh1tw/remoteSwitch?branch=travis-setup)
[![Downloads](https://img.shields.io/github/downloads/dh1tw/remoteSwitch/total.svg?maxAge=1800)](https://github.com/dh1tw/remoteSwitch/releases)

[![Alt text](https://i.imgur.com/YptJPYH.png "remoteSwitch WebUI")](https://demo.switch.shackbus.org)

remoteSwitch is a cross platform application which makes your (antenna/band/power...) switches available on the network.

To get a first impression, you're welcome to play with our public demo at [demo.switch.shackbus.org](https://demo.switch.shackbus.org).

remoteSwitch is written in the programing language [Go](https://golang.org).

**ADVICE**: This project is **under development**. The parameters and the ICD are still **not stable** and subject to change until the first major version has been reached.

## Supported switch types

- Multi Purpose Switch GPIO (e.g Bandswitch, Beverages, etc.)
- Stackmatch GPIO (Stackmatch, Combiners, 4-Squares. etc)
- Dummy Switch (for testing purposes without hardware)
- EA4TX Remotebox

## Supported Transportation Protocols

- [NATS](https://nats.io)
- HTTP & Websockets for the WebUI

## License

remoteSwitch is published under the permissive [MIT license](https://github.com/dh1tw/remoteSwitch/blob/master/LICENSE).

## Download

You can download a zip archive with the compiled binary for MacOS (AMD64), Linux (386/AMD64/ARM/ARM64) and Windows (386/AMD64) from the
[releases](https://github.com/dh1tw/remoteSwitch/releases) page. remoteSwitch works particularly well on SoC boards like the Raspberry / Orange / Banana Pis. The application itself is just a single executable.

## Dependencies

remoteSwitch only depends on a few go libraries which are needed at compile
time. There are no runtime dependencies.

## Getting started

remoteSwitch provides a series of nested commands and flags.

````bash
$ ./remoteSwitch
````

````
Network interface for Switches

Usage:
  remoteSwitch [command]

Available Commands:
  help        Help about any command
  server      Switch Server
  version     Print the version number of remoteSwitch
  web         webserver providing access to all switches on the network

Flags:
      --config string   config file (default is $HOME/.remoteSwitch.yaml)
  -h, --help            help for remoteSwitch

Use "remoteSwitch [command] --help" for more information about a command.
````

So let's fire up a remoteSwitch server for a dummy 6x2 antenna/bandswitch.

To get a list of supported flags for the nats server, execute:

``` bash
$ ./remoteSwitch server nats --help


The nats server allows you to expose an Switch on a nats.io broker. The broker
can be located within your local lan or somewhere on the internet.

Usage:
  remoteSwitch server nats [flags]

Flags:
  -p, --broker-port int     Broker Port (default 4222)
  -u, --broker-url string   Broker URL (default "localhost")
  -h, --help                help for nats
  -P, --password string     NATS Password
  -U, --username string     NATS Username

Global Flags:
      --config string   config file (default is $HOME/.remoteSwitch.yaml)
```

### Configuration

While some of the parameters can be set via pflags, the switch configuration in particular **MUST** be set in a **config file** due to its complexity.

As a starting point you might want to download a copy of the example configuration file
[./remoteSwitch.toml](https://github.com/dh1tw/remoteSwitch/blob/master/.remoteSwitch.toml)
which comes conveniently pre-configured for our dummy antenna/bandswitch. The [example folder](https://github.com/dh1tw/remoteSwitch/tree/master/examples) in this repository contains additional example configurations.

### NATS Broker

[NATS](https://nats.io) is an open source, lightweight, high performance message broker which is needed for the underlying communication. You can decide where to run your NATS instance. You can run it on your local machine, in a VPN or expose it to the internet. You can download the NATS broker [here](https://nats.io/download/nats-io/nats-server).

run the NATS broker:

``` bash
$ ./nats-server

[62418] 2020/04/11 02:46:09.413858 [INF] Starting nats-server version 2.1.6
[62418] 2020/04/11 02:46:09.413959 [INF] Git commit [not set]
[62418] 2020/04/11 02:46:09.414150 [INF] Listening for client connections on 0.0.0.0:4222
[62418] 2020/04/11 02:46:09.414156 [INF] Server id is NDCPYYXYRSKD6PIBTS7YHZGEBWIN3CBPRH232CMHUWU3NXBQTTBBQRNF
[62418] 2020/04/11 02:46:09.414158 [INF] Server is ready
```

### Connecting to the NATS broker

Let's execute:

```bash
$ ./remoteSwitch server nats --config=.remoteSwitch.toml

Using config file:
/home/dh1tw/.remoteSwitch.toml
2019/01/11 23:50:20 Listening on shackbus.switch.6x2_Bandswitch
2019/01/11 23:50:20 Registering node: shackbus.switch.6x2 Bandswitch-45988210-15f3-11e9-b0fa-6c4008b0322c
```

## Web Interface

remoteSwitch comes with a built-in web server which allows to control all switches connected to the same NATS broker. All instances of remoteSwitch are automatically discovered. You can run several instances of the web server. This might be handy if you have to deal with lan/wan restrictions or if you need redundancy.

Simply launch:

```
$ ./remoteSwitch web
```

## Config file

The repository contains a ready-to-go example configuration file.  By convention it is called `.remoteSwitch.[yaml|toml|json]` and is located by default either in the home directory or the directory where the remoteSwitch executable is located.
The format of the file can either be in
[yaml](https://en.wikipedia.org/wiki/YAML),
[toml](https://github.com/toml-lang/toml), or
[json](https://en.wikipedia.org/wiki/JSON).

The first line after starting remoteSwitch will indicate if / which config
file has been found.

More complex example configuration files are located in the folder [examples](https://github.com/dh1tw/remoteSwitch/tree/master/examples).

If you want to run several remoteSwitches on the same machine, you have to create a configuration file for each of them and specify them with the --config flag.

Priority:

1. Pflags (e.g. -p 4040)
2. Values from config file
3. Default values

The GPIO switches are pretty flexible in configuration and should be able to cope with most user requirements.

## Behaviour on Errors

If an error occurs from which remoteSwitch can not recover, the application exits. It is recommended to execute remoteSwitch as a service under the supervision of a scheduler like [systemd](https://en.wikipedia.org/wiki/Systemd) on Linux or [NSSM - the Non-Sucking Service Manager](https://nssm.cc/download) on Windows.

## Bug reports, Questions & Pull Requests

Please use the Github [Issue tracker](https://github.com/dh1tw/remoteSwitch/issues) to report bugs and ask questions! If you would like to contribute to remoteSwitch, [pull requests](https://help.github.com/articles/creating-a-pull-request/) are welcome! However please consider to provide unit tests with the PR to verify the proper behavior.

If you file a bug report, please include always the version of remoteSwitch
you are running:

```` bash
$ remoteSwitch.exe version

copyright Tobias Wellnitz, DH1TW, 2020
remoteSwitch Version: v0.1.0, darwin/amd64, BuildDate: 2020-04-11T02:50:26+02:00, Commit: 528027d
````

## Documentation

The auto generated documentation can be found at [godoc.org](https://godoc.org/github.com/dh1tw/remoteSwitch).

## How to build

In order to compile remoteSwitch from the sources, you need to have [Go](https://golang.org) installed and configured.

This his how to checkout and compile remoteSwitch under Linux/MacOS:

```bash
$ go get -d github.com/dh1tw/remoteSwitch
$ cd $GOPATH/src/github.com/remoteSwitch
$ make
```

## How to execute the tests

All critical packages have their own set of unit tests. The tests can be executed with the following commands:

```bash
$ cd $GOPATH/src/github.com/remoteSwitch
$ go test -v -race ./...

```

The race detector might not be available on all platforms / operating systems.
