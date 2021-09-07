package tcping

import (
	"net"
)

type ProbeWatcher interface {
	WatchFor(string, uint16, func(*TCPHeader))
}

type probeWatcher struct {
	localAddress string
	ipPort       IpPort
	callback     func(*TCPHeader)
}

type IpPort struct {
	ip   string
	port uint16
}

type Packet struct {
	ipPort IpPort
	header *TCPHeader
}

func NewProbeWatcher(localAddress string) ProbeWatcher {
	pw := &probeWatcher{
		localAddress: localAddress,
	}
	go pw.watch()
	return pw
}

func (pw *probeWatcher) WatchFor(srcIp string, srcPort uint16, f func(tcp *TCPHeader)) {
	pw.ipPort = IpPort{srcIp, srcPort}
	pw.callback = f
}

func (pw *probeWatcher) watch() {
	netaddr, err := net.ResolveIPAddr("ip4", pw.localAddress)
	if err != nil {
		return
	}

	conn, err := net.ListenIP("ip4:tcp", netaddr)
	if err != nil {
		return
	}

	var tcp *TCPHeader
	for {
		buf := make([]byte, 1024)
		numRead, raddr, err := conn.ReadFrom(buf)
		if err != nil {
			return
		}

		tcp = ParseTCP(buf[:numRead])

		if raddr.String() == pw.ipPort.ip && tcp.Src == pw.ipPort.port {
			pw.callback(tcp)
		}
	}
}
