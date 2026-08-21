<template>
  <div class="flow-node condition" :class="{ selected }">
    <Handle type="target" :position="Position.Left" />
    <div class="node-header">
      <el-tag size="small" type="warning">Condition</el-tag>
      <span class="node-id">{{ id }}</span>
    </div>
    <div class="node-summary">{{ summary }}</div>
    <div class="condition-outputs">
      <div class="output-label true-label">
        <span>T</span>
        <Handle type="source" :position="Position.Right" id="true" :style="{ top: '35%' }" />
      </div>
      <div class="output-label false-label">
        <span>F</span>
        <Handle type="source" :position="Position.Right" id="false" :style="{ top: '65%' }" />
      </div>
    </div>
  </div>
</template>

<style src="./node-styles.css"></style>

<script setup>
import { computed } from "vue"
import { Handle, Position } from "@vue-flow/core"

const props = defineProps({ id: String, data: Object, selected: Boolean })
const summary = computed(() => (props.data || {}).expression || "未配置")
</script>

<style scoped>
.condition-outputs { display: flex; flex-direction: column; gap: 4px; position: absolute; right: -8px; top: 50%; transform: translateY(-50%); }
.output-label span { font-size: 10px; font-weight: bold; margin-right: 4px; }
.true-label span { color: #67c23a; }
.false-label span { color: #f56c6c; }
</style>
