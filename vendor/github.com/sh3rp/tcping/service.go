package tcping

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"
)

type WebService struct {
	probe Probe
}

func NewWebService(probe Probe) WebService {
	return WebService{probe}
}

func (ws WebService) Start(port int) error {
	server := http.NewServeMux()
	server.HandleFunc("/probe", ws.sendProbe)
	return http.ListenAndServe(fmt.Sprintf(":%d", port), server)
}

func (ws WebService) sendProbe(res http.ResponseWriter, req *http.Request) {
	host := req.URL.Query().Get("ip")
	if host == "" {
		res.Write(response(1, "host: Must supply a host IP", Result{}))
		return
	}
	portStr := req.URL.Query().Get("port")
	port := 80
	var err error
	if portStr != "" {
		port, err = strconv.Atoi(portStr)
		if err != nil {
			res.Write(response(2, fmt.Sprintf("port: %s", err.Error()), Result{}))
			return
		}
	}
	countStr := req.URL.Query().Get("count")
	count := 1
	if countStr != "" {
		count, err = strconv.Atoi(countStr)
		if err != nil {
			res.Write(response(3, fmt.Sprintf("count: %s", err.Error()), Result{}))
			return
		}
	}

	var probes []Mark

	for i := 0; i < count; i++ {
		latency, err := ws.probe.GetLatency(host, uint16(port))
		probes = append(probes, Mark{Timestamp: time.Now(), Latency: float64(latency) / float64(time.Millisecond), Error: err})
	}

	res.Write(okResponse(Result{IP: host, Port: port, Marks: probes}))
}

func okResponse(probes Result) []byte {
	return response(0, "ok", probes)
}

func response(code int, message string, probes Result) []byte {
	res := Response{code, message, probes}
	bytes, _ := json.Marshal(res)
	return bytes
}

type Response struct {
	Code    int    `json:"code"`
	Message string `json:"msg"`
	Probe   Result `json:"probe"`
}

type Result struct {
	IP    string `json:"ip"`
	Port  int    `json:"port"`
	Marks []Mark `json:"marks"`
}

type Mark struct {
	Timestamp time.Time `json:"timestamp"`
	Latency   float64   `json:"delta"`
	Error     error     `json:"error"`
}
