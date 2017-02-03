package common

import (
	"fmt"
	"net"

	"github.com/robotalks/mqhub.go/mqhub"
)

// CastTarget defines a target to cast raw data to
type CastTarget interface {
	Cast([]byte) mqhub.Future
}

// DataPointCast casts to mqhub datapoint
type DataPointCast struct {
	DP *mqhub.DataPoint
}

// Cast implement CastTarget
func (c *DataPointCast) Cast(data []byte) mqhub.Future {
	return c.DP.Update(mqhub.StreamMessage(data))
}

// UDPCast casts via UDP broadcast/multicast
type UDPCast struct {
	BindAddr string
	Address  string
	Conn     *net.UDPConn
}

// Dial creates the UDP socket
func (c *UDPCast) Dial() (err error) {
	var laddr, raddr *net.UDPAddr
	if c.BindAddr != "" {
		laddr, err = net.ResolveUDPAddr("udp", c.BindAddr)
		if err != nil {
			return fmt.Errorf("bind address: %v", err)
		}
	}
	raddr, err = net.ResolveUDPAddr("udp", c.Address)
	if err != nil {
		return fmt.Errorf("cast address: %v", err)
	}
	if raddr.IP == nil {
		raddr.IP = net.IPv4bcast
	}
	c.Conn, err = net.DialUDP("udp", laddr, raddr)
	return err
}

// Close implements io.Closer
func (c *UDPCast) Close() error {
	return c.Conn.Close()
}

// Cast implements CastTarget
func (c *UDPCast) Cast(data []byte) mqhub.Future {
	_, err := c.Conn.Write(data)
	if err != nil {
		println(err.Error(), len(data))
	}
	return &mqhub.ImmediateFuture{Error: err}
}
