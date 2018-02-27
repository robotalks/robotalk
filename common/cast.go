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

	remote *net.UDPAddr
}

// Dial creates the UDP socket
func (c *UDPCast) Dial() (err error) {
	var laddr *net.UDPAddr
	if c.BindAddr != "" {
		laddr, err = net.ResolveUDPAddr("udp", c.BindAddr)
		if err != nil {
			return fmt.Errorf("bind address: %v", err)
		}
	}
	c.remote, err = net.ResolveUDPAddr("udp", c.Address)
	if err != nil {
		return fmt.Errorf("cast address: %v", err)
	}
	c.Conn, err = net.DialUDP("udp", laddr, c.remote)
	return err
}

// Close implements io.Closer
func (c *UDPCast) Close() error {
	return c.Conn.Close()
}

// Cast implements CastTarget
func (c *UDPCast) Cast(data []byte) mqhub.Future {
	f := &mqhub.ImmediateFuture{}
	if c.remote.IP != nil {
		_, f.Error = c.Conn.WriteToUDP(data, c.remote)
		if f.Error != nil {
			println(f.Error.Error(), len(data))
		}
	}
	return f
}

// HasTarget determines if remote address has been set
func (c *UDPCast) HasTarget() bool {
	return c.remote.IP != nil
}

// SetRemoteAddr sets remote address
func (c *UDPCast) SetRemoteAddr(addr string) error {
	raddr, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		return err
	}
	c.remote = raddr
	return nil
}
