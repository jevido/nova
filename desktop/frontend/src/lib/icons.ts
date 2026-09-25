import type { Entry } from "../../bindings/nova/services/models";
import { extOf } from "./format";

/** URL of a named icon from the system theme, with fallbacks. */
export function iconUrl(...names: string[]): string {
  return `/nova/icon/${names.map(encodeURIComponent).join(",")}`;
}

const byExt: Record<string, string> = {
  pdf: "x-office-document",
  doc: "x-office-document",
  docx: "x-office-document",
  odt: "x-office-document",
  rtf: "x-office-document",
  xls: "x-office-spreadsheet",
  xlsx: "x-office-spreadsheet",
  ods: "x-office-spreadsheet",
  csv: "x-office-spreadsheet",
  ppt: "x-office-presentation",
  pptx: "x-office-presentation",
  odp: "x-office-presentation",
  zip: "package-x-generic",
  gz: "package-x-generic",
  tgz: "package-x-generic",
  xz: "package-x-generic",
  zst: "package-x-generic",
  zstd: "package-x-generic",
  bz2: "package-x-generic",
  "7z": "package-x-generic",
  rar: "package-x-generic",
  tar: "package-x-generic",
  deb: "package-x-generic",
  rpm: "package-x-generic",
  iso: "media-optical",
  html: "text-html",
  htm: "text-html",
  sh: "text-x-script",
  py: "text-x-script",
  js: "text-x-script",
  ts: "text-x-script",
  go: "text-x-script",
  rs: "text-x-script",
  c: "text-x-script",
  json: "text-x-script",
  exe: "application-x-executable",
  appimage: "application-x-executable",
  ttf: "font-x-generic",
  otf: "font-x-generic",
  woff: "font-x-generic",
  woff2: "font-x-generic",
};

/** Icon names for an entry, most specific first (freedesktop naming). */
export function iconNames(e: Pick<Entry, "isDir" | "mime" | "name" | "path">): string[] {
  if (e.isDir) {
    if (e.path === "/me") return ["user-home", "folder"];
    if (e.path === "/me/.Trash") return ["user-trash", "folder"];
    return ["folder"];
  }
  const names: string[] = [];
  const mime = (e.mime || "").split(";")[0].trim();
  if (mime) {
    names.push(mime.replace("/", "-"));
    const [top] = mime.split("/");
    if (["image", "video", "audio", "text", "font"].includes(top)) names.push(`${top}-x-generic`);
  }
  const ext = byExt[extOf(e.name)];
  if (ext) names.push(ext);
  names.push("text-x-generic", "application-x-generic");
  return [...new Set(names)];
}

export function fileIconUrl(e: Pick<Entry, "isDir" | "mime" | "name" | "path">): string {
  return iconUrl(...iconNames(e));
}

export function isImage(e: Entry): boolean {
  return !e.isDir && /^image\/(png|jpe?g|gif|webp|bmp|avif|svg\+xml|x-icon|heic|heif)/.test(e.mime);
}
export function isVideo(e: Entry): boolean {
  return !e.isDir && e.mime.startsWith("video/");
}
export function isAudio(e: Entry): boolean {
  return !e.isDir && e.mime.startsWith("audio/");
}
export function isText(e: Entry): boolean {
  if (e.isDir) return false;
  if (e.mime.startsWith("text/")) return true;
  return /^application\/(json|xml|javascript|x-sh|toml|yaml|x-yaml)/.test(e.mime);
}
export function hasThumbnail(e: Entry): boolean {
  return isImage(e) || isVideo(e);
}

export function thumbUrl(e: Entry, size: number): string {
  const q = new URLSearchParams({ path: e.path, size: String(size), v: e.sha256 || e.modified });
  return `/nova/thumb?${q}`;
}

export function rawUrl(path: string): string {
  return `/nova/raw?${new URLSearchParams({ path })}`;
}
