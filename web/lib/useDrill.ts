"use client";

import { useCallback, useEffect, useRef, useState } from "react";
import { api } from "@/lib/api";

export type DrillEvent =
  | { type: "hello"; qs: number }
  | { type: "session"; sessionId: string }
  | { type: "question:start"; ord: number; total: number }
  | { type: "planner:thinking" }
  | { type: "planner:decision"; decision: PlannerDecision }
  | { type: "rag:retrieve" }
  | { type: "interviewer:thinking" }
  | { type: "interviewer:opening"; questionId: string; text: string; context: ContextRef[] }
  | { type: "interviewer:token"; delta: string }
  | { type: "interviewer:deciding"; followupsRemaining: number }
  | { type: "interviewer:followup"; index: number; total: number; text: string }
  | { type: "interviewer:done_with_question" }
  | { type: "judge:grading" }
  | { type: "judge:verdict"; verdict: Verdict }
  | { type: "judge:error"; message: string }
  | { type: "elo:update"; category: string; before: number; after: number; delta: number }
  | { type: "question:end"; ord: number }
  | { type: "error"; message: string }
  | { type: "done" };

export type PlannerDecision = {
  category: string;
  target_topic: string;
  target_difficulty: number;
  why: string;
  jd_id?: string;
};

export type ContextRef = { kind: string; ref: string; chunk_id: string; distance: number };

export type Verdict = {
  rating: number;
  strengths: string[];
  missed: string[];
  better_answer_sketch: string;
  reading: { topic: string; why: string; optional_url?: string }[];
  topic_tags: string[];
};

export type DrillState = {
  status: "idle" | "running" | "awaiting-answer" | "done" | "error";
  events: DrillEvent[];
  sessionId?: string;
  currentQuestionId?: string;
  currentQuestionOrd?: number;
  totalQuestions?: number;
  currentText?: string;
  followupCount: number;
  streamingText?: string; // grows as interviewer:token events arrive
  streaming: boolean;
  decision?: PlannerDecision;
  verdict?: Verdict;
  eloUpdate?: { category: string; before: number; after: number; delta: number };
  error?: string;
};

export type DrillOpts = {
  qs?: number;
  category?: string;
  topic?: string;
  jd?: string;
  difficulty?: number;
};

export function useDrill() {
  const [state, setState] = useState<DrillState>({
    status: "idle",
    events: [],
    followupCount: 0,
    streaming: false,
  });
  const sourceRef = useRef<EventSource | null>(null);

  const start = useCallback((opts: DrillOpts) => {
    sourceRef.current?.close();
    setState({ status: "running", events: [], followupCount: 0, streaming: false });

    const url = api.drillURL(opts);
    const es = new EventSource(url);
    sourceRef.current = es;

    function push(ev: DrillEvent) {
      setState((s) => {
        const next: DrillState = { ...s, events: [...s.events, ev] };

        switch (ev.type) {
          case "session":
            next.sessionId = ev.sessionId;
            break;
          case "question:start":
            next.currentQuestionOrd = ev.ord;
            next.totalQuestions = ev.total;
            next.followupCount = 0;
            next.verdict = undefined;
            next.eloUpdate = undefined;
            next.currentText = undefined;
            next.streamingText = "";
            next.streaming = false;
            break;
          case "planner:decision":
            next.decision = ev.decision;
            break;
          case "interviewer:token":
            next.streaming = true;
            next.streamingText = (next.streamingText ?? "") + ev.delta;
            break;
          case "interviewer:opening":
            next.currentQuestionId = ev.questionId;
            next.currentText = ev.text;
            next.streaming = false;
            next.streamingText = "";
            next.status = "awaiting-answer";
            break;
          case "interviewer:followup":
            next.currentText = ev.text;
            next.followupCount = ev.index;
            next.status = "awaiting-answer";
            break;
          case "judge:verdict":
            next.verdict = ev.verdict;
            break;
          case "elo:update":
            next.eloUpdate = {
              category: ev.category,
              before: ev.before,
              after: ev.after,
              delta: ev.delta,
            };
            break;
          case "error":
            next.status = "error";
            next.error = ev.message;
            break;
          case "done":
            next.status = "done";
            break;
        }
        return next;
      });
    }

    const wireEvent = (name: DrillEvent["type"]) => {
      es.addEventListener(name, (e) => {
        const data = JSON.parse((e as MessageEvent).data || "{}");
        // normalize key names (snake_case → camelCase where needed)
        let ev: DrillEvent;
        switch (name) {
          case "session":
            ev = { type: "session", sessionId: data.session_id };
            break;
          case "interviewer:opening":
            ev = { type: "interviewer:opening", questionId: data.question_id, text: data.text, context: data.context || [] };
            break;
          case "interviewer:token":
            ev = { type: "interviewer:token", delta: data.delta || "" };
            break;
          case "interviewer:deciding":
            ev = { type: "interviewer:deciding", followupsRemaining: data.followups_remaining };
            break;
          case "planner:decision":
            ev = { type: "planner:decision", decision: data };
            break;
          case "judge:verdict":
            ev = { type: "judge:verdict", verdict: data };
            break;
          default:
            ev = { type: name, ...data } as DrillEvent;
        }
        push(ev);
      });
    };

    [
      "hello", "session", "question:start",
      "planner:thinking", "planner:decision", "rag:retrieve",
      "interviewer:thinking", "interviewer:token", "interviewer:opening", "interviewer:deciding",
      "interviewer:followup", "interviewer:done_with_question",
      "judge:grading", "judge:verdict", "judge:error",
      "elo:update", "question:end", "error", "done",
    ].forEach((n) => wireEvent(n as DrillEvent["type"]));

    es.onerror = () => {
      setState((s) =>
        s.status === "done" ? s : { ...s, status: "error", error: "connection closed" }
      );
      es.close();
    };
  }, []);

  const submit = useCallback(
    async (text: string) => {
      if (!state.sessionId || !state.currentQuestionId) return;
      setState((s) => ({ ...s, status: "running" }));
      await api.submitAnswer(state.sessionId, state.currentQuestionId, text);
    },
    [state.sessionId, state.currentQuestionId]
  );

  const endQuestion = useCallback(async () => {
    if (!state.sessionId || !state.currentQuestionId) return;
    await api.endQuestion(state.sessionId, state.currentQuestionId);
  }, [state.sessionId, state.currentQuestionId]);

  const reset = useCallback(() => {
    sourceRef.current?.close();
    setState({ status: "idle", events: [], followupCount: 0, streaming: false });
  }, []);

  useEffect(() => {
    return () => {
      sourceRef.current?.close();
    };
  }, []);

  return { state, start, submit, endQuestion, reset };
}
