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
		defaultIp:   ip.Unmap(),
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

	q := r.Question[0]

	resp, err := d.exchange(r)
	if err != nil {
		dns.HandleFailed(w, r)
		return
	}

	if q.Qtype != dns.TypeAAAA ||
		q.Qclass != dns.ClassINET ||
		resp.Rcode != dns.RcodeSuccess ||
		resp.Truncated {
		w.WriteMsg(resp)
		return
	}

	hasIPv6 := slices.ContainsFunc(resp.Answer, func(rr dns.RR) bool {
		_, ok := rr.(*dns.AAAA)
		return ok
	})

	if hasIPv6 || !d.defaultIp.Is6() {
		w.WriteMsg(resp)
		return
	}

	query := r.Copy()
	query.Question[0].Qtype = dns.TypeA
	v4resp, err := d.exchange(query)
	if err != nil {
		dns.HandleFailed(w, r)
		return
	}

	hasIPv4 := slices.ContainsFunc(v4resp.Answer, func(rr dns.RR) bool {
		_, ok := rr.(*dns.A)
		return ok
	})

	if v4resp.Rcode != dns.RcodeSuccess || v4resp.Truncated || !hasIPv4 {
		w.WriteMsg(resp)
		return
	}

	msg := new(dns.Msg)
	msg.SetReply(r)
	msg.RecursionAvailable = resp.RecursionAvailable
	msg.Answer = []dns.RR{
		d.defaultAnswer(q.Name),
	}
	w.WriteMsg(msg)
}

func (d *dnsServer) exchange(msg *dns.Msg) (*dns.Msg, error) {
	resp, _, err := d.client.Exchange(msg, d.upstreamDNS)
	if err != nil {
		return nil, err
	}

	if resp.Truncated {
		client := *d.client
		client.Net = "tcp"

		resp, _, err = client.Exchange(msg, d.upstreamDNS)
	}

	return resp, err
}

func (d *dnsServer) defaultAnswer(name string) dns.RR {
	var rr dns.RR
	if d.defaultIp.Is4() {
		rr = &dns.A{
			Hdr: dns.RR_Header{
				Name:   name,
				Rrtype: dns.TypeA,
				Class:  dns.ClassINET,
				Ttl:    60,
			},
			A: d.defaultIp.AsSlice(),
		}
	} else {
		rr = &dns.AAAA{
			Hdr: dns.RR_Header{
				Name:   name,
				Rrtype: dns.TypeAAAA,
				Class:  dns.ClassINET,
				Ttl:    60,
			},
			AAAA: d.defaultIp.AsSlice(),
		}
	}
	return rr
}
