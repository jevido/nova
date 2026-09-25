import type { SystemTheme } from "../../bindings/nova/services/models";

// Nova's own look is libadwaita's (app.css). On top of that it takes what
// the desktop says about itself: light or dark, the accent colour, the
// interface font and, with an Omarchy theme, the whole colour palette, so it
// sits next to the other apps like it belongs.

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

/** Colours for an Omarchy colors.toml, laid out like libadwaita's surfaces. */
function paletteVars(p: Record<string, string>, dark: boolean): Record<string, string> {
  const bg = p.background;
  const fg = p.foreground;
  const raised = dark ? p.lighter_background || bg : bg;
  const v: Record<string, string> = {
    "--bg": bg,
    "--view-bg": bg,
    "--header-bg": bg,
    "--sidebar-bg": (dark ? p.lighter_background : p.darker_background) || bg,
    "--sidebar-border": (dark ? p.darker_background : p.dark_background) || "transparent",
    "--popover-bg": raised,
    "--dialog-bg": raised,
    "--card-bg": `color-mix(in srgb, ${fg} 8%, transparent)`,
    "--fg": fg,
    "--fg-dim": `color-mix(in srgb, ${fg} 62%, ${bg})`,
    "--border": `color-mix(in srgb, ${fg} 16%, transparent)`,
  };
  if (p.red) v["--destructive"] = v["--destructive-text"] = p.red;
  if (p.yellow) v["--warning"] = p.yellow;
  if (p.green) v["--success"] = p.green;
  return v;
}

/** Apply the desktop's look; `dark` is the scheme Nova ended up using. */
export function applySystemTheme(t: SystemTheme | null, dark: boolean) {
  const root = document.documentElement;
  for (const k of applied) root.style.removeProperty(k);
  applied = [];
  if (!t) return;

  const v: Record<string, string> = {};
  const pal = t.palette && t.mode === (dark ? "dark" : "light") ? t.palette : null;
  if (pal?.background && pal.foreground) Object.assign(v, paletteVars(pal as Record<string, string>, dark));

  if (t.accent) {
    v["--accent"] = t.accent;
    const l = luminance(t.accent);
    v["--accent-fg"] = l !== null && l > 0.4 ? "rgba(0, 0, 0, 0.8)" : "#ffffff";
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

  for (const [k, val] of Object.entries(v)) {
    root.style.setProperty(k, val);
    applied.push(k);
  }
}
