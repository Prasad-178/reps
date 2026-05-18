"use client";

import { useState } from "react";
import { PageHeader } from "@/components/app/shell";
import { Card, CardContent } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Skeleton } from "@/components/ui/skeleton";
import { Button } from "@/components/ui/button";
import { AddSourceDialog } from "@/components/app/add-source-dialog";
import { useApi } from "@/lib/useApi";
import { api } from "@/lib/api";
import { formatRelative } from "@/lib/utils";
import {
  FileText, Code2, Globe, MessageSquare, Briefcase, StickyNote,
  Trash2, RefreshCcw, Loader2, Plus,
} from "lucide-react";
import { toast } from "sonner";

type Kind = "resume" | "github" | "portfolio" | "jd" | "linkedin" | "x" | "note";

const KIND_ICON: Record<string, React.ComponentType<{ className?: string }>> = {
  resume_pdf: FileText,
  github: Code2,
  portfolio: Globe,
  linkedin: MessageSquare,
  x: MessageSquare,
  jd: Briefcase,
  note: StickyNote,
};

const KIND_LABEL: Record<string, string> = {
  resume_pdf: "Resume",
  github: "GitHub",
  portfolio: "Portfolio",
  linkedin: "LinkedIn",
  x: "X / Twitter",
  jd: "Job description",
  note: "Note",
};

const ADD_KINDS: { id: Kind; label: string }[] = [
  { id: "resume", label: "Resume" },
  { id: "github", label: "GitHub" },
  { id: "portfolio", label: "Portfolio" },
  { id: "jd", label: "JD" },
  { id: "linkedin", label: "LinkedIn" },
  { id: "x", label: "X" },
  { id: "note", label: "Note" },
];

export default function SourcesPage() {
  const { data, loading, error } = useApi(() => api.sources(), []);
  const [refreshTick, setRefreshTick] = useState(0);
  const sources = useApi(() => api.sources(), [refreshTick]);
  const list = sources.data ?? data ?? [];
  const refresh = () => setRefreshTick((t) => t + 1);

  const [rebuilding, setRebuilding] = useState(false);

  async function onDelete(id: string) {
    if (!confirm("Delete this source and all its chunks?")) return;
    try {
      await api.deleteSource(id);
      toast.success("removed");
      refresh();
    } catch (e) {
      toast.error(e instanceof Error ? e.message : "delete failed");
    }
  }

  async function onRebuild() {
    setRebuilding(true);
    try {
      await api.rebuildProfile();
      toast.success("rebuild started — refresh in a moment");
      // light poll
      const poll = setInterval(async () => {
        const s = await api.rebuildStatus();
        if (!s.running) {
          clearInterval(poll);
          setRebuilding(false);
          if (s.error) toast.error("rebuild failed: " + s.error);
          else toast.success("profile rebuilt");
          refresh();
        }
      }, 1500);
    } catch (e) {
      setRebuilding(false);
      toast.error(e instanceof Error ? e.message : "failed");
    }
  }

  return (
    <>
      <PageHeader
        title="Sources"
        subtitle={`${list.length} ingested`}
        action={
          <div className="flex items-center gap-2">
            <Button
              variant="outline"
              size="sm"
              onClick={onRebuild}
              disabled={rebuilding || list.length === 0}
            >
              {rebuilding ? <Loader2 className="animate-spin" /> : <RefreshCcw />}
              {rebuilding ? "Rebuilding…" : "Rebuild profile"}
            </Button>
          </div>
        }
      />
      <div className="p-6 sm:p-8 max-w-4xl space-y-4">
        <Card>
          <CardContent className="p-4 flex flex-wrap items-center gap-2">
            <span className="text-xs font-mono uppercase tracking-[0.08em] text-[var(--muted-foreground)] mr-2">
              + add source
            </span>
            {ADD_KINDS.map((k) => (
              <AddSourceDialog
                key={k.id}
                kind={k.id}
                onAdded={refresh}
                trigger={
                  <Button variant="ghost" size="sm">
                    <Plus /> {k.label}
                  </Button>
                }
              />
            ))}
          </CardContent>
        </Card>

        {(error || sources.error) && (
          <Card className="border-[var(--destructive)]/40">
            <CardContent className="p-4 text-sm text-[var(--destructive)]">
              Failed to load sources. Is{" "}
              <code className="font-mono text-xs px-1.5 py-0.5 rounded bg-[var(--muted)]">
                reps serve
              </code>{" "}
              running?
            </CardContent>
          </Card>
        )}

        {(loading && !list.length) && (
          <div className="space-y-2">
            {Array.from({ length: 3 }).map((_, i) => (
              <Skeleton key={i} className="h-16" />
            ))}
          </div>
        )}

        {!loading && list.length === 0 && (
          <Card>
            <CardContent className="p-6 text-center text-sm text-[var(--muted-foreground)]">
              No sources yet — pick one above to add your first.
            </CardContent>
          </Card>
        )}

        <ul className="space-y-2">
          {list.map((s) => {
            const Icon = KIND_ICON[s.kind] ?? FileText;
            return (
              <li key={s.id}>
                <Card>
                  <CardContent className="p-4 flex items-center gap-4">
                    <div className="grid place-items-center size-10 rounded-md bg-[var(--secondary)] text-[var(--foreground)] shrink-0">
                      <Icon className="size-4" />
                    </div>
                    <div className="min-w-0 flex-1">
                      <div className="flex items-center gap-2 flex-wrap">
                        <Badge variant="outline">{KIND_LABEL[s.kind] ?? s.kind}</Badge>
                        <span className="font-mono text-[10px] text-[var(--muted-foreground)]">
                          {s.id.slice(0, 8)}
                        </span>
                      </div>
                      <p className="text-sm font-medium mt-1 truncate">{s.ref}</p>
                      <p className="text-[11px] text-[var(--muted-foreground)] mt-0.5">
                        added {formatRelative(s.fetched_at)}
                      </p>
                    </div>
                    <Button
                      variant="ghost"
                      size="icon"
                      onClick={() => onDelete(s.id)}
                      aria-label="Delete source"
                      className="text-[var(--muted-foreground)] hover:text-[var(--destructive)]"
                    >
                      <Trash2 />
                    </Button>
                  </CardContent>
                </Card>
              </li>
            );
          })}
        </ul>
      </div>
    </>
  );
}
