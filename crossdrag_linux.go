//go:build linux && !android

package main

/*
#cgo pkg-config: gtk4
#include <stdlib.h>
#include <gtk/gtk.h>

extern void novaDragAt(uintptr_t win, int type, double x, double y, int copy);
extern void novaDragEnded(int dropped);
extern void novaDragExport(uintptr_t task);

#define NOVA_MIME "application/x-nova-items"
#define URI_LIST "text/uri-list"

// The content of an item drag. Nova windows only look for NOVA_MIME; other
// applications ask for text/uri-list, and get the items once Go has
// downloaded them (see nova_export_done).
typedef struct { GdkContentProvider parent; } NovaItems;
typedef struct { GdkContentProviderClass parent_class; } NovaItemsClass;
// cgo copies this preamble into two objects; keep the type function static.
static GType nova_items_get_type(void);
G_DEFINE_TYPE(NovaItems, nova_items, GDK_TYPE_CONTENT_PROVIDER)

static GdkContentFormats *nova_items_ref_formats(GdkContentProvider *p) {
	const char *mimes[] = { NOVA_MIME, URI_LIST };
	return gdk_content_formats_new(mimes, 2);
}

static void nova_items_write_async(GdkContentProvider *p, const char *mime, GOutputStream *stream,
		int io_priority, GCancellable *cancellable, GAsyncReadyCallback cb, gpointer data) {
	GTask *task = g_task_new(p, cancellable, cb, data);
	g_task_set_priority(task, io_priority);
	if (g_str_equal(mime, NOVA_MIME)) {
		// The payload lives in Go.
		g_task_return_boolean(task, TRUE);
		g_object_unref(task);
		return;
	}
	if (!g_str_equal(mime, URI_LIST)) {
		g_task_return_new_error(task, G_IO_ERROR, G_IO_ERROR_NOT_SUPPORTED, "Cannot provide %s", mime);
		g_object_unref(task);
		return;
	}
	g_task_set_task_data(task, g_object_ref(stream), g_object_unref);
	novaDragExport((uintptr_t)task);
}

static gboolean nova_items_write_finish(GdkContentProvider *p, GAsyncResult *res, GError **error) {
	return g_task_propagate_boolean(G_TASK(res), error);
}

static void nova_items_class_init(NovaItemsClass *klass) {
	GdkContentProviderClass *pc = GDK_CONTENT_PROVIDER_CLASS(klass);
	pc->ref_formats = nova_items_ref_formats;
	pc->write_mime_type_async = nova_items_write_async;
	pc->write_mime_type_finish = nova_items_write_finish;
}

static void nova_items_init(NovaItems *self) {}

// Answer a text/uri-list request; err is NULL on success. Main thread only.
static void nova_export_done(uintptr_t t, const char *uris, const char *err) {
	GTask *task = G_TASK((gpointer)t);
	GError *e = NULL;
	if (err) {
		g_task_return_new_error(task, G_IO_ERROR, G_IO_ERROR_FAILED, "%s", err);
	} else if (g_output_stream_write_all(g_task_get_task_data(task), uris, strlen(uris), NULL,
			g_task_get_cancellable(task), &e)) {
		g_task_return_boolean(task, TRUE);
	} else {
		g_task_return_error(task, e);
	}
	g_object_unref(task);
}

// Only drags started by nova_start_drag in this process carry NOVA_MIME.
static gboolean nova_is_ours(GdkDrop *drop) {
	return gdk_drop_get_drag(drop) != NULL &&
		gdk_content_formats_contain_mime_type(gdk_drop_get_formats(drop), NOVA_MIME);
}

// Ctrl copies, as it does inside a window. Some compositors apply the
// modifier themselves and only offer a copy.
static int nova_wants_copy(GdkDrop *drop) {
	if (gdk_drop_get_actions(drop) == GDK_ACTION_COPY) return 1;
	GdkDevice *pointer = gdk_drop_get_device(drop);
	GdkSeat *seat = pointer ? gdk_device_get_seat(pointer) : NULL;
	GdkDevice *kbd = seat ? gdk_seat_get_keyboard(seat) : NULL;
	return kbd && (gdk_device_get_modifier_state(kbd) & GDK_CONTROL_MASK);
}

static GdkDragAction nova_over(GdkDrop *drop, double x, double y, gpointer data) {
	int copy = nova_wants_copy(drop);
	novaDragAt((uintptr_t)data, 0, x, y, copy);
	return copy ? GDK_ACTION_COPY : GDK_ACTION_MOVE;
}

static gboolean nova_accept(GtkDropTargetAsync *t, GdkDrop *drop, gpointer data) {
	return nova_is_ours(drop);
}

static GdkDragAction nova_enter(GtkDropTargetAsync *t, GdkDrop *drop, double x, double y, gpointer data) {
	return nova_over(drop, x, y, data);
}

static GdkDragAction nova_motion(GtkDropTargetAsync *t, GdkDrop *drop, double x, double y, gpointer data) {
	return nova_over(drop, x, y, data);
}

static void nova_leave(GtkDropTargetAsync *t, GdkDrop *drop, gpointer data) {
	novaDragAt((uintptr_t)data, 1, 0, 0, 0);
}

static gboolean nova_drop(GtkDropTargetAsync *t, GdkDrop *drop, double x, double y, gpointer data) {
	int copy = nova_wants_copy(drop);
	// The payload lives in Go, so there is nothing to read from the drop.
	gdk_drop_finish(drop, copy ? GDK_ACTION_COPY : GDK_ACTION_MOVE);
	novaDragAt((uintptr_t)data, 2, x, y, copy);
	return TRUE;
}

static GtkWidget *nova_find_webview(GtkWidget *w) {
	GType t = g_type_from_name("WebKitWebView");
	if (t && g_type_is_a(G_OBJECT_TYPE(w), t)) return w;
	for (GtkWidget *c = gtk_widget_get_first_child(w); c; c = gtk_widget_get_next_sibling(c)) {
		GtkWidget *r = nova_find_webview(c);
		if (r) return r;
	}
	return NULL;
}

// The target sits on the webview so drop coordinates are page coordinates.
// It runs in the capture phase, before WebKit's own drop handling.
static void nova_attach_drop_target(void *window, uintptr_t id) {
	GtkWidget *view = nova_find_webview(GTK_WIDGET(window));
	if (!view) return;
	const char *mimes[] = { NOVA_MIME };
	GtkDropTargetAsync *t = gtk_drop_target_async_new(
		gdk_content_formats_new(mimes, 1), GDK_ACTION_COPY | GDK_ACTION_MOVE);
	gtk_event_controller_set_propagation_phase(GTK_EVENT_CONTROLLER(t), GTK_PHASE_CAPTURE);
	g_signal_connect(t, "accept", G_CALLBACK(nova_accept), (gpointer)id);
	g_signal_connect(t, "drag-enter", G_CALLBACK(nova_enter), (gpointer)id);
	g_signal_connect(t, "drag-motion", G_CALLBACK(nova_motion), (gpointer)id);
	g_signal_connect(t, "drag-leave", G_CALLBACK(nova_leave), (gpointer)id);
	g_signal_connect(t, "drop", G_CALLBACK(nova_drop), (gpointer)id);
	gtk_widget_add_controller(view, GTK_EVENT_CONTROLLER(t));
}

static void nova_drag_end(GdkDrag *drag, int dropped) {
	if (g_object_get_data(G_OBJECT(drag), "nova-done")) return;
	g_object_set_data(G_OBJECT(drag), "nova-done", GINT_TO_POINTER(1));
	novaDragEnded(dropped);
	g_object_unref(drag);
}

static void nova_drag_done(GdkDrag *drag, gpointer data) {
	nova_drag_end(drag, 1);
}

static void nova_drag_cancel(GdkDrag *drag, GdkDragCancelReason reason, gpointer data) {
	nova_drag_end(drag, 0);
}

static void nova_drag_style(GdkDisplay *display) {
	static gboolean done = FALSE;
	if (done) return;
	done = TRUE;
	GtkCssProvider *css = gtk_css_provider_new();
	gtk_css_provider_load_from_string(css,
		".nova-drag-icon { background: #3584e4; color: white; padding: 4px 10px;"
		" border-radius: 6px; font-weight: 600; }");
	gtk_style_context_add_provider_for_display(display, GTK_STYLE_PROVIDER(css),
		GTK_STYLE_PROVIDER_PRIORITY_APPLICATION);
	g_object_unref(css);
}

// Start a platform drag from window, which must still have the mouse button
// held (Wayland needs the press to start a drag).
static int nova_start_drag(void *window, const char *label) {
	GdkSurface *surface = gtk_native_get_surface(GTK_NATIVE(window));
	if (!surface) return 0;
	GdkDisplay *display = gdk_surface_get_display(surface);
	GdkDevice *device = gdk_seat_get_pointer(gdk_display_get_default_seat(display));
	if (!device) return 0;
	GdkContentProvider *content = g_object_new(nova_items_get_type(), NULL);
	GdkDrag *drag = gdk_drag_begin(surface, device, content, GDK_ACTION_COPY | GDK_ACTION_MOVE, 0, 0);
	g_object_unref(content);
	if (!drag) return 0;

	nova_drag_style(display);
	GtkWidget *text = gtk_label_new(label);
	gtk_widget_add_css_class(text, "nova-drag-icon");
	gtk_drag_icon_set_child(GTK_DRAG_ICON(gtk_drag_icon_get_for_drag(drag)), text);
	gdk_drag_set_hotspot(drag, -14, -14);

	g_signal_connect(drag, "dnd-finished", G_CALLBACK(nova_drag_done), NULL);
	g_signal_connect(drag, "cancel", G_CALLBACK(nova_drag_cancel), NULL);
	return 1;
}
*/
import "C"

import (
	"errors"
	"sync"
	"unsafe"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
)

var (
	dropMu      sync.Mutex
	dropWindows = map[uintptr]*application.WebviewWindow{}
	dropNext    uintptr
)

// enableCrossDrag lets win receive item drags that left other Nova windows
// (or itself).
func enableCrossDrag(win *application.WebviewWindow) {
	var once sync.Once
	win.OnWindowEvent(events.Common.WindowShow, func(*application.WindowEvent) {
		once.Do(func() {
			dropMu.Lock()
			dropNext++
			id := dropNext
			dropWindows[id] = win
			dropMu.Unlock()
			application.InvokeSync(func() {
				if p := win.NativeWindow(); p != nil {
					C.nova_attach_drop_target(p, C.uintptr_t(id))
				}
			})
		})
	})
	win.OnWindowEvent(events.Common.WindowClosing, func(*application.WindowEvent) {
		dropMu.Lock()
		for id, w := range dropWindows {
			if w == win {
				delete(dropWindows, id)
			}
		}
		dropMu.Unlock()
	})
}

func startNativeDrag(win *application.WebviewWindow, p *DragPayload) bool {
	setDrag(p)
	label := C.CString(p.Label)
	defer C.free(unsafe.Pointer(label))
	ok := false
	application.InvokeSync(func() {
		if w := win.NativeWindow(); w != nil {
			ok = C.nova_start_drag(w, label) != 0
		}
	})
	if !ok {
		setDrag(nil)
	}
	return ok
}

//export novaDragAt
func novaDragAt(id C.uintptr_t, typ C.int, x, y C.double, copy C.int) {
	dropMu.Lock()
	win := dropWindows[uintptr(id)]
	dropMu.Unlock()
	p := currentDrag()
	if win == nil || p == nil {
		return
	}
	d := CrossDrag{X: int(x), Y: int(y), Copy: copy != 0, Payload: p}
	switch typ {
	case 0:
		d.Type = "over"
	case 1:
		d.Type = "leave"
	default:
		d.Type = "drop"
	}
	sendCrossDrag(win, d)
}

//export novaDragEnded
func novaDragEnded(dropped C.int) {
	endDrag(dropped != 0)
}

// novaDragExport answers another application's request for the dragged
// items as files, once they are downloaded.
//
//export novaDragExport
func novaDragExport(task C.uintptr_t) {
	ex := exportDrag()
	go func() {
		uris, err := "", error(nil)
		if ex == nil {
			err = errors.New("the drag has ended")
		} else {
			uris, err = ex.wait()
		}
		application.InvokeAsync(func() {
			u := C.CString(uris)
			defer C.free(unsafe.Pointer(u))
			var e *C.char
			if err != nil {
				e = C.CString(err.Error())
				defer C.free(unsafe.Pointer(e))
			}
			C.nova_export_done(task, u, e)
		})
	}()
}
