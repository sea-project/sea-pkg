package ip

import (
	"net"
	"strings"
)

//ParseIPAddr return ip address
func ParseIPAddr(addr net.Addr) string {
	s := addr.String()
	i := strings.Index(s, ":")
	if i < 0 {
		return ""
	}
	return s[:i]
}
