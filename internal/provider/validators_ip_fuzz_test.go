package provider

import (
	"net/netip"
	"strings"
	"testing"
)

// FuzzValidIPOrCIDR checks that the whitelist validator never panics and
// accepts only strings netip itself reads as an address or a prefix, and
// never an address with an IPv6 zone.
func FuzzValidIPOrCIDR(f *testing.F) {
	for _, seed := range []string{"1.2.3.4", "10.0.0.0/8", "::1", "2001:db8::/32", "fe80::1%eth0", "1.2.3", "1.2.3.4/33", "", "/", "0x7f.1", "1.2.3.4 "} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, s string) {
		if !validIPOrCIDR(s) {
			return
		}
		if strings.ContainsRune(s, '%') {
			t.Fatalf("validIPOrCIDR(%q) accepted a zoned address", s)
		}
		_, addrErr := netip.ParseAddr(s)
		_, prefixErr := netip.ParsePrefix(s)
		if addrErr != nil && prefixErr != nil {
			t.Fatalf("validIPOrCIDR(%q) accepted a string netip rejects", s)
		}
	})
}
