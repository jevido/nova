package services

import (
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"nova/internal/nova"
)

// TestLiveFiles exercises the real API inside a throwaway folder.
// Run with: NOVA_API_KEY=... go test ./services -run Live -v
func TestLiveFiles(t *testing.T) {
	key := os.Getenv("NOVA_API_KEY")
	if key == "" {
		t.Skip("NOVA_API_KEY not set")
	}
	c := nova.NewClient(os.Getenv("NOVA_SERVER"), key)
	fs := NewFilesService(c)
	name := fmt.Sprintf("nova-desktop-livetest-%d", time.Now().UnixNano())
	root := "/me/" + name
	t.Cleanup(func() { _, _ = fs.Delete([]string{root}) })

	if _, err := fs.CreateFolder("/me", name); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	a, err := fs.CreateFolder(root, "a")
	if err != nil {
		t.Fatalf("mkdir a: %v", err)
	}
	if err := c.Upload(t.Context(), root+"/hello.txt", strings.NewReader("hi"), 2, "text/plain"); err != nil {
		t.Fatalf("upload: %v", err)
	}
	tr := NewTransferService(c)
	if _, err := tr.Copy([]string{root + "/hello.txt"}, root); err != nil {
		t.Fatalf("copy: %v", err)
	}
	for i := 0; i < 100; i++ {
		if st := tr.List()[0].State; st != StateQueued && st != StateRunning {
			if st != StateDone {
				t.Fatalf("copy job: %+v", tr.List()[0])
			}
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	if res, err := fs.Move([]string{root + "/hello (copy).txt"}, a); err != nil || len(res.Done) != 1 {
		t.Fatalf("move: %v %+v", err, res)
	}
	if res, _ := fs.Move([]string{a}, a); len(res.Errors) != 1 {
		t.Fatalf("expected move-into-self error, got %+v", res)
	}
	if _, err := fs.Rename(a+"/hello (copy).txt", "renamed.txt"); err != nil {
		t.Fatalf("rename: %v", err)
	}
	f, err := fs.List(a)
	if err != nil || len(f.Children) != 1 || f.Children[0].Name != "renamed.txt" {
		t.Fatalf("list: %v %+v", err, f)
	}
	url, err := fs.Share(root+"/hello.txt", true)
	if err != nil || url == "" {
		t.Fatalf("share: %v %q", err, url)
	}
	t.Logf("share url %s", url)
	if _, err := fs.Share(root+"/hello.txt", false); err != nil {
		t.Fatalf("unshare: %v", err)
	}
	dl := t.TempDir()
	if _, err := tr.Download([]string{a}, dl); err != nil {
		t.Fatalf("download: %v", err)
	}
	for i := 0; i < 100 && tr.List()[len(tr.List())-1].State != StateDone; i++ {
		time.Sleep(100 * time.Millisecond)
	}
	if b, err := os.ReadFile(dl + "/a/renamed.txt"); err != nil || string(b) != "hi" {
		t.Fatalf("downloaded file: %v %q (%+v)", err, b, tr.List())
	}
	sz, err := fs.Measure(root)
	if err != nil || sz.Files != 2 || sz.Bytes != 4 {
		t.Fatalf("measure: %v %+v", err, sz)
	}
}
