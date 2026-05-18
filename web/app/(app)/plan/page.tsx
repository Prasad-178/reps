"use client";

import { PageHeader } from "@/components/app/shell";
import { Card, CardContent } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import { useApi } from "@/lib/useApi";
import { api } from "@/lib/api";
import { formatRelative } from "@/lib/utils";

export default function PlanPage() {
  const { data, loading } = useApi(() => api.latestPlan(), []);

  return (
    <>
      <PageHeader
        title="Study plan"
        subtitle={
          data
            ? `Last generated ${formatRelative(data.generated_at)} · ${data.window_days}-day window`
            : "Latest synthesized plan"
        }
      />
      <div className="p-6 sm:p-8 max-w-3xl space-y-4">
        {loading && <Skeleton className="h-[400px]" />}

        {!loading && !data && (
          <Card className="border-dashed">
            <CardContent className="p-6 text-center text-sm text-[var(--muted-foreground)]">
              No plan generated yet. Run{" "}
              <code className="font-mono text-xs px-1.5 py-0.5 rounded bg-[var(--muted)] text-[var(--foreground)]">
                reps plan --days 30
              </code>{" "}
              after a few drills.
            </CardContent>
          </Card>
        )}

        {data && (
          <Card>
            <CardContent className="p-6">
              <article className="prose prose-invert max-w-none prose-sm prose-headings:tracking-[-0.015em] prose-headings:font-semibold prose-h2:text-xl prose-h3:text-base prose-code:font-mono prose-code:text-xs prose-code:before:content-none prose-code:after:content-none prose-pre:bg-[var(--muted)] prose-a:text-[var(--primary)] prose-strong:text-foreground">
                <Markdown source={data.markdown} />
              </article>
            </CardContent>
          </Card>
        )}
      </div>
    </>
  );
}

// Naïve markdown renderer — keep it simple, plans are short.
function Markdown({ source }: { source: string }) {
  const lines = source.split("\n");
  const out: React.ReactElement[] = [];
  let para: string[] = [];
  let listBuf: string[] = [];

  function flushPara() {
    if (para.length) {
      out.push(
        <p key={out.length} className="text-sm leading-relaxed text-[var(--foreground)]">
          {inline(para.join(" "))}
        </p>
      );
      para = [];
    }
  }
  function flushList() {
    if (listBuf.length) {
      out.push(
        <ul key={out.length} className="list-disc pl-5 space-y-1 text-sm text-[var(--foreground)]">
          {listBuf.map((l, i) => (
            <li key={i}>{inline(l)}</li>
          ))}
        </ul>
      );
      listBuf = [];
    }
  }

  for (const raw of lines) {
    const line = raw.trimEnd();
    if (!line.trim()) {
      flushPara();
      flushList();
      continue;
    }
    if (line.startsWith("### ")) {
      flushPara();
      flushList();
      out.push(<h3 key={out.length} className="text-base font-semibold tracking-[-0.015em] mt-5 mb-1.5">{line.slice(4)}</h3>);
    } else if (line.startsWith("## ")) {
      flushPara();
      flushList();
      out.push(<h2 key={out.length} className="text-xl font-semibold tracking-[-0.02em] mt-6 mb-2">{line.slice(3)}</h2>);
    } else if (line.startsWith("# ")) {
      flushPara();
      flushList();
      out.push(<h1 key={out.length} className="text-2xl font-semibold tracking-[-0.02em] mt-2 mb-3">{line.slice(2)}</h1>);
    } else if (/^\s*[-*]\s+/.test(line)) {
      flushPara();
      listBuf.push(line.replace(/^\s*[-*]\s+/, ""));
    } else {
      flushList();
      para.push(line);
    }
  }
  flushPara();
  flushList();
  return <>{out}</>;
}

function inline(s: string): React.ReactNode {
  // bold **x**, code `x`
  const parts: React.ReactNode[] = [];
  const re = /(\*\*([^*]+)\*\*|`([^`]+)`)/g;
  let last = 0;
  let m: RegExpExecArray | null;
  while ((m = re.exec(s))) {
    if (m.index > last) parts.push(s.slice(last, m.index));
    if (m[2]) parts.push(<strong key={parts.length}>{m[2]}</strong>);
    else if (m[3]) parts.push(<code key={parts.length} className="font-mono text-xs px-1 py-0.5 rounded bg-[var(--muted)]">{m[3]}</code>);
    last = m.index + m[0].length;
  }
  if (last < s.length) parts.push(s.slice(last));
  return parts;
}
