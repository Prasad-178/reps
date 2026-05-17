import { clsx, type ClassValue } from "clsx";
import { twMerge } from "tailwind-merge";

export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs));
}

export function formatRelative(ts: number | Date | string | null | undefined): string {
  if (!ts) return "—";
  const d = ts instanceof Date ? ts : new Date(typeof ts === "number" ? ts * 1000 : ts);
  const diff = (Date.now() - d.getTime()) / 1000;
  if (diff < 60) return `${Math.floor(diff)}s ago`;
  if (diff < 3600) return `${Math.floor(diff / 60)}m ago`;
  if (diff < 86400) return `${Math.floor(diff / 3600)}h ago`;
  if (diff < 30 * 86400) return `${Math.floor(diff / 86400)}d ago`;
  return d.toLocaleDateString();
}

export function shortId(id: string | null | undefined, n = 8): string {
  if (!id) return "—";
  return id.slice(0, n);
}
