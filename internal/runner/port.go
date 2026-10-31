package runner

import (
	"net"
	"strconv"
	"time"
)

// PortFree reports whether nothing answers on the port on localhost (IPv4 or
// IPv6) and trama can listen on it.
func PortFree(port int) bool {
	p := strconv.Itoa(port)
	for _, host := range []string{"127.0.0.1", "::1"} {
		if c, err := net.DialTimeout("tcp", net.JoinHostPort(host, p), 300*time.Millisecond); err == nil {
			c.Close()
			return false
		}
	}
	l, err := net.Listen("tcp", ":"+p)
	if err != nil {
		return false
	}
	l.Close()
	return true
}
