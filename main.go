package main

import (
	"flag"

	"github.com/miekg/dns"
)

func main() {
	addr := flag.String("addr", ":15553", "the address of the dns server")
	upstreamDNS := flag.String("dns", "1.1.1.1:53", "the upstream dns server")
	defaultIP := flag.String("default", "10.1.11.111", "the default ip when no ip returned")
	flag.Parse()

	s := newDnsServer(*upstreamDNS, *defaultIP)

	dns.HandleFunc(".", s.dnsHandleFunc)

	server := &dns.Server{
		Addr: *addr,
		Net:  "udp",
	}

	if err := server.ListenAndServe(); err != nil {
		panic(err)
	}
}
