"use client";

import { PageHeader } from "@/components/app/shell";
import { Card, CardContent } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Skeleton } from "@/components/ui/skeleton";
import { useApi } from "@/lib/useApi";
import { api } from "@/lib/api";
import { formatRelative } from "@/lib/utils";
import { FileText, Code2, Globe, MessageSquare, Briefcase, StickyNote } from "lucide-react";

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

export default function SourcesPage() {
  const { data, loading, error } = useApi(() => api.sources(), []);

  return (
    <>
      <PageHeader
        title="Sources"
        subtitle={data ? `${data.length} ingested` : "Loading…"}
      />
      <div className="p-6 sm:p-8 max-w-4xl space-y-4">
        <Card className="border-dashed">
          <CardContent className="p-4 text-sm text-[var(--muted-foreground)]">
            Sources are added via the CLI:{" "}
            <code className="font-mono text-xs px-1.5 py-0.5 rounded bg-[var(--muted)] text-[var(--foreground)]">
              reps add resume / github / portfolio / jd / linkedin / x / note
            </code>
            . Then{" "}
            <code className="font-mono text-xs px-1.5 py-0.5 rounded bg-[var(--muted)] text-[var(--foreground)]">
              reps profile --rebuild
            </code>{" "}
            to chunk + embed.
          </CardContent>
        </Card>

        {error && (
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

        {loading && (
          <div className="space-y-2">
            {Array.from({ length: 3 }).map((_, i) => (
              <Skeleton key={i} className="h-16" />
            ))}
          </div>
        )}

        {data?.length === 0 && (
          <Card>
            <CardContent className="p-6 text-center text-sm text-[var(--muted-foreground)]">
              No sources yet. Run{" "}
              <code className="font-mono text-xs px-1.5 py-0.5 rounded bg-[var(--muted)] text-[var(--foreground)]">
                reps init
              </code>{" "}
              to start.
            </CardContent>
          </Card>
        )}

        <ul className="space-y-2">
          {data?.map((s) => {
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
