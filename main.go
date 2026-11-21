package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/netip"
	"strings"
	"time"

	"github.com/miekg/dns"
)

func main() {
	addr := flag.String("addr", ":53", "the address of the dns server")
	upstreamDNS := flag.String("dns", "1.1.1.1:53", "the upstream dns server")
	tcpPort := flag.Int64("tcp", 443, "the tcp server's port")
	defaultIP := flag.String("default", "", "the default ip when no ip returned")
	configPath := flag.String("config", "/usr/local/tiny-dns/config.json", "the config file including a white list")
	flag.Parse()

	var defaultAddr netip.Addr
	var err error
	if *defaultIP != "" {
		defaultAddr, err = netip.ParseAddr(*defaultIP)
	} else {
		defaultAddr, err = getPublicIpv6()
	}
	if err != nil {
		panic(err)
	}

	provider, err := NewFileProvider(*configPath)
	if err != nil {
		panic(err)
	}

	defer provider.Close()

	s := newDnsServer(*upstreamDNS, defaultAddr, provider)

	// dns.HandleFunc(".", s.dnsHandleFunc)

	server := &dns.Server{
		Addr:    *addr,
		Net:     "udp",
		Handler: dns.HandlerFunc(s.dnsHandleFunc),
		NotifyStartedFunc: func() {
			log.Printf("DNS listening on [%s]/%s", fmt.Sprintf("%s%s", defaultAddr.String(), *addr), "udp")
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
		if err != nil {
			log.Println(err)
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

func getPublicIpv6() (netip.Addr, error) {
	url := "https://api6.ipify.org"

	client := &http.Client{
		Timeout: 5 * time.Second,
	}
	resp, err := client.Get(url)
	if err != nil {
		return netip.Addr{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return netip.Addr{}, fmt.Errorf(
			"unexpected status: %d", resp.StatusCode,
		)
	}

	data, err := io.ReadAll(io.LimitReader(resp.Body, 128))
	if err != nil {
		return netip.Addr{}, err
	}

	return netip.ParseAddr(strings.TrimSpace(string(data)))
}
