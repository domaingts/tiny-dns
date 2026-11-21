package main

import (
	"bytes"
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"time"
)

var errStopHandshake = errors.New("stop handshake")

var resolver = &net.Resolver{
	PreferGo: true,

	Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
		dialer := &net.Dialer{
			Timeout: 3 * time.Second,
		}

		return dialer.DialContext(
			ctx, "tcp4", "1.1.1.1:53",
		)
	},
}

var dialer = &net.Dialer{
	Timeout:  5 * time.Second,
	Resolver: resolver,
}

type sniffConn struct {
	net.Conn
	reader io.Reader
}

func (c *sniffConn) Read(p []byte) (int, error) {
	return c.reader.Read(p)
}

func (c *sniffConn) Write(b []byte) (int, error) {
	return len(b), nil
}

func handleConn(client *net.TCPConn) {
	defer client.Close()

	client.SetReadDeadline(
		time.Now().Add(5 * time.Second),
	)

	sni, data, err := readSNI(client)
	if err != nil || sni == "" {
		return
	}

	client.SetReadDeadline(time.Time{})

	addr := net.JoinHostPort(sni, "443")

	conn, err := dialer.Dial("tcp4", addr)
	if err != nil {
		log.Println(err)
		return
	}

	backend := conn.(*net.TCPConn)
	defer backend.Close()

	log.Printf("%s -> %s", sni, backend.RemoteAddr())

	backend.SetWriteDeadline(
		time.Now().Add(5 * time.Second),
	)

	if _, err := io.Copy(
		backend, bytes.NewBuffer(data),
	); err != nil {
		return
	}

	backend.SetDeadline(time.Time{})

	relay(client, backend)
}

func readSNI(conn net.Conn) (string, []byte, error) {
	var sni string
	var buf bytes.Buffer

	reader := io.TeeReader(
		io.LimitReader(conn, 128*1024),
		&buf,
	)

	wrapped := &sniffConn{
		Conn:   conn,
		reader: reader,
	}

	tlsConn := tls.Server(wrapped, &tls.Config{
		GetConfigForClient: func(hello *tls.ClientHelloInfo) (*tls.Config, error) {
			sni = hello.ServerName

			// stop the handshake after getting the sni
			return nil, errStopHandshake
		},
	})

	err := tlsConn.Handshake()

	if !errors.Is(err, errStopHandshake) {
		return "", nil, fmt.Errorf(
			"parse ClentHello: %v", err,
		)
	}

	return sni, buf.Bytes(), nil
}

func relay(a, b *net.TCPConn) {
	done := make(chan struct{})

	transfer := func(dst, src *net.TCPConn) {
		_, err := io.Copy(dst, src)

		if err != nil {
			a.Close()
			b.Close()
			return
		}

		dst.CloseWrite()
	}

	go func() {
		defer close(done)
		transfer(a, b)
	}()

	transfer(b, a)
	<-done
}
