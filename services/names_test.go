package services

import "testing"

func TestUniqueName(t *testing.T) {
	taken := map[string]bool{"a.txt": true, "a (copy).txt": true, "dir": true, ".hidden": true}
	cases := []struct {
		name  string
		isDir bool
		want  string
	}{
		{"b.txt", false, "b.txt"},
		{"a.txt", false, "a (copy 2).txt"},
		{"dir", true, "dir (copy)"},
		{".hidden", false, ".hidden (copy)"},
	}
	for _, c := range cases {
		if got := uniqueName(c.name, taken, c.isDir); got != c.want {
			t.Errorf("uniqueName(%q) = %q, want %q", c.name, got, c.want)
		}
	}
}

func TestCheckPath(t *testing.T) {
	for _, p := range []string{"/me", "/me/a", "/me//a/../b"} {
		if _, err := checkPath(p); err != nil {
			t.Errorf("checkPath(%q) unexpected error %v", p, err)
		}
	}
	for _, p := range []string{"/", "/other", "/me/../etc", "/mex"} {
		if _, err := checkPath(p); err == nil {
			t.Errorf("checkPath(%q) should fail", p)
		}
	}
}

func TestSafeLocalName(t *testing.T) {
	for in, want := range map[string]string{"a/b": "a_b", "..": "_", "": "_", `a\b`: "a_b", "ok.txt": "ok.txt"} {
		if got := safeLocalName(in); got != want {
			t.Errorf("safeLocalName(%q) = %q, want %q", in, got, want)
		}
	}
}
