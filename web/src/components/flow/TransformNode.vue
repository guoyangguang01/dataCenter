<template>
  <div class="flow-node transform" :class="{ selected }">
    <Handle type="target" :position="Position.Left" />
    <div class="node-header">
      <el-tag size="small" type="success">Transform</el-tag>
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
const summary = computed(() => `extract: ${(props.data || {}).extract || "?"}`)
</script>
