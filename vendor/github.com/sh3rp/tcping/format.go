package tcping

import (
	"fmt"
	"strings"
	"time"

	"github.com/aybabtme/rgbterm"
)

type Color struct {
	R uint8
	G uint8
	B uint8
}

func FormatResult(result ProbeResult, useColor bool) string {
	tx := result.TxPacket.Header
	rx := result.RxPacket.Header
	var str string
	txBumperSpace := (27 - len(result.TxPacket.IP)) / 2
	rxBumperSpace := (27 - len(result.RxPacket.IP)) / 2
	msLabel := fmt.Sprintf("%d ms", result.Latency()/int64(time.Millisecond))

	str = str + strings.Repeat(" ", ((27+27)+len(msLabel))/2) + msLabel + "\n"
	fmtStr := "%s" +
		strings.Repeat(" ", txBumperSpace) +
		"%s" +
		strings.Repeat(" ", txBumperSpace) +
		"%s     %s" +
		strings.Repeat(" ", rxBumperSpace) +
		"%s" +
		strings.Repeat(" ", rxBumperSpace) +
		"%s\n"
	str = str + fmt.Sprintf(fmtStr,
		Open(),
		rgbterm.FgString(result.TxPacket.IP, 231, 127, 255),
		Close(),
		Open(),
		rgbterm.FgString(result.RxPacket.IP, 231, 127, 255),
		Close())
	str = str + fmt.Sprintf("%s %s %s %s %s %s %s %s     %s %s %s %s %s %s %s %s\n",
		Open(),
		Field("SRC"),
		Value(int(tx.Src), 5),
		Close(),
		Open(),
		Field("DST"),
		Value(int(tx.Dst), 5),
		Close(),
		Open(),
		Field("SRC"),
		Value(int(rx.Src), 5),
		Close(),
		Open(),
		Field("DST"),
		Value(int(rx.Dst), 5),
		Close())
	str = str + fmt.Sprintf("%s %s %s %s     %s %s %s %s\n",
		Open(),
		Field("SEQ"),
		Value(int(tx.Seq), 20),
		Close(),
		Open(),
		Field("SEQ"),
		Value(int(rx.Seq), 20),
		Close())
	str = str + fmt.Sprintf("%s %s %s %s     %s %s %s %s\n",
		Open(),
		Field("ACK"),
		Value(int(tx.Ack), 20),
		Close(),
		Open(),
		Field("ACK"),
		Value(int(rx.Ack), 20),
		Close())
	str = str + fmt.Sprintf("%s %s %s%s%s%s%s%s%s %s %s %s %s     %s %s %s%s%s%s%s%s%s %s %s %s %s\n",
		Open(),
		Field("FLG"),
		FlagEntry(tx, URG, useColor),
		FlagEntry(tx, ACK, useColor),
		FlagEntry(tx, PSH, useColor),
		FlagEntry(tx, RST, useColor),
		FlagEntry(tx, SYN, useColor),
		FlagEntry(tx, FIN, useColor),
		Close(),
		Open(),
		Field("WIN"),
		Value(int(tx.Window), 5),
		Close(),
		Open(),
		Field("FLG"),
		FlagEntry(rx, URG, useColor),
		FlagEntry(rx, ACK, useColor),
		FlagEntry(rx, PSH, useColor),
		FlagEntry(rx, RST, useColor),
		FlagEntry(rx, SYN, useColor),
		FlagEntry(rx, FIN, useColor),
		Close(),
		Open(),
		Field("WIN"),
		Value(int(rx.Window), 5),
		Close(),
	)
	str = str + fmt.Sprintf("%s %s %s %s %s %s %s %s     %s %s %s %s %s %s %s %s\n",
		Open(),
		Field("SUM"),
		Value(int(tx.Checksum), 5),
		Close(),
		Open(),
		Field("URG"),
		Value(int(tx.Urgent), 5),
		Close(),
		Open(),
		Field("SUM"),
		Value(int(rx.Checksum), 5),
		Close(),
		Open(),
		Field("URG"),
		Value(int(rx.Urgent), 5),
		Close())
	//for _, o := range tx.Options {
	//	str = str + fmt.Sprintf("[ Option: kind=%d len=%d data=%v ]\n", o.Kind, o.Length, o.Data)
	//}
	return str
}

func Open() string {
	return rgbterm.FgString("[", 0, 102, 255)
}

func Close() string {
	return rgbterm.FgString("]", 0, 102, 255)
}

func Field(name string) string {
	return rgbterm.FgString(fmt.Sprintf("%s:", name), 102, 153, 255)
}

func Value(val int, size int) string {
	return rgbterm.FgString(fmt.Sprintf(fmt.Sprintf("%%%dd", size), val), 66, 241, 244)
}

func FlagEntry(header TCPHeader, flag byte, color bool) string {
	if header.HasFlag(flag) {
		switch flag {
		case URG:
			return rgbterm.FgString("U", 200, 66, 244)
		case ACK:
			return rgbterm.FgString("A", 66, 244, 113)
		case PSH:
			return rgbterm.FgString("P", 76, 183, 255)
		case RST:
			return rgbterm.FgString("R", 244, 89, 66)
		case SYN:
			return rgbterm.FgString("S", 255, 180, 76)
		case FIN:
			return rgbterm.FgString("F", 255, 177, 15)
		default:
			return " "
		}
	} else {
		return rgbterm.FgString("_", 155, 155, 155)
	}
}
