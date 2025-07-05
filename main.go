package main

import (
	"flag"

	"github.com/miekg/dns"
)

func main() {
	addr := flag.String("addr", "127.0.0.1:53", "the address of the dns server")
	dns := flag.String("dns", "1.1.1.1", "the upstream dns server")
	flag.Parse()

	dns.HandleFunc(".", handleDNSRequest)

	server := &dns.Server {
		Addr: addr,
		Net: "udp",
	}

	if err := server.ListenAndServe(); err != nil {
		panic(err)
	}
}


func handleDNSRequest(w dns.ResponseWriter, r *dns.Msg) {
	
}