<template>
  <div>
    <el-input :modelValue="data.expression" @update:modelValue="update('expression', $event)" placeholder="表达式，如 temp > 30" />
    <div style="display: flex; gap: 8px; margin-top: 8px">
      <el-input :modelValue="data.true_branch" @update:modelValue="update('true_branch', $event)" placeholder="true_branch（可选）" style="flex: 1" />
      <el-input :modelValue="data.false_branch" @update:modelValue="update('false_branch', $event)" placeholder="false_branch（可选）" style="flex: 1" />
    </div>
  </div>
</template>

<script setup>
const props = defineProps({ data: Object })
const emit = defineEmits(["update:data"])
const update = (key, val) => emit("update:data", { ...props.data, [key]: val })
</script>
