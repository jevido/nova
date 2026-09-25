import raw from "../../../CHANGELOG.md?raw";

// CHANGELOG.md, parsed for the What's New dialog. See the top of that file
// for the format.

export type ChangeItem = { icon: string; title: string; text: string };
export type ChangeSection = { kind: string; items: ChangeItem[] };
export type Release = { version: string; date: string; summary: string; sections: ChangeSection[] };

function parse(md: string): Release[] {
  const out: Release[] = [];
  let rel: Release | null = null;
  let sec: ChangeSection | null = null;
  for (const line of md.split("\n")) {
    const h2 = /^##\s+v?(\S+)(?:\s+[—–-]\s+(.+))?\s*$/.exec(line);
    if (h2) {
      rel = { version: h2[1], date: h2[2] ?? "", summary: "", sections: [] };
      sec = null;
      out.push(rel);
      continue;
    }
    if (!rel) continue;
    const h3 = /^###\s+(.+?)\s*$/.exec(line);
    if (h3) {
      sec = { kind: h3[1], items: [] };
      rel.sections.push(sec);
      continue;
    }
    const li = /^-\s+(?:\[([\w-]+)\]\s+)?(?:\*\*(.+?)\*\*\s*(?:[—–-]\s*)?)?(.*)$/.exec(line);
    if (li && sec) {
      sec.items.push({ icon: li[1] ?? "", title: li[2] ?? "", text: li[3].trim() });
    } else if (li && rel.sections.length === 0) {
      // A list before any section still belongs somewhere.
      sec = { kind: "New", items: [] };
      rel.sections.push(sec);
      sec.items.push({ icon: li[1] ?? "", title: li[2] ?? "", text: li[3].trim() });
    } else if (!sec && line.trim()) {
      rel.summary = (rel.summary + " " + line.trim()).trim();
    }
  }
  return out;
}

export const releases: Release[] = parse(raw);

function parts(v: string): number[] {
  const [core, pre] = v.replace(/^v/, "").split("-");
  const n = core.split(".").map((x) => Number(x) || 0);
  while (n.length < 3) n.push(0);
  return [...n, pre ? -1 : 0];
}

/** a < b, as versions ("0.10.0" > "0.9.1", "1.0.0-rc.1" < "1.0.0"). */
export function older(a: string, b: string): boolean {
  const x = parts(a);
  const y = parts(b);
  for (let i = 0; i < x.length; i++) if (x[i] !== y[i]) return x[i] < y[i];
  return false;
}

/**
 * The releases to show after updating from `seen` to `current`: everything
 * newer than `seen` up to `current`, or only the newest when nothing was seen
 * yet (a fresh install).
 */
export function unseen(seen: string, current: string): Release[] {
  const upTo = releases.filter((r) => !older(current, r.version));
  if (!seen) return upTo.slice(0, 1);
  return upTo.filter((r) => older(seen, r.version));
}

export function formatDate(d: string): string {
  const t = Date.parse(d);
  return Number.isNaN(t) ? d : new Date(t).toLocaleDateString(undefined, { day: "numeric", month: "long", year: "numeric" });
}
