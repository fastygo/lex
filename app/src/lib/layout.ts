import type { Edge, Node } from "@xyflow/svelte";
import { copy } from "$lib/copy";
import type { SliceDef } from "$lib/slices";

export type StepData = {
  title: string;
  detail: string;
  lane: "input" | "control" | "mechanism" | "output";
  questionId?: string;
  included?: boolean;
  ontoggle?: (id: string) => void;
  includeLabel: string;
  includedLabel: string;
};

const gap = 148;

export function layoutSlice(slice: SliceDef, selected: string[]): { nodes: Node<StepData>[]; edges: Edge[] } {
  const nodes: Node<StepData>[] = [];
  const edges: Edge[] = [];
  let row = 0;

  const push = (id: string, data: StepData) => {
    nodes.push({
      id,
      type: "step",
      position: { x: 48, y: row * gap },
      data,
      draggable: false,
      connectable: false,
    });
    row += 1;
  };

  const labels = {
    includeLabel: copy.playground.include,
    includedLabel: copy.playground.included,
  };

  push("state", {
    title: "State",
    detail: "Input. Caller-owned JSON.",
    lane: "input",
    included: true,
    ...labels,
  });

  for (const question of slice.questions) {
    const included = selected.includes(question.id);
    push(question.id, {
      title: question.id,
      detail: `${question.type}. ${question.instructions}`,
      lane: "control",
      questionId: question.id,
      included,
      ...labels,
    });
    if (included) {
      edges.push({ id: `state-${question.id}`, source: "state", target: question.id, type: "smoothstep" });
    }
  }

  if (slice.context) {
    const included = selected.includes("context");
    push("context", {
      title: "Context binding",
      detail: "Frozen pack, snapshot, and pack request.",
      lane: "control",
      questionId: "context",
      included,
      ...labels,
    });
    if (included) {
      edges.push({ id: "state-context", source: "state", target: "context", type: "smoothstep" });
    }
  }

  push("decision", {
    title: "Decision",
    detail: "Mechanism. One POST /v1/decisions when Run opens.",
    lane: "mechanism",
    included: true,
    ...labels,
  });

  const controls = nodes.filter((node) => node.data.lane === "control" && node.data.included);
  for (const node of controls) {
    edges.push({ id: `${node.id}-decision`, source: node.id, target: "decision", type: "smoothstep" });
  }

  const required = slice.questions.length + (slice.context ? 1 : 0);
  const complete = controls.length === required;
  push("output", {
    title: "DecisionSet",
    detail: complete ? "Output. Answers, report, and replay bundle." : "Output waits until every block is included.",
    lane: "output",
    included: complete,
    ...labels,
  });
  if (complete) {
    edges.push({ id: "decision-output", source: "decision", target: "output", type: "smoothstep" });
  }

  return { nodes, edges };
}
