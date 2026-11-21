package main

import (
	"errors"
	"flag"
	"log"
	"net"
	"net/netip"
	"time"

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

	defer provider.Close()

	s := newDnsServer(*upstreamDNS, *defaultIP, provider)

	// dns.HandleFunc(".", s.dnsHandleFunc)

	server := &dns.Server{
		Addr:    *addr,
		Net:     "udp",
		Handler: dns.HandlerFunc(s.dnsHandleFunc),
		NotifyStartedFunc: func() {
			log.Printf("DNS listening on %s/%s", *addr, "udp")
		},
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
		conn, err := listener.AcceptTCP()
		log.Println(err)
		if err != nil {
			if errors.Is(err, net.ErrClosed) {
				return
			}
			time.Sleep(100 * time.Millisecond)
			continue
		}
		ip := sourceIP(conn.RemoteAddr())
		if !ip.IsValid() || !provider.InWhiteList(ip) {
			log.Printf("unknown ip: %s\n", ip.String())
			conn.Close()
			continue
		}

		go handleConn(conn)
	}
}

func sourceIP(remoteAddr net.Addr) netip.Addr {
	switch addr := remoteAddr.(type) {
	case *net.UDPAddr:
		return addr.AddrPort().Addr().Unmap()
	case *net.TCPAddr:
		return addr.AddrPort().Addr().Unmap()
	default:
		return netip.Addr{}
	}
}
