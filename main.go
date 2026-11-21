package main

import (
	"flag"
	"log"
	"net"

	"github.com/miekg/dns"
)

func main() {
	addr := flag.String("addr", ":53", "the address of the dns server")
	upstreamDNS := flag.String("dns", "1.1.1.1:53", "the upstream dns server")
	tcpPort := flag.Int64("tcp", 443, "the tcp server's port")
	defaultIP := flag.String("default", "10.1.11.111", "the default ip when no ip returned")
	configPath := flag.String("config", "/usr/local/tiny-dns/config.json", "the config file including a white list")
	flag.Parse()

	provider, err := NewFileProvider(*configPath)
	if err != nil {
		panic(err)
	}

	s := newDnsServer(*upstreamDNS, *defaultIP, provider)

	dns.HandleFunc(".", s.dnsHandleFunc)

	server := &dns.Server{
		Addr: *addr,
		Net:  "udp",
	}

	go func() {
		if err := server.ListenAndServe(); err != nil {
			panic(err)
		}
	}()

	listener, err := net.ListenTCP("tcp", &net.TCPAddr{Port: int(*tcpPort)})
	if err != nil {
		log.Fatal(err)
	}
	defer listener.Close()

	log.Println("SNI proxy listening on :443")

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Println(err)
			continue
		}

		tcpConn, ok := conn.(*net.TCPConn)
		if !ok {
			log.Println("invalid type cast")
			continue
		}

		go handleConn(tcpConn)
	}
}
