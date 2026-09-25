import type { Entry } from "../../bindings/nova/services/models";
import { app, parentOf, type MenuItem } from "./store.svelte";

/** The menu for selected items: the context menu, and ⋮ on phones. */
export function itemMenu(sel: Entry[], isSearch: boolean): MenuItem[] {
  const one = sel.length === 1 ? sel[0] : null;
  const starred = sel.every((x) => app.isStarred(x.path));
  if (app.inTrash) {
    return [
      { label: "Restore From Trash", run: () => app.restore(sel) },
      { sep: true },
      { label: "Delete Permanently", accel: "Delete", run: () => app.deleteForever(sel) },
      { sep: true },
      { label: "Properties", accel: "Ctrl+I", disabled: !one, run: () => one && (app.modal = { kind: "properties", entry: one }) },
    ];
  }
  return [
    ...(one?.isDir
      ? [
          { label: "Open", accel: "Return", run: () => app.open(one) },
          ...(app.mobile ? [] : [{ label: "Open in New Window", run: () => app.newWindow(one.path) }]),
        ]
      : app.mobile
        ? []
        : [{ label: "Open With Default Application", accel: "Return", run: () => app.openSelection() }]),
    ...(one && !one.isDir ? [{ label: "Preview", accel: "Space", run: () => app.preview(one) }] : []),
    ...(isSearch && one ? [{ label: "Open Item Location", run: () => app.navigate(parentOf(one.path), true, [one.path]) }] : []),
    { sep: true },
    {
      row: [
        { label: "Cut", icon: "edit-cut", accel: "Ctrl+X", run: () => app.copy(true) },
        { label: "Copy", icon: "edit-copy", accel: "Ctrl+C", run: () => app.copy(false) },
        ...(one?.isDir ? [{ label: "Paste Into Folder", icon: "edit-paste", disabled: !app.canPaste, run: () => app.paste(one.path) }] : []),
        { label: "Rename", icon: "document-edit", accel: "F2", disabled: !one, run: () => app.startRename(one!) },
        starred
          ? { label: "Unstar", icon: "starred", run: () => app.toggleStar(sel) }
          : { label: "Star", icon: "non-starred", run: () => app.toggleStar(sel) },
        { label: "Move to Trash", icon: "user-trash", accel: "Delete", run: () => app.trash(sel) },
      ],
    },
    { sep: true },
    { label: app.mobile ? "Download" : "Download…", run: () => app.download(sel) },
    ...(one
      ? [
          { label: "Share…", run: () => app.openShare(one) },
          { label: one.public ? "Copy Link" : "Copy Public Link", run: () => app.copyLink(one) },
          ...(one.public ? [{ label: "Stop Sharing Link", run: () => app.stopSharing(one) }] : []),
        ]
      : []),
    { sep: true },
    ...(one?.isDir
      ? [{ label: app.isBookmarked(one.path) ? "Remove from Bookmarks" : "Add to Bookmarks", run: () => app.toggleBookmark(one.path) }]
      : []),
    { label: "Copy Location", disabled: !one, run: () => one && app.copyPath(one) },
    { label: "Properties", accel: "Ctrl+I", disabled: !one, run: () => one && (app.modal = { kind: "properties", entry: one }) },
  ];
}
