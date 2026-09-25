//go:build linux && !android

package main

/*
#cgo pkg-config: gtk4
#include <stdlib.h>
#include <gtk/gtk.h>

extern void novaClipChanged(int hasFiles);
extern void novaClipFiles(uintptr_t req, char *paths, char *err);

static GdkClipboard *nova_clipboard(void) {
	GdkDisplay *d = gdk_display_get_default();
	return d ? gdk_display_get_clipboard(d) : NULL;
}

// Files copied in another application, like Nautilus' Copy. Once Nova puts
// something on the clipboard itself, it is local and Nova's own items win.
static int nova_clip_has_files(GdkClipboard *cb) {
	if (gdk_clipboard_is_local(cb)) return 0;
	GdkContentFormats *f = gdk_content_formats_union_deserialize_gtypes(
		gdk_content_formats_ref(gdk_clipboard_get_formats(cb)));
	int has = gdk_content_formats_contain_gtype(f, GDK_TYPE_FILE_LIST);
	gdk_content_formats_unref(f);
	return has;
}

static void nova_clip_changed(GdkClipboard *cb, gpointer data) {
	novaClipChanged(nova_clip_has_files(cb));
}

static void nova_watch_clipboard(void) {
	GdkClipboard *cb = nova_clipboard();
	if (!cb) return;
	g_signal_connect(cb, "changed", G_CALLBACK(nova_clip_changed), NULL);
	novaClipChanged(nova_clip_has_files(cb));
}

// Local paths, one per line; files that aren't local (sftp:// and such) are
// left out.
static void nova_clip_read_done(GObject *src, GAsyncResult *res, gpointer data) {
	GError *err = NULL;
	const GValue *v = gdk_clipboard_read_value_finish(GDK_CLIPBOARD(src), res, &err);
	if (!v) {
		novaClipFiles((uintptr_t)data, NULL, err ? err->message : "The clipboard has no files");
		g_clear_error(&err);
		return;
	}
	GString *s = g_string_new(NULL);
	for (GSList *l = g_value_get_boxed(v); l; l = l->next) {
		char *p = g_file_get_path(G_FILE(l->data));
		if (p && !strchr(p, '\n')) {
			g_string_append(s, p);
			g_string_append_c(s, '\n');
		}
		g_free(p);
	}
	novaClipFiles((uintptr_t)data, s->str, NULL);
	g_string_free(s, TRUE);
}

static void nova_read_clipboard_files(uintptr_t req) {
	GdkClipboard *cb = nova_clipboard();
	if (!cb) {
		novaClipFiles(req, NULL, "No clipboard");
		return;
	}
	gdk_clipboard_read_value_async(cb, GDK_TYPE_FILE_LIST, G_PRIORITY_DEFAULT, NULL,
		nova_clip_read_done, (gpointer)req);
}

static void nova_set_clipboard_text(const char *text) {
	GdkClipboard *cb = nova_clipboard();
	if (cb) gdk_clipboard_set_text(cb, text);
}
*/
import "C"

import (
	"errors"
	"strings"
	"sync"
	"time"
	"unsafe"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
)

var watchClipOnce sync.Once

// enableOSClipboard starts following the system clipboard once the first
// window is up (GTK has no display before that).
func enableOSClipboard(win *application.WebviewWindow) {
	win.OnWindowEvent(events.Common.WindowShow, func(*application.WindowEvent) {
		watchClipOnce.Do(func() {
			application.InvokeSync(func() { C.nova_watch_clipboard() })
		})
	})
}

//export novaClipChanged
func novaClipChanged(has C.int) {
	setOSClipboardFiles(has != 0)
}

type clipResult struct {
	paths []string
	err   error
}

var (
	clipReqMu   sync.Mutex
	clipReqs    = map[uintptr]chan clipResult{}
	clipReqNext uintptr
)

//export novaClipFiles
func novaClipFiles(req C.uintptr_t, paths, errMsg *C.char) {
	clipReqMu.Lock()
	ch := clipReqs[uintptr(req)]
	delete(clipReqs, uintptr(req))
	clipReqMu.Unlock()
	if ch == nil {
		return
	}
	var r clipResult
	if errMsg != nil {
		r.err = errors.New(C.GoString(errMsg))
	} else {
		r.paths = strings.FieldsFunc(C.GoString(paths), func(c rune) bool { return c == '\n' })
	}
	ch <- r
}

func readOSClipboardFiles() ([]string, error) {
	ch := make(chan clipResult, 1)
	clipReqMu.Lock()
	clipReqNext++
	id := clipReqNext
	clipReqs[id] = ch
	clipReqMu.Unlock()
	application.InvokeAsync(func() { C.nova_read_clipboard_files(C.uintptr_t(id)) })
	select {
	case r := <-ch:
		return r.paths, r.err
	case <-time.After(10 * time.Second):
		clipReqMu.Lock()
		delete(clipReqs, id)
		clipReqMu.Unlock()
		return nil, errors.New("the clipboard did not answer")
	}
}

func claimOSClipboard(text string) {
	c := C.CString(text)
	application.InvokeAsync(func() {
		defer C.free(unsafe.Pointer(c))
		C.nova_set_clipboard_text(c)
	})
}
