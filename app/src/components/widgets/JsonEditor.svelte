<script lang="ts">
  import { basicSetup } from "codemirror";
  import { json } from "@codemirror/lang-json";
  import { EditorState } from "@codemirror/state";
  import { EditorView } from "@codemirror/view";
  import { Box } from "$ui8kit/ui";
  import { studioSyntaxHighlighting, workspaceEditorTheme } from "$lib/editor-theme";

  let {
    value,
    revision,
    onchange,
  }: {
    value: string;
    revision: number;
    onchange: (text: string) => void;
  } = $props();

  let view = $state<EditorView | null>(null);

  function mount(node: HTMLElement) {
    const editor = new EditorView({
      parent: node,
      state: EditorState.create({
        doc: value,
        extensions: [
          basicSetup,
          json(),
          workspaceEditorTheme,
          studioSyntaxHighlighting,
          EditorView.lineWrapping,
          EditorView.updateListener.of((update) => {
            if (!update.docChanged) return;
            onchange(update.state.doc.toString());
          }),
        ],
      }),
    });
    view = editor;
    return () => {
      editor.destroy();
      view = null;
    };
  }

  $effect(() => {
    const next = value;
    const token = revision;
    if (!view || token < 0) return;
    const current = view.state.doc.toString();
    if (current === next) return;
    view.dispatch({ changes: { from: 0, to: current.length, insert: next } });
  });
</script>

<Box class="min-h-0 w-full flex-1 overflow-hidden rounded-md border border-border" {@attach mount}></Box>
