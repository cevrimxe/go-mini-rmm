package collector

import (
	"fmt"
	"net"
	"os"
	"runtime"
)

type HostInfo struct {
	Hostname string
	OS       string
	IP       string
}

func Host() (HostInfo, error) {
	hostname, err := os.Hostname()
	if err != nil {
		hostname = "unknown"
	}

	ip := getOutboundIP()

	return HostInfo{
		Hostname: hostname,
		OS:       fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH),
		IP:       ip,
	}, nil
}

func getOutboundIP() string {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		return "127.0.0.1"
	}
	defer conn.Close()
	localAddr := conn.LocalAddr().(*net.UDPAddr)
	return localAddr.IP.String()
}
