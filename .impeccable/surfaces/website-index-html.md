---
version: 1
slug: "website-index-html"
primary_target: "website/index.html"
related_targets: []
---

# Website (nova.jevido.app)

Scope: the public download site in `website/` (index.html, style.css, main.js). Mode: **Persuade**.

Audience: nova.storage users, Linux desktop first, arriving on any device. Job: understand Nova in one look and install it on the device in hand. Action: the detected-device Install button (Linux: the curl one-liner copied; Windows: setup.exe; Android: APK; iOS: sideload steps), every other platform one click away. Proof: the app itself, rebuilt live from its own libadwaita tokens with labelled sample files; real release version and changelog; checksums. Constraints: independent client framing (not official); no invented sizes, ratings, users or license names (repo has no LICENSE file); keep `?ref=jeffero` links; static nginx, no build step; works without JS.

## Direction contract

THESIS: Nova's site is its own GNOME Software details page, the screen a Linux user already trusts to judge an app, except the screenshots are working Nova windows. It refuses the dark-gradient split hero with a tilted mockup.

OWN-WORLD: libadwaita surfaces taken from `frontend/src/app.css`: window #fafafb / #222226, view #fff / #1d1d20, sidebar #ebebed / #2e2e32, shade hairlines, 6/12px radii, pill buttons, boxed lists, Adwaita Sans (variable, self-hosted) and Adwaita Mono, Adwaita symbolic icons. The only hue is the logo's green as libadwaita accent. Follows system light/dark, with the main-menu style switcher.

STORY: The visitor recognises an app page (icon, name, Install), plays with the live window (folders, grid/list, trash + Undo, phone layout), scans context tiles and the feature list, then installs from the hero button or the per-platform list.

FIRST VIEWPORT: Full-width header bar (icon + Nova, view switcher Overview / Download / What's New, GitHub + main menu). App header: 128px icon, "Nova" large, one-line summary, developer line; green Install pill for the detected device at right with "Other platforms" under it. Below, a full-bleed screenshot band whose live Nova window starts above the fold.

FORM: GNOME Software app details page; #3 of the re-rolled ranked list; seed key ca0e8c8f (re-roll 1). Signature interaction: the carousel's slides are live windows (click, switch views, Delete then Undo toast, Ctrl+Z, phone slide with bottom bar). Motion grammar: libadwaita carousel slide, toast rise, popover scale-fade; none under reduced motion.

FINISH: unreviewed and undocumented is unfinished; this build ends with the finish review, the verdict, DESIGN.md, and every shipping raster carrying its provenance
