package main

import (
	"net"
	"testing"

	"github.com/miekg/dns"
)

func TestNewDnsServer_InvalidIP(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic for invalid IP address")
		}
	}()
	newDnsServer("1.1.1.1:53", "not-an-ip")
}

func TestNewDnsServer_ValidIP(t *testing.T) {
	s := newDnsServer("1.1.1.1:53", "10.1.11.111")
	if s == nil {
		t.Fatal("expected non-nil server")
	}
	if !s.defaultIp.Equal(net.ParseIP("10.1.11.111").To4()) {
		t.Fatalf("expected default IP 10.1.11.111, got %s", s.defaultIp)
	}
}

func TestDnsHandleFunc_FallbackARecord(t *testing.T) {
	s := newDnsServer("1.1.1.1:53", "10.1.11.111")

	r := new(dns.Msg)
	r.SetQuestion("nonexistent.example.com.", dns.TypeAAAA)
	r.Id = 1

	w := &mockResponseWriter{msg: nil}

	s.dnsHandleFunc(w, r)

	if w.msg == nil {
		t.Fatal("expected response message")
	}

	if len(w.msg.Answer) != 1 {
		t.Fatalf("expected 1 answer, got %d", len(w.msg.Answer))
	}

	a, ok := w.msg.Answer[0].(*dns.A)
	if !ok {
		t.Fatal("expected A record in answer")
	}

	if !a.A.Equal(net.ParseIP("10.1.11.111").To4()) {
		t.Fatalf("expected A record with 10.1.11.111, got %s", a.A)
	}

	if a.Hdr.Ttl != 300 {
		t.Fatalf("expected TTL 300, got %d", a.Hdr.Ttl)
	}
}

func TestDnsHandleFunc_NoQuestions(t *testing.T) {
	s := newDnsServer("1.1.1.1:53", "10.1.11.111")

	r := new(dns.Msg)
	r.Id = 2

	w := &mockResponseWriter{msg: nil}

	s.dnsHandleFunc(w, r)

	// Should call HandleFailed, which sets Rcode to RcodeServerFailure
	if w.msg == nil {
		t.Fatal("expected response message")
	}
	if w.msg.Rcode != dns.RcodeServerFailure {
		t.Fatalf("expected RcodeServerFailure, got %d", w.msg.Rcode)
	}
}

func TestLRUCache(t *testing.T) {
	c := newLRU[string, int](2)

	if _, ok := c.get("a"); ok {
		t.Fatal("expected cache miss")
	}

	c.put("a", 1)
	if v, ok := c.get("a"); !ok || v != 1 {
		t.Fatalf("expected cache hit with value 1, got %d, ok=%v", v, ok)
	}

	c.put("b", 2)
	c.put("c", 3) // should evict "a" (LRU)

	if _, ok := c.get("a"); ok {
		t.Fatal("expected 'a' to be evicted")
	}
	if v, ok := c.get("b"); !ok || v != 2 {
		t.Fatalf("expected 'b'=2, got %d, ok=%v", v, ok)
	}
	if v, ok := c.get("c"); !ok || v != 3 {
		t.Fatalf("expected 'c'=3, got %d, ok=%v", v, ok)
	}
}

func TestLRUCache_UpdateExistingKey(t *testing.T) {
	c := newLRU[string, int](2)

	c.put("a", 1)
	c.put("a", 10)

	if v, ok := c.get("a"); !ok || v != 10 {
		t.Fatalf("expected updated value 10, got %d, ok=%v", v, ok)
	}

	if c.list.Len() != 1 {
		t.Fatalf("expected 1 element in list, got %d", c.list.Len())
	}
}

// mockResponseWriter implements dns.ResponseWriter for testing
type mockResponseWriter struct {
	msg *dns.Msg
}

func (m *mockResponseWriter) WriteMsg(msg *dns.Msg) error {
	m.msg = msg
	return nil
}

func (m *mockResponseWriter) LocalAddr() net.Addr                { return nil }
func (m *mockResponseWriter) RemoteAddr() net.Addr               { return nil }
func (m *mockResponseWriter) Write([]byte) (int, error)          { return 0, nil }
func (m *mockResponseWriter) Close() error                       { return nil }
func (m *mockResponseWriter) TsigStatus() error                  { return nil }
func (m *mockResponseWriter) TsigTimersOnly(bool)                {}
func (m *mockResponseWriter) Hijack()                            {}
