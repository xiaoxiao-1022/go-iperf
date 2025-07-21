package main

import (
	"github.com/xiaoxiao-1022/go-iperf"
	"os"

	//"github.com/BGrewell/go-iperf"
	//"time"
	"fmt"
)

func main() {
	proto := "udp"
	//ping.online.net
	c := iperf.NewClient("18.143.196.2")
	c.BinaryPath = "D:\\Programs\\iperf3\\iperf3.exe"
	c.SetJSON(true)
	c.SetStreams(4)
	c.SetTimeSec(5)
	//c.SetInterval(1)
	c.SetPort(40012)
	//c.SetReverse(true)
	c.SetConnectTimeout(5000)
	c.SetProto((iperf.Protocol)(proto))
	c.SetLength("1180")
	err := c.Start()
	if err != nil {
		fmt.Printf("failed to start client: %v\n", err)
		os.Exit(-1)
	}

	<-c.Done
	fmt.Println("report:", c.Report().String())
}
