"use client";

import { useEffect, useState } from "react";
import { PageHeader } from "@/components/app/shell";
import { Card, CardHeader, CardTitle, CardContent } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Switch } from "@/components/ui/switch";
import { Badge } from "@/components/ui/badge";
import { Skeleton } from "@/components/ui/skeleton";
import { Separator } from "@/components/ui/separator";
import { api, type ConfigPublic } from "@/lib/api";
import { toast } from "sonner";
import { Save, ShieldCheck, Loader2, KeyRound } from "lucide-react";

const PRESET_MODELS = [
  "google/gemini-2.0-flash-001",
  "anthropic/claude-3.5-haiku",
  "anthropic/claude-3.5-sonnet",
  "openai/gpt-4o-mini",
  "meta-llama/llama-3.3-70b-instruct:free",
];

const TTS_PROVIDERS = ["say", "openai", "elevenlabs"];

export default function SettingsPage() {
  const [cfg, setCfg] = useState<ConfigPublic | null>(null);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [newKey, setNewKey] = useState("");
  const [probing, setProbing] = useState(false);

  useEffect(() => {
    api.getConfig()
      .then(setCfg)
      .catch((e) => toast.error("load config: " + String(e)))
      .finally(() => setLoading(false));
  }, []);

  function update<K extends keyof ConfigPublic>(key: K, patch: Partial<ConfigPublic[K]>) {
    setCfg((c) => (c ? { ...c, [key]: { ...c[key], ...patch } } : c));
  }

  async function save() {
    if (!cfg) return;
    setSaving(true);
    try {
      const next = await api.patchConfig({
        llm: {
          model: cfg.llm.model,
          embed_model: cfg.llm.embed_model,
          judge_model: cfg.llm.judge_model,
          api_key: newKey || undefined,
        },
        drill: {
          default_qs: cfg.drill.default_qs,
          followup_max: cfg.drill.followup_max,
        },
        elo: {
          k_factor: cfg.elo.k_factor,
          start_rating: cfg.elo.start_rating,
        },
        voice: {
          tts_enabled: cfg.voice.tts_enabled,
          tts_provider: cfg.voice.tts_provider,
          tts_voice: cfg.voice.tts_voice,
          tts_model: cfg.voice.tts_model,
          tts_rate: cfg.voice.tts_rate,
        },
      });
      setCfg(next);
      setNewKey("");
      toast.success("settings saved");
    } catch (e) {
      toast.error(e instanceof Error ? e.message : "save failed");
    } finally {
      setSaving(false);
    }
  }

  async function probe() {
    if (!cfg) return;
    setProbing(true);
    try {
      const r = await api.probeModel(cfg.llm.model);
      if (r.ok) toast.success(`model ${cfg.llm.model} works`);
      else toast.error(`probe failed: ${r.error}`);
    } catch (e) {
      toast.error(e instanceof Error ? e.message : "probe failed");
    } finally {
      setProbing(false);
    }
  }

  return (
    <>
      <PageHeader
        title="Settings"
        subtitle={cfg ? cfg.paths.home : "Loading…"}
        action={
          <Button onClick={save} disabled={saving || !cfg}>
            {saving ? <Loader2 className="animate-spin" /> : <Save />}
            {saving ? "Saving…" : "Save"}
          </Button>
        }
      />
      <div className="p-6 sm:p-8 max-w-3xl space-y-5">
        {loading && <Skeleton className="h-[500px]" />}

        {cfg && (
          <>
            <SectionCard
              title="LLM"
              subtitle="OpenRouter is the only provider for now. Model choice affects the Planner, Interviewer, Coach, Analyst."
            >
              <Row label="API key">
                <div className="flex items-center gap-2 flex-1">
                  <Badge variant="outline" className="font-mono normal-case tracking-normal">
                    <KeyRound className="size-3 mr-1" />
                    {cfg.llm.api_key_mask || "(unset)"}
                  </Badge>
                  <Input
                    type="password"
                    placeholder="paste a new key to replace"
                    value={newKey}
                    onChange={(e) => setNewKey(e.target.value)}
                    className="flex-1"
                  />
                </div>
              </Row>

              <Row label="Primary model">
                <div className="flex items-center gap-2 flex-1">
                  <Input
                    list="presetmodels"
                    value={cfg.llm.model}
                    onChange={(e) => update("llm", { model: e.target.value })}
                    placeholder="provider/model"
                    className="flex-1"
                  />
                  <datalist id="presetmodels">
                    {PRESET_MODELS.map((m) => <option key={m} value={m} />)}
                  </datalist>
                  <Button variant="outline" size="sm" onClick={probe} disabled={probing}>
                    {probing ? <Loader2 className="animate-spin" /> : <ShieldCheck />}
                    Test
                  </Button>
                </div>
              </Row>

              <Row label="Embed model">
                <Input
                  value={cfg.llm.embed_model}
                  onChange={(e) => update("llm", { embed_model: e.target.value })}
                  className="flex-1"
                />
              </Row>

              <Row label="Judge model" hint="optional override; uses primary if blank">
                <Input
                  value={cfg.llm.judge_model}
                  onChange={(e) => update("llm", { judge_model: e.target.value })}
                  className="flex-1"
                />
              </Row>
            </SectionCard>

            <SectionCard title="Drill" subtitle="Defaults applied to every new drill session.">
              <Row label="Default questions per drill">
                <NumberInput
                  value={cfg.drill.default_qs}
                  min={1}
                  max={10}
                  onChange={(n) => update("drill", { default_qs: n })}
                />
              </Row>
              <Row label="Max follow-ups per Q" hint="hard cap; Interviewer asks fewer if the answer is already deep">
                <NumberInput
                  value={cfg.drill.followup_max}
                  min={0}
                  max={5}
                  onChange={(n) => update("drill", { followup_max: n })}
                />
              </Row>
            </SectionCard>

            <SectionCard title="ELO" subtitle="K-factor controls swing per drill (chess default 24; lower = slower).">
              <Row label="K-factor">
                <NumberInput
                  value={cfg.elo.k_factor}
                  min={4}
                  max={48}
                  onChange={(n) => update("elo", { k_factor: n })}
                />
              </Row>
              <Row label="Starting rating">
                <NumberInput
                  value={cfg.elo.start_rating}
                  min={800}
                  max={2000}
                  onChange={(n) => update("elo", { start_rating: n })}
                />
              </Row>
            </SectionCard>

            <SectionCard title="Voice (TTS)" subtitle="Read questions aloud during drills.">
              <Row label="Enable TTS">
                <Switch
                  checked={cfg.voice.tts_enabled}
                  onCheckedChange={(v) => update("voice", { tts_enabled: v })}
                />
              </Row>
              <Row label="Provider" hint="say = macOS native; others need API keys in env">
                <select
                  value={cfg.voice.tts_provider}
                  onChange={(e) => update("voice", { tts_provider: e.target.value })}
                  className="h-9 px-3 rounded-md border border-[var(--border)] bg-[var(--input)] text-sm flex-1"
                >
                  {TTS_PROVIDERS.map((p) => <option key={p} value={p}>{p}</option>)}
                </select>
              </Row>
              <Row label="Voice">
                <Input
                  value={cfg.voice.tts_voice}
                  onChange={(e) => update("voice", { tts_voice: e.target.value })}
                  className="flex-1"
                  placeholder={cfg.voice.tts_provider === "say" ? "Daniel / Karen / etc." : "voice id"}
                />
              </Row>
              {cfg.voice.tts_provider === "say" && (
                <Row label="Rate (wpm)">
                  <NumberInput
                    value={cfg.voice.tts_rate}
                    min={80}
                    max={400}
                    onChange={(n) => update("voice", { tts_rate: n })}
                  />
                </Row>
              )}
            </SectionCard>
          </>
        )}
      </div>
    </>
  );
}

function SectionCard({
  title,
  subtitle,
  children,
}: {
  title: string;
  subtitle?: string;
  children: React.ReactNode;
}) {
  return (
    <Card>
      <CardHeader>
        <CardTitle>{title}</CardTitle>
        {subtitle && <p className="text-xs text-[var(--muted-foreground)]">{subtitle}</p>}
      </CardHeader>
      <CardContent className="space-y-3">
        <Separator className="!mt-0" />
        {children}
      </CardContent>
    </Card>
  );
}

function Row({
  label,
  hint,
  children,
}: {
  label: string;
  hint?: string;
  children: React.ReactNode;
}) {
  return (
    <div className="grid grid-cols-1 sm:grid-cols-[180px_1fr] items-center gap-3 py-1.5">
      <div>
        <div className="text-sm">{label}</div>
        {hint && <div className="text-[11px] text-[var(--muted-foreground)]">{hint}</div>}
      </div>
      <div className="flex items-center gap-2">{children}</div>
    </div>
  );
}

function NumberInput({
  value,
  onChange,
  min,
  max,
}: {
  value: number;
  onChange: (v: number) => void;
  min?: number;
  max?: number;
}) {
  return (
    <Input
      type="number"
      value={value}
      min={min}
      max={max}
      onChange={(e) => {
        const n = parseInt(e.target.value, 10);
        if (!Number.isNaN(n)) onChange(n);
      }}
      className="w-28 font-mono"
    />
  );
}
