package tcping

import (
	"bytes"
	"encoding/binary"
)

const (
	FIN = 1
	SYN = 2
	RST = 4
	PSH = 8
	ACK = 16
	URG = 32
)

type TCPHeader struct {
	Src        uint16
	Dst        uint16
	Seq        uint32
	Ack        uint32
	DataOffset uint8
	Reserved   uint8
	ECN        uint8
	Ctrl       uint8
	Window     uint16
	Checksum   uint16
	Urgent     uint16
	Options    []TCPOption
}

type TCPOption struct {
	Kind   uint8
	Length uint8
	Data   []byte
}

func NewTCPHeader() *TCPHeader {
	return &TCPHeader{}
}

func (tcp *TCPHeader) SrcPort(port uint16) *TCPHeader {
	tcp.Src = port
	return tcp
}

func (tcp *TCPHeader) DstPort(port uint16) *TCPHeader {
	tcp.Dst = port
	return tcp
}

func (tcp *TCPHeader) SeqNum(num uint32) *TCPHeader {
	tcp.Seq = num
	return tcp
}

func (tcp *TCPHeader) Win(window uint16) *TCPHeader {
	tcp.Window = window
	return tcp
}

func (tcp *TCPHeader) WithOption(option TCPOption) *TCPHeader {
	tcp.Options = append(tcp.Options, option)
	return tcp
}

func (tcp *TCPHeader) WithFlag(flagBit byte) *TCPHeader {
	tcp.Ctrl = tcp.Ctrl | flagBit
	return tcp
}

func ParseTCP(data []byte) *TCPHeader {
	var tcp TCPHeader
	r := bytes.NewReader(data)
	binary.Read(r, binary.BigEndian, &tcp.Src)
	binary.Read(r, binary.BigEndian, &tcp.Dst)
	binary.Read(r, binary.BigEndian, &tcp.Seq)
	binary.Read(r, binary.BigEndian, &tcp.Ack)

	var mix uint16
	binary.Read(r, binary.BigEndian, &mix)
	tcp.DataOffset = byte(mix >> 12)
	tcp.Reserved = byte(mix >> 9 & 7)
	tcp.ECN = byte(mix >> 6 & 7)
	tcp.Ctrl = byte(mix & 0x3f)

	binary.Read(r, binary.BigEndian, &tcp.Window)
	binary.Read(r, binary.BigEndian, &tcp.Checksum)
	binary.Read(r, binary.BigEndian, &tcp.Urgent)

	return &tcp
}

func (tcp *TCPHeader) HasFlag(flagBit byte) bool {
	return tcp.Ctrl&flagBit != 0
}

func (tcp *TCPHeader) MarshalTCP() []byte {
	buf := new(bytes.Buffer)
	binary.Write(buf, binary.BigEndian, tcp.Src)
	binary.Write(buf, binary.BigEndian, tcp.Dst)
	binary.Write(buf, binary.BigEndian, tcp.Seq)
	binary.Write(buf, binary.BigEndian, tcp.Ack)

	offset := 5
	for _, o := range tcp.Options {
		offset += 2 + len(o.Data)
	}
	tcp.DataOffset = uint8(offset)

	var mix uint16
	mix = uint16(tcp.DataOffset)<<12 |
		uint16(tcp.Reserved)<<9 |
		uint16(tcp.ECN)<<6 |
		uint16(tcp.Ctrl)
	binary.Write(buf, binary.BigEndian, mix)
	binary.Write(buf, binary.BigEndian, tcp.Window)
	binary.Write(buf, binary.BigEndian, tcp.Checksum)
	binary.Write(buf, binary.BigEndian, tcp.Urgent)

	for _, option := range tcp.Options {
		binary.Write(buf, binary.BigEndian, option.Kind)
		if option.Length > 1 {
			binary.Write(buf, binary.BigEndian, option.Length)
			binary.Write(buf, binary.BigEndian, option.Data)
		}
	}

	out := buf.Bytes()

	padding := 20 - len(out)
	for i := 0; i < padding; i++ {
		out = append(out, 0)
	}

	return out
}

func Checksum(data []byte, srcip [4]byte, dstip [4]byte) uint16 {
	hdr := []byte{
		srcip[0], srcip[1], srcip[2], srcip[3],
		dstip[0], dstip[1], dstip[2], dstip[3],
		0,
		6,
		0, byte(len(data)),
	}

	target := make([]byte, 0, len(hdr)+len(data))
	target = append(target, hdr...)
	target = append(target, data...)

	lenSumThis := len(target)
	var nextWord uint16
	var sum uint32
	for i := 0; i+1 < lenSumThis; i += 2 {
		nextWord = uint16(target[i])<<8 | uint16(target[i+1])
		sum += uint32(nextWord)
	}
	if lenSumThis%2 != 0 {
		sum += uint32(target[len(target)-1])
	}

	sum = (sum >> 16) + (sum & 0xffff)
	sum = sum + (sum >> 16)

	return uint16(^sum)
}
