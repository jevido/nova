// GLib's g_format_size uses SI (1 kB = 1000 bytes); Nautilus shows sizes that way.
export function formatSize(bytes: number): string {
  if (!Number.isFinite(bytes) || bytes < 0) return "—";
  if (bytes < 1000) return bytes === 1 ? "1 byte" : `${bytes} bytes`;
  const units = ["kB", "MB", "GB", "TB", "PB"];
  let v = bytes;
  let i = -1;
  do {
    v /= 1000;
    i++;
  } while (v >= 1000 && i < units.length - 1);
  return `${v.toFixed(1)} ${units[i]}`;
}

export function formatRate(bps: number): string {
  return `${formatSize(Math.round(bps))}/s`;
}

export function formatDuration(seconds: number): string {
  if (!Number.isFinite(seconds) || seconds <= 0) return "";
  if (seconds < 60) return `${Math.ceil(seconds)} s`;
  if (seconds < 3600) return `${Math.ceil(seconds / 60)} min`;
  return `${Math.floor(seconds / 3600)} h ${Math.ceil((seconds % 3600) / 60)} min`;
}

const time = new Intl.DateTimeFormat(undefined, { hour: "2-digit", minute: "2-digit" });
const dayMonth = new Intl.DateTimeFormat(undefined, { day: "numeric", month: "short" });
const full = new Intl.DateTimeFormat(undefined, { day: "numeric", month: "short", year: "numeric" });
const long = new Intl.DateTimeFormat(undefined, { dateStyle: "full", timeStyle: "short" });

function validDate(iso: string | null | undefined): Date | null {
  if (!iso) return null;
  const d = new Date(iso);
  if (isNaN(d.getTime()) || d.getFullYear() < 1971) return null;
  return d;
}

/** Nautilus-style relative date: "10:42", "Yesterday", "3 Jan", "3 Jan 2024". */
export function formatDate(iso: string | null | undefined): string {
  const d = validDate(iso);
  if (!d) return "—";
  const now = new Date();
  const startOfToday = new Date(now.getFullYear(), now.getMonth(), now.getDate()).getTime();
  const t = d.getTime();
  if (t >= startOfToday) return time.format(d);
  if (t >= startOfToday - 86400000) return "Yesterday";
  if (d.getFullYear() === now.getFullYear()) return dayMonth.format(d);
  return full.format(d);
}

export function formatDateLong(iso: string | null | undefined): string {
  const d = validDate(iso);
  return d ? long.format(d) : "Unknown";
}

export function pluralize(n: number, one: string, many: string): string {
  return `${n} ${n === 1 ? one : many}`;
}

export function extOf(name: string): string {
  const i = name.lastIndexOf(".");
  return i > 0 ? name.slice(i + 1).toLowerCase() : "";
}

/** Split name into stem/extension so rename can preselect only the stem. */
export function stemLength(name: string, isDir: boolean): number {
  if (isDir) return name.length;
  const i = name.lastIndexOf(".");
  return i > 0 ? i : name.length;
}
