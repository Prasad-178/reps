"use client";

import { PageHeader } from "@/components/app/shell";
import { Card, CardHeader, CardTitle, CardContent } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Skeleton } from "@/components/ui/skeleton";
import { useApi } from "@/lib/useApi";
import { api } from "@/lib/api";

export default function JDsPage() {
  const { data, loading } = useApi(() => api.jds(), []);

  return (
    <>
      <PageHeader
        title="Job descriptions"
        subtitle={data ? `${data.length} target roles` : "Loading…"}
      />
      <div className="p-6 sm:p-8 max-w-4xl space-y-3">
        {loading && (
          <div className="space-y-2">
            {Array.from({ length: 2 }).map((_, i) => (
              <Skeleton key={i} className="h-40" />
            ))}
          </div>
        )}
        {data?.length === 0 && (
          <Card>
            <CardContent className="p-6 text-center text-sm text-[var(--muted-foreground)]">
              No JDs ingested. Run{" "}
              <code className="font-mono text-xs px-1.5 py-0.5 rounded bg-[var(--muted)] text-[var(--foreground)]">
                reps add jd &lt;url&gt;
              </code>
              .
            </CardContent>
          </Card>
        )}

        {data?.map((j) => (
          <Card key={j.id}>
            <CardHeader className="flex-row items-start justify-between">
              <div>
                <CardTitle className="text-lg">{j.company || "—"}</CardTitle>
                <p className="text-sm text-[var(--muted-foreground)] mt-1">
                  {j.role || "—"}
                  {j.card?.location && ` · ${j.card.location}`}
                  {j.card?.level && (
                    <span className="font-mono uppercase tracking-[0.06em] text-[10px] ml-2">
                      {j.card.level}
                    </span>
                  )}
                </p>
              </div>
              <span className="font-mono text-[10px] text-[var(--muted-foreground)]">
                {j.id.slice(0, 8)}
              </span>
            </CardHeader>
            <CardContent className="space-y-4">
              {j.card?.summary && (
                <p className="text-sm text-[var(--foreground)] leading-relaxed">
                  {j.card.summary}
                </p>
              )}
              <Group label="Must-haves" items={j.card?.must_haves} variant="primary" />
              <Group label="Nice-to-haves" items={j.card?.nice_to_haves} variant="default" />
              <Group label="Tech" items={j.card?.tech} variant="outline" mono />
              <Group label="Culture" items={j.card?.culture} variant="default" />
            </CardContent>
          </Card>
        ))}
      </div>
    </>
  );
}

function Group({
  label,
  items,
  variant,
  mono = false,
}: {
  label: string;
  items?: string[];
  variant: "primary" | "default" | "outline";
  mono?: boolean;
}) {
  if (!items?.length) return null;
  return (
    <div>
      <p className="text-xs font-mono uppercase tracking-[0.08em] text-[var(--muted-foreground)] mb-2">
        {label}
      </p>
      <div className="flex flex-wrap gap-1.5">
        {items.map((s, i) => (
          <Badge key={i} variant={variant} className={mono ? "font-mono normal-case tracking-normal text-[11px]" : ""}>
            {s}
          </Badge>
        ))}
      </div>
    </div>
  );
}
