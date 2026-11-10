package domains_test

import (
	"strings"
	"testing"

	"github.com/charpand/terraform-provider-openprovider/internal/client/domains"
)

// FuzzSplitFullName checks that a successful split loses nothing and never
// yields an empty side, whatever the input.
func FuzzSplitFullName(f *testing.F) {
	for _, seed := range []string{"example.com", "example.co.uk", "nodot", ".com", "example.", "", ".", "a..b"} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, domainName string) {
		name, extension, ok := domains.SplitFullName(domainName)
		if !ok {
			if name != "" || extension != "" {
				t.Fatalf("SplitFullName(%q) = %q, %q, false; want empty parts on failure", domainName, name, extension)
			}
			return
		}
		if name == "" || extension == "" {
			t.Fatalf("SplitFullName(%q) = %q, %q, true; want no empty part", domainName, name, extension)
		}
		if strings.Contains(name, ".") {
			t.Fatalf("SplitFullName(%q): name %q holds a dot; the split is on the first dot", domainName, name)
		}
		if name+"."+extension != domainName {
			t.Fatalf("SplitFullName(%q) = %q, %q; joining them does not give the input back", domainName, name, extension)
		}
	})
}
