package services

import "testing"

func TestRoots(t *testing.T) {
	t.Cleanup(func() { setRoots(nil) })

	setRoots(nil)
	if _, err := checkPath("/me/a"); err != nil {
		t.Errorf("/me/a refused without a limit: %v", err)
	}
	if _, err := checkPath("/abc/a"); err == nil {
		t.Error("/abc/a allowed without a limit")
	}
	if !isRoot("/me") || isRoot("/me/a") || limited() {
		t.Error("home is not the only top folder")
	}

	setRoots([]string{"abc", "def", "bad/id", ""})
	if got := topFolders(); len(got) != 2 || got[0] != "/abc" || got[1] != "/def" {
		t.Fatalf("topFolders = %v", got)
	}
	for _, p := range []string{"/abc", "/def/x/y", "/abc/../def"} {
		if _, err := checkPath(p); err != nil {
			t.Errorf("%s refused: %v", p, err)
		}
	}
	for _, p := range []string{"/me", "/me/.nova/bookmarks.json", "/abcd", "/"} {
		if _, err := checkPath(p); err == nil {
			t.Errorf("%s allowed with a limit", p)
		}
	}
	if !isRoot("/abc") || isRoot("/abc/x") {
		t.Error("isRoot is wrong")
	}
	if !sameRoot("/abc/x", "/abc/y/z") || sameRoot("/abc/x", "/def") {
		t.Error("sameRoot is wrong")
	}
	if _, err := (&FilesService{}).Trash([]string{"/abc/x"}); err != errLimited {
		t.Errorf("Trash with a limit: %v", err)
	}
}
