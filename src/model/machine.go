package model

import "net"

type Machine struct {
	Id     string
	Ip     net.Addr
	Client Client
}
