package main

import (
	"net/netip"
	"slices"
	"time"

	"github.com/miekg/dns"
)

type dnsServer struct {
	client      *dns.Client
	upstreamDNS string
	defaultIp   netip.Addr
	// cache       *lru[string, *dns.Msg]
}

func newDnsServer(upstreamDNS, defaultIp string) *dnsServer {
	ip, err := netip.ParseAddr(defaultIp)
	if err != nil {
		panic("Invalid IP address")
	}
	return &dnsServer{
		defaultIp:   ip,
		upstreamDNS: upstreamDNS,
		client: &dns.Client{
			Net:     "udp",
			Timeout: time.Second,
		},
		// cache: newLRU[string, *dns.Msg](256),
	}
}

func (d *dnsServer) dnsHandleFunc(w dns.ResponseWriter, r *dns.Msg) {
	if len(r.Question) < 1 {
		dns.HandleFailed(w, r)
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

	answers := slices.Clone(resp.Answer)

	if !slices.ContainsFunc(answers, func(rr dns.RR) bool {
		_, ok := rr.(*dns.AAAA)
		return ok
	}) {
		answers = append(answers, d.defaultAnswer(r.Question[0].Name))
	}
	filtered.Answer = answers
	w.WriteMsg(filtered)
}

func (d *dnsServer) defaultAnswer(name string) dns.RR {
	var rr dns.RR
	if d.defaultIp.Is4() {
		rr = &dns.A{
			Hdr: dns.RR_Header{
				Name:   name,
				Rrtype: dns.TypeA,
				Class:  dns.ClassINET,
				Ttl:    300,
			},
			A: d.defaultIp.AsSlice(),
		}
	} else {
		rr = &dns.AAAA{
			Hdr: dns.RR_Header{
				Name:   name,
				Rrtype: dns.TypeAAAA,
				Class:  dns.ClassINET,
				Ttl:    300,
			},
			AAAA: d.defaultIp.AsSlice(),
		}
	}
	return rr
}
