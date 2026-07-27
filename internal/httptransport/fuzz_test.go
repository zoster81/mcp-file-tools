package httptransport

import (
	"bytes"
	"net/http"
	"net/netip"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/jsonrpc"
)

func FuzzNormalizeHostValue(f *testing.F) {
	for _, seed := range []string{
		"localhost:8765",
		"MCP.EXAMPLE.TEST:443",
		"127.0.0.1",
		"[::1]:8765",
		"user@example.test",
		"*.example.test",
		"example.test:not-a-port",
	} {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, value string) {
		if len(value) > 4096 {
			t.Skip()
		}
		normalized, err := normalizeHostValue(value)
		if err != nil {
			return
		}
		if normalized == "" || normalized != strings.TrimSpace(normalized) {
			t.Fatalf("invalid normalized host %q from %q", normalized, value)
		}
		for _, current := range normalized {
			if current <= 0x20 || current >= 0x7f {
				t.Fatalf("normalized host contains unsupported rune %q: %q", current, normalized)
			}
		}
		again, err := normalizeHostValue(normalized)
		if err != nil || again != normalized {
			t.Fatalf("host normalization is not idempotent: %q -> %q, err=%v", normalized, again, err)
		}
	})
}

func FuzzNormalizeOrigin(f *testing.F) {
	for _, seed := range []string{
		"https://app.example.test",
		"https://app.example.test:443",
		"http://127.0.0.1:8765",
		"null",
		"https://user@app.example.test",
		"https://app.example.test/path",
	} {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, value string) {
		if len(value) > 4096 {
			t.Skip()
		}
		normalized, err := normalizeOrigin(value)
		if err != nil {
			return
		}
		if !strings.HasPrefix(normalized, "http://") && !strings.HasPrefix(normalized, "https://") {
			t.Fatalf("normalized origin has unsupported scheme: %q", normalized)
		}
		again, err := normalizeOrigin(normalized)
		if err != nil || again != normalized {
			t.Fatalf("origin normalization is not idempotent: %q -> %q, err=%v", normalized, again, err)
		}
	})
}

func FuzzTrustedProxyClientAddress(f *testing.F) {
	for _, seed := range []string{
		"",
		"198.51.100.10",
		"198.51.100.10, 192.0.2.20",
		"2001:db8::10, 192.0.2.20",
		"not-an-ip",
		strings.Repeat("192.0.2.1,", maxForwardedHops+1),
	} {
		f.Add(seed)
	}

	handler := &Handler{config: Config{TrustedProxyCIDRs: []netip.Prefix{
		netip.MustParsePrefix("192.0.2.0/24"),
		netip.MustParsePrefix("2001:db8::/32"),
	}}}
	peer := netip.MustParseAddr("192.0.2.1")

	f.Fuzz(func(t *testing.T, forwarded string) {
		if len(forwarded) > 4096 {
			t.Skip()
		}
		request := &http.Request{Header: make(http.Header)}
		request.Header.Set("X-Forwarded-For", forwarded)
		address, err := handler.clientAddress(request, peer, true)
		if err != nil {
			return
		}
		if !address.IsValid() || address != address.Unmap() {
			t.Fatalf("invalid normalized client address %q from %q", address, forwarded)
		}
		reparsed, err := parseRemoteAddress(address.String())
		if err != nil || reparsed != address {
			t.Fatalf("client address is not stable: %q -> %q, err=%v", address, reparsed, err)
		}
	})
}

func FuzzJSONRPCMessageRoundTrip(f *testing.F) {
	for _, seed := range [][]byte{
		[]byte(`{"jsonrpc":"2.0","id":1,"method":"tools/list","params":{}}`),
		[]byte(`{"jsonrpc":"2.0","method":"notifications/initialized"}`),
		[]byte(`{"jsonrpc":"2.0","id":"request","result":{}}`),
		[]byte(`{"jsonrpc":"2.0","id":1,"error":{"code":-32600,"message":"invalid"}}`),
		[]byte(`{}`),
		[]byte(`not-json`),
	} {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, data []byte) {
		if len(data) > 64*1024 {
			t.Skip()
		}
		message, err := jsonrpc.DecodeMessage(data)
		if err != nil {
			return
		}
		encoded, err := jsonrpc.EncodeMessage(message)
		if err != nil {
			t.Fatalf("encode decoded JSON-RPC message: %v", err)
		}
		again, err := jsonrpc.DecodeMessage(encoded)
		if err != nil {
			t.Fatalf("decode canonical JSON-RPC message: %v", err)
		}
		reencoded, err := jsonrpc.EncodeMessage(again)
		if err != nil {
			t.Fatalf("re-encode canonical JSON-RPC message: %v", err)
		}
		if !bytes.Equal(encoded, reencoded) {
			t.Fatalf("JSON-RPC canonical encoding is unstable: %q != %q", encoded, reencoded)
		}
	})
}
