<template>
  <div class="flow-node action" :class="{ selected }">
    <Handle type="target" :position="Position.Left" />
    <div class="node-header">
      <el-tag size="small" type="danger">Action</el-tag>
      <span class="node-id">{{ id }}</span>
    </div>
    <div class="node-summary">{{ summary }}</div>
  </div>
</template>

<style src="./node-styles.css"></style>

<script setup>
import { computed } from "vue"
import { Handle, Position } from "@vue-flow/core"

const props = defineProps({ id: String, data: Object, selected: Boolean })
const summary = computed(() => {
  const c = props.data || {}
  return `${c.type || "?"} → ${c.topic_template || "?"}`
})
</script>
