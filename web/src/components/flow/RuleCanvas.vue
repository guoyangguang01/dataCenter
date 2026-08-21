<template>
  <div class="rule-canvas-container">
    <NodePalette />
    <div class="canvas-area" @drop="onDrop" @dragover="onDragOver">
      <VueFlow
        v-model:nodes="nodes"
        v-model:edges="edges"
        :node-types="nodeTypes"
        :default-edge-options="{ type: 'smoothstep', animated: true }"
        :snap-to-grid="true"
        :snap-grid="[20, 20]"
        fit-view-on-init
        @node-click="onNodeClick"
        @connect="onConnect"
      >
        <Background />
        <Controls />
        <MiniMap />
      </VueFlow>
    </div>

    <!-- 节点配置 Drawer -->
    <el-drawer v-model="drawerVisible" :title="drawerTitle" size="400px" direction="rtl">
      <component :is="configComponent" v-if="selectedNode" :data="selectedNode.data" @update:data="onConfigUpdate" />
      <template #footer>
        <el-button type="danger" @click="deleteSelectedNode">删除节点</el-button>
        <el-button type="primary" @click="drawerVisible = false">完成</el-button>
      </template>
    </el-drawer>
  </div>
</template>

<script setup>
import { ref, computed, watch, markRaw } from "vue"
import { VueFlow, useVueFlow } from "@vue-flow/core"
import "@vue-flow/core/dist/style.css"
import "@vue-flow/core/dist/theme-default.css"
import "@vue-flow/minimap/dist/style.css"
import { Background } from "@vue-flow/background"
import { Controls } from "@vue-flow/controls"
import { MiniMap } from "@vue-flow/minimap"
import NodePalette from "./NodePalette.vue"

import FilterNode from "./FilterNode.vue"
import TransformNode from "./TransformNode.vue"
import ConditionNode from "./ConditionNode.vue"
import AggregateNode from "./AggregateNode.vue"
import ActionNode from "./ActionNode.vue"
import ScriptNode from "./ScriptNode.vue"

import FilterConfig from "./config/FilterConfig.vue"
import TransformConfig from "./config/TransformConfig.vue"
import ConditionConfig from "./config/ConditionConfig.vue"
import AggregateConfig from "./config/AggregateConfig.vue"
import ActionConfig from "./config/ActionConfig.vue"
import ScriptConfig from "./config/ScriptConfig.vue"

const nodeTypes = {
  filter: markRaw(FilterNode),
  transform: markRaw(TransformNode),
  condition: markRaw(ConditionNode),
  aggregate: markRaw(AggregateNode),
  action: markRaw(ActionNode),
  script: markRaw(ScriptNode)
}

const configComponents = {
  filter: markRaw(FilterConfig),
  transform: markRaw(TransformConfig),
  condition: markRaw(ConditionConfig),
  aggregate: markRaw(AggregateConfig),
  action: markRaw(ActionConfig),
  script: markRaw(ScriptConfig)
}

const props = defineProps({ chain: { type: Array, default: () => [] } })
const emit = defineEmits(["update:chain"])

const nodes = ref([])
const edges = ref([])
const drawerVisible = ref(false)
const selectedNodeId = ref(null)

const { screenToFlowCoordinate } = useVueFlow()

const selectedNode = computed(() => nodes.value.find(n => n.id === selectedNodeId.value))
const configComponent = computed(() => selectedNode.value ? configComponents[selectedNode.value.type] : null)
const drawerTitle = computed(() => {
  if (!selectedNode.value) return ""
  const labels = { filter: "Filter", transform: "Transform", condition: "Condition", aggregate: "Aggregate", action: "Action", script: "Script" }
  return `${labels[selectedNode.value.type] || ""} 节点配置`
})

// chain → nodes/edges
const chainToFlow = (chain) => {
  if (!chain || chain.length === 0) { nodes.value = []; edges.value = []; return }

  const positions = autoLayout(chain)
  nodes.value = chain.map((n, i) => ({
    id: n.id,
    type: n.type,
    position: positions[i] || { x: i * 250, y: 100 },
    data: { ...n.config }
  }))

  const newEdges = []
  for (let i = 0; i < chain.length; i++) {
    const node = chain[i]
    if (node.type === "condition") {
      if (node.config.true_branch) {
        newEdges.push({ id: `e-${node.id}-true`, source: node.id, target: node.config.true_branch, sourceHandle: "true", label: "T", type: "smoothstep", animated: true })
      }
      if (node.config.false_branch) {
        newEdges.push({ id: `e-${node.id}-false`, source: node.id, target: node.config.false_branch, sourceHandle: "false", label: "F", type: "smoothstep", animated: true })
      }
    }
    // Sequential edge (if not condition with branches, or as fallback)
    if (i < chain.length - 1 && node.type !== "condition") {
      newEdges.push({ id: `e-${node.id}-${chain[i + 1].id}`, source: node.id, target: chain[i + 1].id, type: "smoothstep", animated: true })
    } else if (i < chain.length - 1 && node.type === "condition" && !node.config.true_branch && !node.config.false_branch) {
      newEdges.push({ id: `e-${node.id}-${chain[i + 1].id}`, source: node.id, target: chain[i + 1].id, type: "smoothstep", animated: true })
    }
  }
  edges.value = newEdges
}

// Auto-layout: topological sort with level assignment
const autoLayout = (chain) => {
  const positions = {}
  const nodeMap = new Map(chain.map(n => [n.id, n]))

  // Build adjacency and in-degree
  const inDegree = new Map(chain.map(n => [n.id, 0]))
  const children = new Map(chain.map(n => [n.id, []]))

  for (let i = 0; i < chain.length - 1; i++) {
    const node = chain[i]
    if (node.type === "condition" && node.config.true_branch) {
      children.get(node.id).push(node.config.true_branch)
      inDegree.set(node.config.true_branch, (inDegree.get(node.config.true_branch) || 0) + 1)
    }
    if (node.type === "condition" && node.config.false_branch) {
      children.get(node.id).push(node.config.false_branch)
      inDegree.set(node.config.false_branch, (inDegree.get(node.config.false_branch) || 0) + 1)
    }
    // Sequential link (skip if condition has branches)
    if (node.type !== "condition" || (!node.config.true_branch && !node.config.false_branch)) {
      children.get(node.id).push(chain[i + 1].id)
      inDegree.set(chain[i + 1].id, (inDegree.get(chain[i + 1].id) || 0) + 1)
    }
  }

  // BFS level assignment
  const levels = new Map()
  const queue = []
  for (const [id, deg] of inDegree) {
    if (deg === 0) { queue.push(id); levels.set(id, 0) }
  }

  while (queue.length > 0) {
    const current = queue.shift()
    const currentLevel = levels.get(current)
    for (const child of (children.get(current) || [])) {
      const existing = levels.get(child)
      const newLevel = currentLevel + 1
      if (existing === undefined || newLevel > existing) {
        levels.set(child, newLevel)
      }
      const deg = inDegree.get(child) - 1
      inDegree.set(child, deg)
      if (deg === 0) queue.push(child)
    }
  }

  // Assign positions
  const levelNodes = new Map()
  for (const [id, level] of levels) {
    if (!levelNodes.has(level)) levelNodes.set(level, [])
    levelNodes.get(level).push(id)
  }

  for (const [level, ids] of levelNodes) {
    ids.forEach((id, idx) => {
      positions[id] = { x: level * 280, y: idx * 150 + 50 }
    })
  }

  // Fallback for any nodes not positioned
  chain.forEach((n, i) => {
    if (!positions[n.id]) positions[n.id] = { x: i * 280, y: 50 }
  })

  return positions
}

// nodes/edges → chain
const flowToChain = () => {
  if (nodes.value.length === 0) return []

  // Build adjacency from edges
  const adj = new Map()
  const inDeg = new Map(nodes.value.map(n => [n.id, 0]))
  for (const e of edges.value) {
    if (!adj.has(e.source)) adj.set(e.source, [])
    adj.get(e.source).push({ target: e.target, handle: e.sourceHandle })
    inDeg.set(e.target, (inDeg.get(e.target) || 0) + 1)
  }

  // Topological sort (BFS)
  const queue = []
  for (const [id, deg] of inDeg) { if (deg === 0) queue.push(id) }

  const sorted = []
  const visited = new Set()
  while (queue.length > 0) {
    const current = queue.shift()
    if (visited.has(current)) continue
    visited.add(current)
    sorted.push(current)
    for (const edge of (adj.get(current) || [])) {
      const deg = inDeg.get(edge.target) - 1
      inDeg.set(edge.target, deg)
      if (deg === 0) queue.push(edge.target)
    }
  }

  // Add any unvisited nodes (isolated)
  for (const n of nodes.value) {
    if (!visited.has(n.id)) sorted.push(n.id)
  }

  return sorted.map(id => {
    const node = nodes.value.find(n => n.id === id)
    const config = { ...node.data }
    // For action nodes, parse payload_template_str back
    if (node.type === "action" && config.payload_template_str) {
      try { config.payload_template = JSON.parse(config.payload_template_str) } catch { config.payload_template = {} }
      delete config.payload_template_str
    }
    // For condition nodes, extract branches from edges
    if (node.type === "condition") {
      const trueEdge = edges.value.find(e => e.source === id && e.sourceHandle === "true")
      const falseEdge = edges.value.find(e => e.source === id && e.sourceHandle === "false")
      config.true_branch = trueEdge ? trueEdge.target : ""
      config.false_branch = falseEdge ? falseEdge.target : ""
    }
    return { id: node.id, type: node.type, config }
  })
}

const onNodeClick = ({ node }) => {
  selectedNodeId.value = node.id
  drawerVisible.value = true
}

const onConnect = (params) => {
  edges.value.push({
    id: `e-${params.source}-${params.target}${params.sourceHandle ? "-" + params.sourceHandle : ""}`,
    ...params,
    type: "smoothstep",
    animated: true
  })
}

const onDragOver = (event) => {
  event.preventDefault()
  if (event.dataTransfer) event.dataTransfer.dropEffect = "move"
}

const onDrop = (event) => {
  const type = event.dataTransfer.getData("application/vueflow")
  if (!type) return
  const position = screenToFlowCoordinate({ x: event.clientX, y: event.clientY })
  const defaultConfigs = {
    filter: { field: "topic", operator: "eq", value: "" },
    transform: { extract: "", parse: "" },
    condition: { expression: "", true_branch: "", false_branch: "" },
    aggregate: { window_size: 10, function: "avg" },
    action: { type: "publish", topic_template: "", payload_template_str: "{}" },
    script: { script: "", timeout_ms: 5000 }
  }
  const newNode = {
    id: `${type}_${Date.now()}`,
    type,
    position,
    data: { ...(defaultConfigs[type] || {}) }
  }
  nodes.value.push(newNode)
  emitChain()
}

const onConfigUpdate = (newData) => {
  if (!selectedNodeId.value) return
  const node = nodes.value.find(n => n.id === selectedNodeId.value)
  if (node) { node.data = newData; emitChain() }
}

const deleteSelectedNode = () => {
  if (!selectedNodeId.value) return
  nodes.value = nodes.value.filter(n => n.id !== selectedNodeId.value)
  edges.value = edges.value.filter(e => e.source !== selectedNodeId.value && e.target !== selectedNodeId.value)
  drawerVisible.value = false
  selectedNodeId.value = null
  emitChain()
}

const emitChain = () => {
  emit("update:chain", flowToChain())
}

// Watch for external chain changes
watch(() => props.chain, (newChain) => {
  chainToFlow(newChain)
}, { deep: true, immediate: true })

// Watch nodes/edges for internal changes → emit chain
watch([nodes, edges], () => { emitChain() }, { deep: true })
</script>

<style scoped>
.rule-canvas-container {
  display: flex;
  height: 500px;
  border: 1px solid #e4e7ed;
  border-radius: 4px;
  overflow: hidden;
}
.canvas-area {
  flex: 1;
  position: relative;
}
</style>
