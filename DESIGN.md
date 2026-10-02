---
name: Nova
description: A nova.storage file manager in libadwaita's own look, on the desktop, the phone, and its download site.
colors:
  accent-green: "#2b7f3e"
  accent-green-text: "#1c7a33"
  accent-green-text-dark: "#78d38a"
  accent-blue: "#3584e4"
  accent-blue-text: "#0461be"
  accent-blue-text-dark: "#78aeed"
  accent-fg: "#ffffff"
  destructive: "#e01b24"
  warning: "#e5a50a"
  fg: "rgba(0, 0, 6, 0.8)"
  fg-dark: "#ffffff"
  window-bg: "#fafafb"
  window-bg-dark: "#222226"
  view-bg: "#ffffff"
  view-bg-dark: "#1d1d20"
  sidebar-bg: "#ebebed"
  sidebar-bg-dark: "#2e2e32"
  popover-bg-dark: "#36363a"
  band-bg-dark: "#1a1a1d"
  border: "rgba(0, 0, 6, 0.15)"
  shade: "rgba(0, 0, 6, 0.07)"
  shade-dark: "rgba(0, 0, 6, 0.36)"
  toast-bg: "rgba(40, 40, 44, 0.96)"
  toast-bg-dark: "rgba(64, 64, 68, 0.97)"
typography:
  display:
    fontFamily: "Adwaita Sans, Cantarell, system-ui, -apple-system, Segoe UI, sans-serif"
    fontSize: "clamp(2.5rem, 5vw, 3.5rem)"
    fontWeight: 800
    lineHeight: 1
    letterSpacing: "-0.03em"
    fontVariation: "\"opsz\" 32"
  headline:
    fontFamily: "Adwaita Sans, Cantarell, system-ui, sans-serif"
    fontSize: "clamp(1.5rem, 2.6vw, 1.875rem)"
    fontWeight: 800
    lineHeight: 1.15
    letterSpacing: "-0.02em"
  title:
    fontFamily: "Adwaita Sans, Cantarell, system-ui, sans-serif"
    fontSize: "1.0625rem"
    fontWeight: 700
  body:
    fontFamily: "Adwaita Sans, Cantarell, system-ui, sans-serif"
    fontSize: "16px"
    fontWeight: 400
    lineHeight: 1.5
  body-app:
    fontFamily: "Adwaita Sans, Cantarell, system-ui, sans-serif"
    fontSize: "14.667px"
    fontWeight: 400
    lineHeight: 1.35
  label:
    fontFamily: "Adwaita Sans, Cantarell, system-ui, sans-serif"
    fontSize: "0.9375rem"
    fontWeight: 700
    lineHeight: 1.2
  caption:
    fontFamily: "Adwaita Sans, Cantarell, system-ui, sans-serif"
    fontSize: "0.875rem"
    fontWeight: 400
  mono:
    fontFamily: "Adwaita Mono, Source Code Pro, ui-monospace, monospace"
    fontSize: "0.875em"
    fontWeight: 400
rounded:
  xs: "4px"
  sm: "6px"
  md: "8px"
  lg: "12px"
  pill: "999px"
spacing:
  gutter: "clamp(16px, 4vw, 40px)"
  content-max: "960px"
  row-x: "16px"
  row-y: "10px"
  group-gap: "36px"
  section-top: "48px"
  headerbar-height: "47px"
components:
  button:
    backgroundColor: "color-mix(in srgb, {colors.fg} 10%, transparent)"
    textColor: "{colors.fg}"
    typography: "{typography.label}"
    rounded: "{rounded.sm}"
    padding: "6px 14px"
    height: "34px"
  button-flat-icon:
    backgroundColor: "transparent"
    textColor: "{colors.fg}"
    rounded: "{rounded.sm}"
    size: "34px"
  button-pill-suggested:
    backgroundColor: "{colors.accent-green}"
    textColor: "{colors.accent-fg}"
    typography: "{typography.label}"
    rounded: "{rounded.pill}"
    padding: "10px 28px"
    height: "44px"
  headerbar:
    backgroundColor: "{colors.view-bg}"
    textColor: "{colors.fg}"
    height: "47px"
    padding: "0 6px 0 12px"
  view-switcher-item:
    backgroundColor: "transparent"
    textColor: "{colors.fg}"
    typography: "{typography.label}"
    rounded: "{rounded.sm}"
    padding: "0 14px"
    height: "34px"
  boxed-list:
    backgroundColor: "{colors.view-bg}"
    rounded: "{rounded.lg}"
  boxed-list-row:
    textColor: "{colors.fg}"
    padding: "10px 16px"
    height: "56px"
  expander-row:
    textColor: "{colors.fg}"
    padding: "10px 16px"
    height: "64px"
  popover:
    backgroundColor: "{colors.view-bg}"
    rounded: "{rounded.lg}"
    padding: "6px"
    width: "220px"
  toast:
    backgroundColor: "{colors.toast-bg}"
    textColor: "#ffffff"
    rounded: "{rounded.pill}"
    padding: "6px 6px 6px 18px"
    height: "44px"
  accent-badge:
    backgroundColor: "color-mix(in srgb, {colors.accent-green} 15%, transparent)"
    textColor: "{colors.accent-green-text}"
    rounded: "{rounded.pill}"
    padding: "2px 10px"
  style-switcher-swatch:
    rounded: "{rounded.pill}"
    size: "44px"
---

# Design System: Nova

## Overview

**Creative North Star: "Nova's Own Window"**

Nova does not have a brand look separate from its software: the visual system is
libadwaita, as GNOME Files 50 renders it, re-created in CSS. The desktop and
mobile app (`frontend/src/app.css`) is the source of truth; the download site
(`website/`) is laid out as a GNOME Software app details page and borrows the
app's surfaces, hairlines, radii, buttons, lists, popovers and toasts token for
token. The screenshots on the site are not pictures but working Nova windows
built from the same rules.

The mood is calm, familiar and dense in the GNOME way: neutral grey-white
surfaces, one accent hue, bold sans labels, symbolic icons, and depth from
hairline rings and soft layered shadows rather than borders or gradients.
Everything follows the system light or dark preference, and every surface offers
the libadwaita style switcher (System / Light / Dark) in its main menu. The one
deliberate difference between the two expressions is the accent: the app keeps
libadwaita's default blue, the website uses the logo's green as its libadwaita
accent.

The site explicitly refuses the dark-gradient split hero with a tilted product
mockup; it presents Nova the way a Linux user already judges apps.

**Key Characteristics:**
- libadwaita surfaces: window, view, sidebar, popover, card, with paired light
  and dark values.
- A single accent hue per surface (blue in the app, logo green on the site);
  everything else is neutral.
- Adwaita Sans (variable, self-hosted) and Adwaita Mono; no other faces.
- Adwaita symbolic icons from one SVG sprite, 16px, `currentColor`.
- Depth by hairline ring plus soft shadow stacks; separators are 1px inset shade
  lines.
- Rounded but not soft: 6px controls, 12px containers, full pills for primary
  actions, toasts and badges.

## Colors

A neutral libadwaita palette with exactly one accent hue, used for primary
actions, selection, focus, links and small badges.

### Primary
- **Logo Green** (`accent-green`): the website's accent background. Fills the
  suggested-action Install and Download pills, the skip link, switch-on state,
  the style-switcher selection ring and its check badge. Darkened from the logo
  green so white text on it passes contrast (about 5:1).
- **Leaf Text Green** (`accent-green-text`, dark: `accent-green-text-dark`): the
  website's accent as foreground. Links, small accent buttons, badges and tags,
  "Recommended" file labels, disclosure summaries. Roughly 5.2:1 on the light
  window and 8.7:1 on the dark one.
- **Adwaita Blue** (`accent-blue`, text `accent-blue-text`, dark text
  `accent-blue-text-dark`): libadwaita's default accent, used by the app itself
  for the same roles. The site does not use it.

### Secondary
- **Warning Amber** (`warning`): starred-file stars and the tinted warning
  banner (16% mix behind text). Never a fill for actions.
- **Destructive Red** (`destructive`): the app's destructive actions. Declared
  on the site but not rendered there.

### Neutral
- **Window Grey-White** (`window-bg`, dark `window-bg-dark`): the page and
  window background.
- **View White** (`view-bg`, dark `view-bg-dark`): header bar, content views,
  cards and popovers in light mode.
- **Sidebar Grey** (`sidebar-bg`, dark `sidebar-bg-dark`): the app sidebar, and
  on the site the screenshot band behind the live windows (dark band
  `band-bg-dark`).
- **Popover Charcoal** (`popover-bg-dark`): popovers and dialogs in dark mode.
- **Ink** (`fg`, dark `fg-dark`): text at 80% near-black in light mode, white in
  dark. Dim text is the same ink at about 55-62% alpha (app 0.55, site 0.6 light
  / 0.62 dark).
- **Hairline** (`border`) and **Shade** (`shade`, dark `shade-dark`): `border`
  for menu separators and swatch rings, `shade` for row separators, sidebar edge
  and band edges.
- **Toast Smoke** (`toast-bg`, dark `toast-bg-dark`): the near-opaque dark pill
  behind toasts, white text in both modes.

State tints are derived, not new colours: hover is ink at 7%, active 16%, button
fill 10% / hover 15% / pressed 30%; selection is the accent at 22% on the site
(25% in the app), focus ring the accent at 55% (50% in the app).

### Named Rules
**The One Hue Rule.** A surface carries one accent hue and nothing else
saturated, apart from amber stars and warnings. The app's is Adwaita blue; the
website's is logo green. Do not mix them on one surface.

**The Mixed, Not Picked Rule.** Hover, pressed, selected and tinted-badge states
are `color-mix` percentages of ink or accent over transparent, so they work on
any surface in both modes. Don't introduce hand-picked hover hexes.

## Typography

**Display Font:** Adwaita Sans (variable 100-900, self-hosted woff2; falls back
to Cantarell, system-ui) **Body Font:** Adwaita Sans **Label/Mono Font:**
Adwaita Mono (with Source Code Pro, ui-monospace) for commands, file names in
download lists, and code

**Character:** One humanist GNOME sans doing every job, separated only by weight
and size; heavy 800 headings against 400 body, with bold 700 labels the way
libadwaita sets buttons and titles.

### Hierarchy
- **Display** (800, clamp(2.5rem, 5vw, 3.5rem), line-height 1, -0.03em, opsz
  32): the app name in the site's app header. One per page.
- **Headline** (800, clamp(1.5rem, 2.6vw, 1.875rem), 1.15, -0.02em): section
  headings such as the description title; group headings that act as section
  titles use a slightly smaller clamp (1.375rem to 1.625rem, -0.015em).
- **Title** (700, 1.0625rem): boxed-list group titles, expander row titles,
  release versions.
- **Body** (400, 16px, 1.5): site copy; prose paragraphs at 1.0625rem, measure
  66ch. Inside Nova windows the app's body is 14.667px (11pt, GNOME's default)
  at 1.35.
- **Label** (700, 0.9375rem, 1.2): buttons, view-switcher items, toasts (600).
- **Caption** (400, 0.875rem, dim ink): row subtitles, metadata, carousel
  caption, footer.

### Named Rules
**The Weight Not Face Rule.** Hierarchy comes from Adwaita Sans weight (400 /
600 / 700 / 800) and size alone. No second display face, no uppercase tracking
labels.

**The Tabular Numbers Rule.** Sizes, dates and version dates use
`font-variant-numeric: tabular-nums`.

## Layout

The site is a single centred column (`content-max`, inside a fluid `gutter`)
under a sticky full-width header bar, broken once by a full-bleed screenshot
band. Sections stack with `section-top` spacing; groups of boxed lists stack at
`group-gap`, each with a bold title above and optional dim description or
footnote. Rows have a 56px minimum (64px for expander rows) with 10px by 16px
padding.

The header bar is a three-column grid: title at start, view switcher centred,
GitHub and main-menu buttons at end. The app header is a three-column grid
(128px icon, text, action column right-aligned).

Responsive behaviour, observed: at 860px the switcher drops its icons, the
Install column wraps under the text and the context tiles go from four to two
columns, carousel OSD arrows hide. At 560px the app header centres and stacks
with a 96px icon, the Install pill stretches full width, shortcut keys in rows
hide, and expander bodies lose their icon indent. At 480px the GitHub button
leaves the header bar.

The Nova windows keep the app's own metrics: 47px header bars, 220px sidebar,
104px grid cells, 360 by 740 phone with a 64px bottom bar.

## Elevation & Depth

Hybrid, in libadwaita's manner: surfaces are flat colour planes separated by 1px
inset shade lines, and only containers lift, using layered shadow stacks whose
first layer is a 1px ring that does the job of a border. The header bar is flat
at rest and gains a shade line plus soft drop only once the page scrolls under
it. Dark mode deepens every stack rather than inverting it.

### Shadow Vocabulary
- **Card shade**
  (`0 0 0 1px rgba(0,0,6,0.03), 0 1px 3px 1px rgba(0,0,6,0.07), 0 2px 6px 2px rgba(0,0,6,0.03)`):
  boxed lists, cards, tiles, the properties list, the QR plate.
- **Menu shadow**
  (`0 0 0 1px rgba(0,0,6,0.07), 0 1px 5px 1px rgba(0,0,6,0.09), 0 2px 14px 3px rgba(0,0,6,0.05)`):
  popovers and context menus.
- **Window shadow**
  (`0 0 0 1px rgba(0,0,6,0.1), 0 2px 8px 2px rgba(0,0,6,0.1), 0 18px 40px -6px rgba(0,0,6,0.22)`):
  the live Nova windows on the site.
- **Toast shadow** (`0 2px 10px rgba(0,0,0,0.3)`): toasts.
- **Separator** (`inset 0 1px 0 shade`): between rows, release items, expander
  body, band edges, phone bottom bar.

### Named Rules
**The Ring Not Border Rule.** Containers are edged by the first 1px ring of
their shadow stack, and rows are split by inset shade lines. Don't add CSS
borders to cards, lists or popovers.

## Shapes

Gently rounded, libadwaita radii: 4px for inner segments (path-bar segments,
thumbnails), 6px for every control (buttons, rows, switcher items, menu items,
focus outline), 8px for command fields and file links, 12px for containers
(boxed lists, cards, tiles, popovers, windows, grid items), and full pills for
primary actions, toasts, badges, tags, switches and the tile icon wells. The
properties dialog uses 14px, the phone 36px inside a 46px frame. The
style-switcher swatches are circles.

## Components

### Buttons
Quiet tinted rectangles, with one loud green pill.
- **Shape:** gently rounded (6px), 34px tall; pill variant 44px and fully round.
- **Default:** ink at 10% fill, bold label; hover 15%, pressed 30%.
- **Flat / icon:** transparent, 34px square for icon buttons, hover 7% and
  pressed 16% ink.
- **Suggested pill:** logo-green fill, white bold label, 10px by 28px; hover
  mixes 9% white, pressed 12% black. The Install pill is at least 220px wide.
- **Small:** 28px, accent-text label, used for Copy in command fields.
- **Focus:** 2px outline at accent 55%, offset 2px, 6px radius, everywhere.

### Header bar
Sticky, 47px, view-bg at 92% with `saturate(1.4) blur(12px)` backdrop, flat
until scrolled. Title (24px logo plus bold "Nova") at start, flat icon buttons
at end.

### View switcher
Inline, header-centred links (Overview / Download / What's New), each an icon
plus bold label, 34px, 6px radius; hover ink 7%, current page ink 16%. No
underline indicator.

### Boxed lists
- **Corner Style:** 12px, clipped.
- **Background:** card-bg (white light, white 8% dark).
- **Shadow Strategy:** card shade; rows split by inset shade lines.
- **Rows:** title (600) over dim caption, optional 16px leading symbolic icon at
  85% opacity, optional trailing keycaps or chevron. Link rows hover at ink 7%.

### Expander rows
Boxed-list rows built on `details`: 64px summary with a 20px platform icon,
1.0625rem title, an accent "This device" style badge, and a pan-down chevron
that rotates 180 degrees. The open body sits on a 2% ink tint behind an inset
shade line, indented to the title (50px), and holds pills, command fields, file
links and nested disclosures.

### Popovers
12px, 6px padding, popover-bg, menu shadow; opens from top-right with a 0.16s
scale-fade (0.94 to 1, rising 4px). Items are 6px-rounded rows; separators are
border lines at 60% opacity.

### Style switcher
The first block of the main menu: three 44px circular swatches (System split
diagonally, Light, Dark) with a 1px border ring. The selected one gets a 2px gap
ring and a 2px accent ring plus an 18px accent check badge at bottom right.
Choice is applied as `data-theme` on the root.

### Toasts
Dark pill (toast-bg), 44px, white 600 label, optional pill action button on
white 12% and a close icon. Rises 16px with a slight scale over 0.3s, leaves in
0.2s. Used for Undo in the live windows.

### Context tiles and badges
Tiles are centred 12px cards with an accent-tinted pill icon well (16% accent,
accent-text icon), bold title and dim caption. Badges and tags are small pill
chips: accent at 14-15% behind accent-text, bold 0.75-0.8125rem.

### Live Nova windows (signature)
The screenshot carousel's slides are working Nova windows (desktop, phone,
properties dialog) using the app's metrics and tokens: sidebar rows, path bar
with segment buttons, grid and list views, selection at accent 22%, context
menu, Undo toast, phone drawer and bottom bar. Carousel slides snap horizontally
with round dark OSD arrows and dot tabs. Motion: carousel slide, toast rise,
popover scale-fade, drawer slide, ease `cubic-bezier(0.22, 1, 0.36, 1)`; all
reduced to near zero under `prefers-reduced-motion`.

## Do's and Don'ts

### Do:
- **Do** take surface, ink, hairline and shadow values from the libadwaita token
  set in `frontend/src/app.css`, with both light and dark values.
- **Do** keep a single accent per surface: Adwaita blue (#3584e4) in the app,
  logo green (#2b7f3e fill, #1c7a33 / #78d38a text) on the website.
- **Do** derive hover, pressed, selected and tinted states by `color-mix` from
  ink or accent.
- **Do** use Adwaita symbolic icons from the sprite at 16px in `currentColor`.
- **Do** group settings-like and feature content in 12px boxed lists with inset
  shade separators.
- **Do** offer the System / Light / Dark style switcher in the main menu and
  follow the system scheme by default.
- **Do** reduce all animation and transition durations under
  `prefers-reduced-motion`.

### Don't:
- **Don't** build the dark-gradient split hero with a tilted product mockup.
- **Don't** put CSS borders on cards, lists or popovers; use the shadow ring and
  inset shade lines.
- **Don't** add a second accent hue or a second type family.
- **Don't** use emoji or text glyphs as icons; use the Adwaita symbolic sprite.
