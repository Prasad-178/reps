"use client";

import { useState } from "react";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
  DialogClose,
} from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { Input, Textarea } from "@/components/ui/input";
import { Badge } from "@/components/ui/badge";
import { api } from "@/lib/api";
import { toast } from "sonner";
import { Plus, Upload, Loader2 } from "lucide-react";

type Kind = "resume" | "github" | "portfolio" | "jd" | "linkedin" | "x" | "note";

const KIND_TITLE: Record<Kind, string> = {
  resume: "Add resume",
  github: "Add GitHub user",
  portfolio: "Add portfolio URL",
  jd: "Add job description",
  linkedin: "Add LinkedIn (paste)",
  x: "Add X handle (paste)",
  note: "Add markdown note",
};

const KIND_DESC: Record<Kind, string> = {
  resume:    "Upload a .pdf. Parsed locally via pdftotext, stored under ~/.reps/sources/.",
  github:    "Lists your non-fork non-archived repos and their READMEs via the gh CLI (needs `gh auth login`).",
  portfolio: "URL → crawled via Jina Reader (sitemap.xml + same-origin links, up to 12 pages). Path → walks the folder, reads every .md/.txt/.html/.rst inside.",
  jd:        "Fetched via Jina Reader (clean markdown). One LLM call then extracts company / role / must-haves / tech / culture.",
  linkedin:  "If PROXYCURL_API_KEY is set, the profile is fetched via Proxycurl. Otherwise paste the page content (LinkedIn blocks scrapers).",
  x:         "Same as LinkedIn — paste recent posts. We'll use them as profile signal.",
  note:      "Any markdown you want the agents to know about (style notes, side projects, prep notes).",
};

export function AddSourceDialog({
  kind,
  onAdded,
  trigger,
}: {
  kind: Kind;
  onAdded?: () => void;
  trigger?: React.ReactNode;
}) {
  const [open, setOpen] = useState(false);
  const [submitting, setSubmitting] = useState(false);

  // shared field state
  const [textValue, setTextValue] = useState(""); // url / user / ref / handle
  const [paste, setPaste] = useState("");
  const [file, setFile] = useState<File | null>(null);
  const [noteName, setNoteName] = useState("");

  function reset() {
    setTextValue(""); setPaste(""); setFile(null); setNoteName("");
  }

  async function onSubmit() {
    setSubmitting(true);
    try {
      let id = "";
      switch (kind) {
        case "github":
          if (!textValue.trim()) throw new Error("username required");
          ({ id } = await api.addGithub(textValue.trim()));
          break;
        case "portfolio":
          ({ id } = await api.addPortfolio(textValue.trim()));
          break;
        case "jd":
          ({ id } = await api.addJD(textValue.trim()));
          break;
        case "linkedin":
          if (!paste.trim()) throw new Error("paste content required");
          ({ id } = await api.addLinkedIn(textValue.trim() || "linkedin", paste));
          break;
        case "x":
          if (!paste.trim()) throw new Error("paste content required");
          ({ id } = await api.addX(textValue.trim() || "x", paste));
          break;
        case "note":
          if (!paste.trim()) throw new Error("note body required");
          ({ id } = await api.addNote(noteName.trim() || "note.md", paste));
          break;
        case "resume":
          if (!file) throw new Error("file required");
          ({ id } = await api.addResume(file));
          break;
      }
      toast.success(`added ${kind} · ${id.slice(0, 8)}`);
      onAdded?.();
      reset();
      setOpen(false);
    } catch (e) {
      toast.error(e instanceof Error ? e.message : "failed");
    } finally {
      setSubmitting(false);
    }
  }

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger asChild>
        {trigger ?? <Button size="sm" variant="outline"><Plus /> Add</Button>}
      </DialogTrigger>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{KIND_TITLE[kind]}</DialogTitle>
          <DialogDescription>{KIND_DESC[kind]}</DialogDescription>
        </DialogHeader>

        <div className="space-y-3">
          {kind === "resume" && (
            <FileInput onFile={setFile} file={file} />
          )}

          {(kind === "github" || kind === "portfolio" || kind === "jd") && (
            <Input
              autoFocus
              value={textValue}
              onChange={(e) => setTextValue(e.target.value)}
              placeholder={
                kind === "github"
                  ? "username (e.g. Prasad-178)"
                  : kind === "portfolio"
                    ? "https://your-portfolio.dev  or  /abs/path/to/folder"
                    : "https://jobs.example.com/staff-eng"
              }
            />
          )}

          {(kind === "linkedin" || kind === "x") && (
            <>
              <Input
                value={textValue}
                onChange={(e) => setTextValue(e.target.value)}
                placeholder={kind === "linkedin" ? "https://linkedin.com/in/you" : "@handle"}
              />
              <Textarea
                rows={10}
                value={paste}
                onChange={(e) => setPaste(e.target.value)}
                placeholder="Paste the page content (about, headline, experience, posts)…"
              />
              <p className="text-[10px] font-mono uppercase tracking-[0.06em] text-[var(--muted-foreground)]">
                <Badge variant="warning" className="mr-1">!</Badge>
                site blocks scrapers — paste is the canonical input
              </p>
            </>
          )}

          {kind === "note" && (
            <>
              <Input
                value={noteName}
                onChange={(e) => setNoteName(e.target.value)}
                placeholder="filename (optional, e.g. about-me.md)"
              />
              <Textarea
                rows={12}
                value={paste}
                onChange={(e) => setPaste(e.target.value)}
                placeholder="# About me\n\nFree-form markdown the agents should know about…"
              />
            </>
          )}
        </div>

        <div className="flex items-center justify-end gap-2 pt-4 border-t border-[var(--border)] mt-4">
          <DialogClose asChild>
            <Button variant="ghost" size="sm" disabled={submitting}>Cancel</Button>
          </DialogClose>
          <Button onClick={onSubmit} disabled={submitting} size="sm">
            {submitting && <Loader2 className="animate-spin" />}
            {submitting ? "Ingesting…" : "Add source"}
          </Button>
        </div>
      </DialogContent>
    </Dialog>
  );
}

function FileInput({
  onFile,
  file,
}: {
  onFile: (f: File | null) => void;
  file: File | null;
}) {
  return (
    <label className="flex flex-col items-center justify-center gap-2 py-8 px-4 rounded-lg border-2 border-dashed border-[var(--border)] hover:border-[var(--primary)] cursor-pointer transition-colors duration-150 [transition-timing-function:var(--ease-out)]">
      <Upload className="size-5 text-[var(--muted-foreground)]" />
      <span className="text-sm">
        {file ? <strong>{file.name}</strong> : "click to choose a PDF"}
      </span>
      {file && (
        <span className="text-[10px] font-mono text-[var(--muted-foreground)]">
          {(file.size / 1024).toFixed(1)} KB
        </span>
      )}
      <input
        type="file"
        accept="application/pdf,.pdf"
        className="hidden"
        onChange={(e) => onFile(e.target.files?.[0] ?? null)}
      />
    </label>
  );
}
