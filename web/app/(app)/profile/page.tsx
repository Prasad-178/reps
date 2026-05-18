"use client";

import { PageHeader } from "@/components/app/shell";
import { Card, CardContent } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import { useApi } from "@/lib/useApi";
import { api } from "@/lib/api";
import { formatRelative } from "@/lib/utils";

export default function ProfilePage() {
  const { data, loading } = useApi(() => api.profile(), []);

  return (
    <>
      <PageHeader
        title="Profile"
        subtitle={
          data?.built_at
            ? `Built ${formatRelative(data.built_at)} · ${data.model_used || "—"}`
            : "Synthesized candidate profile"
        }
      />
      <div className="p-6 sm:p-8 max-w-3xl space-y-4">
        {loading && <Skeleton className="h-[400px]" />}

        {!loading && !data?.markdown && (
          <Card className="border-dashed">
            <CardContent className="p-6 text-center text-sm text-[var(--muted-foreground)]">
              No profile yet. Run{" "}
              <code className="font-mono text-xs px-1.5 py-0.5 rounded bg-[var(--muted)] text-[var(--foreground)]">
                reps profile --rebuild
              </code>{" "}
              after adding sources.
            </CardContent>
          </Card>
        )}

        {data?.markdown && (
          <Card>
            <CardContent className="p-6">
              <pre className="whitespace-pre-wrap text-sm leading-relaxed font-sans text-[var(--foreground)]">
                {data.markdown}
              </pre>
            </CardContent>
          </Card>
        )}
      </div>
    </>
  );
}
