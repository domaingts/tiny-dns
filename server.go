package main

import (
	"net"
	"time"

	"github.com/miekg/dns"
)

type dnsServer struct {
	client      *dns.Client
	upstreamDNS string
	defaultIp   net.IP
	cache       *lru[string, *dns.Msg]
}

func newDnsServer(upstreamDNS, defaultIp string) *dnsServer {
	ip := net.ParseIP(defaultIp)
	if ip == nil {
		panic("Invalid IP address")
	}
	return &dnsServer{
		defaultIp:   ip.To4(),
		upstreamDNS: upstreamDNS,
		client: &dns.Client{
			Net:     "udp",
			Timeout: time.Second,
		},
		cache: newLRU[string, *dns.Msg](256),
	}
}

func (d *dnsServer) dnsHandleFunc(w dns.ResponseWriter, r *dns.Msg) {
	if len(r.Question) < 1 {
		dns.HandleFailed(w, r)
		return
	}

	if filtered, ok := d.cache.get(r.Question[0].Name); ok {
		filtered.SetReply(r)
		w.WriteMsg(filtered)
		return
	}

	msg := new(dns.Msg)
	msg.SetQuestion(r.Question[0].Name, dns.TypeAAAA)

	resp, _, err := d.client.Exchange(msg, d.upstreamDNS)
	if err != nil {
		dns.HandleFailed(w, r)
		return
	}

	filtered := new(dns.Msg)
	filtered.SetReply(r)
	filtered.Rcode = resp.Rcode
	filtered.RecursionAvailable = resp.RecursionAvailable

	for _, rr := range resp.Answer {
		if aaaa, ok := rr.(*dns.AAAA); ok {
			filtered.Answer = append(filtered.Answer, aaaa)
		}
	}

	if len(filtered.Answer) == 0 {
		filtered.Answer = append(filtered.Answer, &dns.A{
			Hdr: dns.RR_Header{
				Name:   r.Question[0].Name,
				Rrtype: dns.TypeA,
				Class:  dns.ClassINET,
				Ttl:    300,
			},
			A: d.defaultIp,
		})
		d.cache.put(r.Question[0].Name, filtered)
	}

	w.WriteMsg(filtered)
}
