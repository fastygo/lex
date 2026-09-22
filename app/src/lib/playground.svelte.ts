import { copy } from "$lib/copy";
import { layoutSlice } from "$lib/layout";
import {
  buildDraft,
  formatDraft,
  projectFile,
  readDraft,
  readProjectFile,
  selectedFromDraft,
  type ReasonCode,
} from "$lib/request";
import { sliceById, type SliceDef } from "$lib/slices";

const dwellMs = 2000;
const quietMs = 1500;
const burstMs = 8000;
const burstWindowMs = 2000;
const burstCount = 6;
const runLockMs = 3500;
const sessionMs = 1200;

export function createPlayground(initialSliceId: string) {
  let sliceId = $state(sliceById(initialSliceId).id);
  let requestText = $state("");
  let revision = $state(0);
  let trap = $state("");
  let status = $state("");
  let now = $state(Date.now());
  let assembledAt = $state(0);
  let quietUntil = 0;
  let holdUntil = 0;
  let runReadyAt = 0;
  let mountedAt = Date.now();
  const clicks: number[] = [];

  function slice(): SliceDef {
    return sliceById(sliceId);
  }

  function writeFresh(id: string) {
    const next = sliceById(id);
    sliceId = next.id;
    requestText = formatDraft(buildDraft(next, next.state, []));
    revision += 1;
    status = "";
    assembledAt = 0;
  }

  writeFresh(initialSliceId);
  revision = 0;

  function structural(): ReasonCode[] {
    return readDraft(requestText, slice()).codes;
  }

  function syncAssembly(codes: ReasonCode[]) {
    const ready = codes.length === 0 && trap.trim() === "";
    if (ready && assembledAt === 0) assembledAt = Date.now();
    if (!ready) assembledAt = 0;
  }

  function noteClick() {
    const t = Date.now();
    while (clicks.length > 0 && t - clicks[0] > burstWindowMs) clicks.shift();
    clicks.push(t);
    if (clicks.length >= burstCount) holdUntil = t + burstMs;
    quietUntil = t + quietMs;
  }

  function noteEdit() {
    quietUntil = Date.now() + quietMs;
  }

  return {
    get sliceId() {
      return sliceId;
    },
    get slice() {
      return slice();
    },
    get requestText() {
      return requestText;
    },
    get revision() {
      return revision;
    },
    get trap() {
      return trap;
    },
    set trap(value: string) {
      trap = value;
      syncAssembly(structural());
    },
    get status() {
      return status;
    },
    get selected() {
      return selectedFromDraft(slice(), readDraft(requestText, slice()).draft);
    },
    get graph() {
      return layoutSlice(slice(), this.selected);
    },
    reasons(): string[] {
      const codes = new Set<ReasonCode>(structural());
      if (trap.trim() !== "") codes.add("trap");
      const t = now;
      if (codes.size === 0) {
        if (t < mountedAt + sessionMs) codes.add("session");
        if (assembledAt !== 0 && t < assembledAt + dwellMs) codes.add("dwell");
        if (t < quietUntil) codes.add("quiet");
        if (t < holdUntil) codes.add("burst");
        if (t < runReadyAt) codes.add("lock");
      }
      return [...codes].map((code) => copy.playground.reasons[code]);
    },
    get canRun() {
      return this.reasons().length === 0;
    },
    tick() {
      now = Date.now();
      const ready = structural().length === 0 && trap.trim() === "";
      if (ready && assembledAt === 0) assembledAt = now;
      if (!ready) assembledAt = 0;
    },
    load(id: string) {
      if (sliceId === sliceById(id).id && requestText) return;
      writeFresh(id);
    },
    forceLoad(id: string) {
      writeFresh(id);
    },
    toggle(id: string) {
      noteClick();
      const current = new Set(this.selected);
      if (current.has(id)) current.delete(id);
      else current.add(id);
      const parsed = readDraft(requestText, slice());
      const state = parsed.draft?.state ?? slice().state;
      const next = buildDraft(slice(), state, [...current]);
      if (parsed.draft) {
        for (const [key, question] of Object.entries(parsed.draft.question_set.questions)) {
          if (next.question_set.questions[key]) next.question_set.questions[key] = question;
        }
      }
      requestText = formatDraft(next);
      revision += 1;
      status = "";
      syncAssembly(readDraft(requestText, slice()).codes);
    },
    editRequest(text: string) {
      if (text === requestText) return;
      noteEdit();
      requestText = text;
      status = "";
      syncAssembly(structural());
    },
    run() {
      if (!this.canRun) return;
      const parsed = readDraft(requestText, slice());
      if (!parsed.draft || parsed.codes.length > 0) return;
      runReadyAt = Date.now() + runLockMs;
      status = copy.playground.composed;
      requestText = formatDraft(parsed.draft);
      revision += 1;
    },
    exportProject() {
      return projectFile(sliceId, requestText);
    },
    importProject(value: unknown): boolean {
      const file = readProjectFile(value);
      if (!file) return false;
      sliceId = file.sliceId;
      requestText = file.requestText;
      revision += 1;
      status = "";
      syncAssembly(structural());
      return true;
    },
  };
}

export type Playground = ReturnType<typeof createPlayground>;
