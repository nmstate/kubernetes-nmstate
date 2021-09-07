# TCPing

A tool that utilizes the TCP handshake to measure latency between two hosts.  Since
ICMP bears both the virtues of being blocked at the firewall and bearing a lower 
priority in most routing engines, using standard ping techniques often times renders
a less than desireable result.

This tool sends a TCP SYN packet (just the header) to a specified destination IP
and port.  The resulting SYN/ACK if the port of the destination host is open/listening
or RST if the port and the time it takes to receive that packet is measured to yield
a reasonably accurate latency measurement.

# Install

Make sure Go is installed.  You can get Go for your OS of choice easily at 
https://golang.org.

Then:

```
go get github.com/sh3rp/tcping
```

# Usage

The simplest way to execute is:

```
sudo tcping -h <dst ip> -p <dst port>
```

You can add the '-d' flag for debugging the packets sent and received.

To add the capability to tcping (and not have to run as root):

```
setcap cap_net_raw+ep <tcping location>