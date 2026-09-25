package icons

import "testing"

func TestLookup(t *testing.T) {
	r := NewResolver()
	for _, n := range []string{"folder", "go-previous-symbolic", "user-home-symbolic", "text-x-generic"} {
		b, ct, ok := r.Lookup(n)
		if !ok || len(b) == 0 {
			t.Errorf("%s: not found", n)
			continue
		}
		t.Logf("%s -> %s (%d bytes) %s", n, ct, len(b), r.index[n].path)
	}
	t.Logf("themes: %v", r.themes)
}
