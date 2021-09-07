package tcping

type ProbePacket struct {
	IP     string
	Header TCPHeader
	Mark   int64
}

type ProbeResult struct {
	TxPacket ProbePacket
	RxPacket ProbePacket
	IsAlive  bool
}

func (result ProbeResult) Latency() int64 {
	return result.RxPacket.Mark - result.TxPacket.Mark
}
