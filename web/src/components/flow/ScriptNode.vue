<template>
  <div class="flow-node script" :class="{ selected }">
    <Handle type="target" :position="Position.Left" />
    <div class="node-header">
      <el-tag size="small" type="primary">Script</el-tag>
      <span class="node-id">{{ id }}</span>
    </div>
    <div class="node-summary">{{ summary }}</div>
    <Handle type="source" :position="Position.Right" />
  </div>
</template>

<style src="./node-styles.css"></style>

<script setup>
import { computed } from "vue"
import { Handle, Position } from "@vue-flow/core"

const props = defineProps({ id: String, data: Object, selected: Boolean })
const summary = computed(() => {
  const s = (props.data || {}).script || ""
  return s ? s.substring(0, 40) + (s.length > 40 ? "..." : "") : "未配置"
})
</script>
