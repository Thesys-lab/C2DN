package myutils

import (
	"errors"
	"io"
	"log"
	"net"
	"time"
)

func GetExternalIP() (string, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return "", err
	}
	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 {
			continue // interface down
		}
		if iface.Flags&net.FlagLoopback != 0 {
			continue // loopback interface
		}
		addrs, err := iface.Addrs()
		if err != nil {
			return "", err
		}
		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}
			if ip == nil || ip.IsLoopback() {
				continue
			}
			ip = ip.To4()
			if ip == nil {
				continue // not an ipv4 address
			}
			return ip.String(), nil
		}
	}
	return "", errors.New("are you connected to the network?")
}

func ConnIsClosed(c *net.TCPConn) (closed bool){
	_ = c.SetReadDeadline(time.Now())
	var one []byte
	if _, err := c.Read(one); err == io.EOF {
		log.Println("Client disconnect: %s", c.RemoteAddr())
		_ = c.Close()
		return true
	} else {
		var zero time.Time
		c.SetReadDeadline(zero)
	}
	return false
}