<script lang="ts">
  import { Handle, Position, type NodeProps } from "@xyflow/svelte";
  import { Block } from "$ui8kit/ui";

  let _: NodeProps = $props();
  const handleStyle = "width: 12px; height: 12px;";
</script>

<Block class="h-full w-full">
  <Handle type="target" position={Position.Top} style={handleStyle} />
  <Handle type="source" position={Position.Bottom} style={handleStyle} />
</Block>
