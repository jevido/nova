import type { SystemTheme } from "../../bindings/nova/services/models";

// Nova's own look is libadwaita's (app.css). On top of that it takes what
// the desktop says about itself: light or dark, the accent colour, the
// interface font and, with an Omarchy theme or a KDE colour scheme, the whole
// palette, so it sits next to the other apps like it belongs.

let applied: string[] = [];

function luminance(hex: string): number | null {
  const m = /^#?([0-9a-f]{6})$/i.exec(hex.trim());
  if (!m) return null;
  const n = parseInt(m[1], 16);
  const lin = (c: number) => {
    c /= 255;
    return c <= 0.03928 ? c / 12.92 : ((c + 0.055) / 1.055) ** 2.4;
  };
  return 0.2126 * lin(n >> 16) + 0.7152 * lin((n >> 8) & 255) + 0.0722 * lin(n & 255);
}

/** Text colour that reads on `bg`. */
function textOn(bg: string): string | null {
  const l = luminance(bg);
  return l === null ? null : l > 0.3 ? "rgba(0, 0, 0, 0.8)" : "#ffffff";
}

/** CSS variables for a desktop palette (see the Pal* keys in services/theme.go). */
function paletteVars(p: Record<string, string>): Record<string, string> {
  const v: Record<string, string> = {};
  const set = (key: string, ...vars: string[]) => {
    if (p[key]) for (const name of vars) v[name] = p[key];
  };
  set("window", "--bg");
  set("view", "--view-bg");
  set("header", "--header-bg");
  set("sidebar", "--sidebar-bg");
  set("popover", "--popover-bg", "--dialog-bg");
  set("fg", "--fg");
  set("fgDim", "--fg-dim");
  set("accentFg", "--accent-fg");
  set("destructive", "--destructive", "--destructive-text");
  set("warning", "--warning");
  set("success", "--success");
  const bg = p.view || p.window;
  if (p.fg && bg && !p.fgDim) v["--fg-dim"] = `color-mix(in srgb, ${p.fg} 62%, ${bg})`;
  if (p.fg) {
    v["--border"] = `color-mix(in srgb, ${p.fg} 16%, transparent)`;
    v["--card-bg"] = `color-mix(in srgb, ${p.fg} 7%, transparent)`;
    v["--sidebar-border"] = `color-mix(in srgb, ${p.fg} 8%, transparent)`;
  }
  return v;
}

const CACHE = "nova-theme";

/** Apply the desktop's look; `dark` is the scheme Nova ended up using. */
export function applySystemTheme(t: SystemTheme | null, dark: boolean) {
  const root = document.documentElement;
  for (const k of applied) root.style.removeProperty(k);
  applied = [];
  if (!t) return;

  const v: Record<string, string> = {};
  const pal = t.palette && t.mode === (dark ? "dark" : "light") ? t.palette : null;
  if (pal) Object.assign(v, paletteVars(pal as Record<string, string>));

  if (t.accent) {
    v["--accent"] = t.accent;
    v["--accent-fg"] ??= textOn(t.accent) ?? "#ffffff";
    // A desktop palette picks its accent for its own background; a plain
    // accent colour is tuned for text like libadwaita does.
    v["--accent-text"] = pal
      ? t.accent
      : dark
        ? `color-mix(in srgb, ${t.accent} 60%, white)`
        : `color-mix(in srgb, ${t.accent} 75%, black)`;
  } else if (t.platform === "darwin") {
    v["--accent"] = "AccentColor";
    v["--accent-text"] = "AccentColor";
  }

  const fallback = `"Adwaita Sans", "Cantarell", "Inter", system-ui, -apple-system, "Segoe UI", sans-serif`;
  if (t.platform === "darwin") v["--font"] = `-apple-system, BlinkMacSystemFont, ${fallback}`;
  else if (t.font) v["--font"] = `"${t.font.replace(/"/g, "")}", ${fallback}`;
  if (t.fontSize >= 6 && t.fontSize <= 24) v["--font-size"] = `${(t.fontSize * 4) / 3}px`;

  if (pal?.destructive) v["--destructive-fg"] = textOn(pal.destructive) ?? "#ffffff";

  for (const [k, val] of Object.entries(v)) {
    root.style.setProperty(k, val);
    applied.push(k);
  }
  // index.html applies this before anything draws, so a new window starts
  // in the desktop's colours instead of switching after a moment.
  try {
    localStorage.setItem(CACHE, JSON.stringify({ theme: root.dataset.theme, vars: v }));
  } catch {
    /* storage may be unavailable */
  }
}
