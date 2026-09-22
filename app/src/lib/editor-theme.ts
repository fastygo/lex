import { HighlightStyle, syntaxHighlighting } from "@codemirror/language";
import { EditorView } from "@codemirror/view";
import { tags as t } from "@lezer/highlight";

/** CodeMirror chrome. Syntax colors are the --cm-* tokens in theme.css. */
export const workspaceEditorTheme = EditorView.theme(
  {
    "&": {
      height: "100%",
      backgroundColor: "var(--cm-editor-bg)",
      color: "var(--cm-editor-fg)",
    },
    "&.cm-focused": {
      outline: "none",
    },
    ".cm-scroller": {
      fontFamily: "var(--font-mono)",
      lineHeight: "1.45",
      overflow: "auto",
    },
    ".cm-content": {
      caretColor: "var(--primary)",
      padding: "0.5rem 0",
    },
    ".cm-cursor, .cm-dropCursor": {
      borderLeftColor: "var(--primary)",
    },
    "&.cm-focused .cm-selectionBackground, .cm-selectionBackground, .cm-content ::selection": {
      backgroundColor: "var(--cm-selection) !important",
    },
    ".cm-activeLine": {
      backgroundColor: "color-mix(in srgb, var(--cm-editor-fg) 5%, transparent)",
    },
    ".cm-gutters": {
      backgroundColor: "var(--cm-editor-gutter)",
      color: "var(--cm-meta)",
      border: "none",
      borderRight: "1px solid color-mix(in oklab, var(--cm-editor-fg) 12%, transparent)",
    },
    ".cm-activeLineGutter": {
      backgroundColor: "color-mix(in srgb, var(--cm-editor-fg) 8%, transparent)",
    },
    ".cm-foldPlaceholder": {
      backgroundColor: "var(--muted)",
      border: "none",
      color: "var(--muted-foreground)",
    },
  },
  { dark: true },
);

const studioHighlight = HighlightStyle.define([
  { tag: t.keyword, color: "var(--cm-keyword)" },
  { tag: t.propertyName, color: "var(--cm-property)" },
  { tag: t.string, color: "var(--cm-string)" },
  { tag: t.number, color: "var(--cm-number)" },
  { tag: t.bool, color: "var(--cm-number)" },
  { tag: t.null, color: "var(--cm-number)" },
  { tag: t.comment, color: "var(--cm-comment)", fontStyle: "italic" },
  { tag: t.invalid, color: "var(--cm-invalid)" },
]);

export const studioSyntaxHighlighting = syntaxHighlighting(studioHighlight);
