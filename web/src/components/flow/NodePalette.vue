<template>
  <div class="node-palette">
    <div class="palette-title">节点类型</div>
    <div
      v-for="item in nodeTypes"
      :key="item.type"
      class="palette-item"
      :class="item.type"
      draggable="true"
      @dragstart="onDragStart($event, item.type)"
    >
      <el-tag size="small" :type="item.tagType">{{ item.label }}</el-tag>
      <span class="palette-desc">{{ item.desc }}</span>
    </div>
  </div>
</template>

<script setup>
const nodeTypes = [
  { type: "filter", label: "Filter", tagType: "", desc: "数据过滤" },
  { type: "transform", label: "Transform", tagType: "success", desc: "数据转换" },
  { type: "condition", label: "Condition", tagType: "warning", desc: "条件判断" },
  { type: "aggregate", label: "Aggregate", tagType: "info", desc: "聚合计算" },
  { type: "action", label: "Action", tagType: "danger", desc: "动作执行" },
  { type: "script", label: "Script", tagType: "primary", desc: "JS 脚本" }
]

const onDragStart = (event, type) => {
  event.dataTransfer.setData("application/vueflow", type)
  event.dataTransfer.effectAllowed = "move"
}
</script>

<style scoped>
.node-palette {
  width: 160px;
  border-right: 1px solid #eee;
  padding: 12px;
  background: #fafafa;
  overflow-y: auto;
}
.palette-title {
  font-size: 13px;
  font-weight: 600;
  color: #333;
  margin-bottom: 12px;
}
.palette-item {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 10px;
  margin-bottom: 6px;
  border: 1px solid #e4e7ed;
  border-radius: 6px;
  background: #fff;
  cursor: grab;
  transition: border-color 0.2s, box-shadow 0.2s;
  font-size: 12px;
}
.palette-item:hover {
  border-color: #409eff;
  box-shadow: 0 2px 8px rgba(64, 158, 255, 0.15);
}
.palette-item:active { cursor: grabbing; }
.palette-item.filter { border-left: 3px solid #409eff; }
.palette-item.transform { border-left: 3px solid #67c23a; }
.palette-item.condition { border-left: 3px solid #e6a23c; }
.palette-item.aggregate { border-left: 3px solid #909399; }
.palette-item.action { border-left: 3px solid #f56c6c; }
.palette-item.script { border-left: 3px solid #7b68ee; }
.palette-desc { font-size: 11px; color: #999; }
</style>
