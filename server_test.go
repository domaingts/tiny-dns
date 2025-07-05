package main

import (
	"context"
	"net"
	"testing"
)

func TestQueryDns(t *testing.T) {
	resolver := &net.Resolver{
		PreferGo: true,
		Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
			d := net.Dialer{}
			return d.Dial("ip", "127.0.0.1:15553")
		},
	}

	ips, err := resolver.LookupIP(context.TODO(), "udp", "wwww.google.com")
	if err != nil {
		panic(err)
	}
	t.Log(ips)
}