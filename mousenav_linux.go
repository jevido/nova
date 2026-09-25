//go:build linux && !android

package main

/*
#cgo pkg-config: gtk4
#include <gtk/gtk.h>

extern void novaMouseNav(uintptr_t win, int forward);

// WebKitGTK only turns the left, middle and right buttons into DOM events, so
// the page never sees the back/forward buttons. Catch them on the window and
// claim them before WebKit does.
static void nova_nav_pressed(GtkGestureClick *g, gint n, gdouble x, gdouble y, gpointer data) {
	guint button = gtk_gesture_single_get_current_button(GTK_GESTURE_SINGLE(g));
	gtk_gesture_set_state(GTK_GESTURE(g), GTK_EVENT_SEQUENCE_CLAIMED);
	novaMouseNav((uintptr_t)data, button == 9);
}

static void nova_attach_mouse_nav(void *window, uintptr_t id) {
	for (guint b = 8; b <= 9; b++) {
		GtkGesture *g = gtk_gesture_click_new();
		gtk_gesture_single_set_button(GTK_GESTURE_SINGLE(g), b);
		gtk_event_controller_set_propagation_phase(GTK_EVENT_CONTROLLER(g), GTK_PHASE_CAPTURE);
		g_signal_connect(g, "pressed", G_CALLBACK(nova_nav_pressed), (gpointer)id);
		gtk_widget_add_controller(GTK_WIDGET(window), GTK_EVENT_CONTROLLER(g));
	}
}
*/
import "C"

import (
	"sync"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
)

var (
	navMu      sync.Mutex
	navWindows = map[uintptr]*application.WebviewWindow{}
	navNext    uintptr
)

// enableMouseNav makes the mouse's back and forward buttons work in win.
func enableMouseNav(win *application.WebviewWindow) {
	var once sync.Once
	win.OnWindowEvent(events.Common.WindowShow, func(*application.WindowEvent) {
		once.Do(func() {
			navMu.Lock()
			navNext++
			id := navNext
			navWindows[id] = win
			navMu.Unlock()
			application.InvokeSync(func() {
				if p := win.NativeWindow(); p != nil {
					C.nova_attach_mouse_nav(p, C.uintptr_t(id))
				}
			})
		})
	})
	win.OnWindowEvent(events.Common.WindowClosing, func(*application.WindowEvent) {
		navMu.Lock()
		for id, w := range navWindows {
			if w == win {
				delete(navWindows, id)
			}
		}
		navMu.Unlock()
	})
}

//export novaMouseNav
func novaMouseNav(id C.uintptr_t, forward C.int) {
	navMu.Lock()
	win := navWindows[uintptr(id)]
	navMu.Unlock()
	if win == nil {
		return
	}
	dir := "back"
	if forward != 0 {
		dir = "forward"
	}
	win.DispatchWailsEvent(&application.CustomEvent{Name: EventMouseNav, Data: dir})
}
