# Source [![Go Report Card](https://goreportcard.com/badge/github.com/multiplay/go-source)](https://goreportcard.com/report/github.com/multiplay/go-source) [![License](https://img.shields.io/badge/license-BSD-blue.svg)](https://github.com/multiplay/go-source/blob/master/LICENSE) [![GoDoc](https://godoc.org/github.com/multiplay/go-source?status.svg)](https://godoc.org/github.com/multiplay/go-source) [![Build Status](https://travis-ci.org/multiplay/go-source.svg?branch=master)](https://travis-ci.org/multiplay/go-source)

go-source is a [Go](http://golang.org/) client for the [Source RCON Protocol](https://developer.valvesoftware.com/wiki/Source_RCON_Protocol).

Features
--------
* Full [Source RCON](https://developer.valvesoftware.com/wiki/Source_RCON_Protocol) Support.
* [Multi-Packet Responses](https://developer.valvesoftware.com/wiki/Source_RCON_Protocol#Multiple-packet_Responses) Support.

Supports
--------
* [Valve](http://www.valvesoftware.com/) [Counter-Strike Global Offensive](http://steamcommunity.com/app/730) and others.
* [Mojang](https://mojang.com/) [Minecraft](https://minecraft.net/).
* [Chucklefish](https://chucklefish.org/) [Starbound](https://playstarbound.com/).

Installation
------------
```sh
go get -u github.com/multiplay/go-source
```

Examples
--------

Using go-source is simple just create a client, login and then send commands e.g.
```go
package main

import (
	"log"

	"github.com/multiplay/go-source"
)

func main() {
	c, err := source.NewClient("192.168.1.102:27015", source.Password("mypass"))
	if err != nil {
		log.Fatal(err)
	}
	defer c.Close()

	if s, err := c.Exec("status"); err != nil {
		log.Fatal(err)
	} else {
		log.Println("server status:", s)
	}
}
```

Documentation
-------------
- [GoDoc API Reference](http://godoc.org/github.com/multiplay/go-source).

License
-------
go-source is available under the [BSD 2-Clause License](https://opensource.org/licenses/BSD-2-Clause).
