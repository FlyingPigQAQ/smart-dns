package dynforward

import (
	"context"
	"fmt"
	"github.com/coredns/coredns/plugin"
	"github.com/go-ping/ping"
	"github.com/miekg/dns"
	"time"
)

type DynForward struct {
	Next        plugin.Handler
	InternalDNS string
	ExternalDNS string
}

type DynResponse struct {
	Response *dns.Msg
	Error    error
	Source   string
}

func (p DynForward) ServeDNS(ctx context.Context, w dns.ResponseWriter, r *dns.Msg) (int, error) {
	// Create a new DNS client
	client := &dns.Client{}
	channel := make(chan DynResponse, 2)
	go func() {
		internalIn, _, err := client.Exchange(r, p.InternalDNS)

		channel <- DynResponse{internalIn, err, "internal"}
	}()

	go func() {
		externalIN, _, err := client.Exchange(r, p.ExternalDNS)
		channel <- DynResponse{externalIN, err, "external"}
	}()
	// Wait for both responses
	var internalResponse, externalResponse *dns.Msg
	var internalError, externalError error
	for i := 0; i < 2; i++ {
		select {
		case res := <-channel:
			if res.Source == "internal" {
				internalResponse = res.Response
				internalError = res.Error
			} else {
				externalResponse = res.Response
				externalError = res.Error
			}
		}
	}
	// Check for errors
	if internalError != nil && externalError != nil {
		return dns.RcodeServerFailure, internalError
	}

	if internalError == nil && internalResponse.Rcode == 0 {
		if internalResponse.Question[0].Qtype == dns.TypeA {
			ipv4 := getIpv4(internalResponse)
			if ipv4 != "" && pingHost(ipv4) {
				return success(internalResponse, &w)
			}
		} else {
			return success(internalResponse, &w)
		}
	}

	if externalError == nil && externalResponse.Rcode == 0 {
		return success(externalResponse, &w)
	}

	return dns.RcodeServerFailure, nil
}

func (p DynForward) Name() string {
	return "dynforward"
}

func pingHost(host string) bool {
	pinger, err := ping.NewPinger(host)
	if err != nil {
		fmt.Println("Error creating pinger:", err)
		return false
	}

	pinger.Count = 1
	pinger.Timeout = 20 * time.Millisecond
	pinger.SetPrivileged(false) // ← 关键：非 root 用户设置

	err = pinger.Run() // Blocks until finished.
	if err != nil {
		fmt.Println("Ping failed:", err)
		return false
	}

	stats := pinger.Statistics()
	return stats.PacketsRecv > 0
}

func getIpv4(response *dns.Msg) string {
	for _, ans := range response.Answer {
		switch rr := ans.(type) {
		case *dns.A:
			return rr.A.String()
		}
	}
	return ""
}

func success(response *dns.Msg, w *dns.ResponseWriter) (int, error) {
	err := (*w).WriteMsg(response)
	if err != nil {
		return dns.RcodeServerFailure, err
	}
	return dns.RcodeSuccess, nil
}
