<template>
  <div style="display: flex; gap: 8px">
    <el-input :modelValue="data.extract" @update:modelValue="update('extract', $event)" placeholder="extract topic" style="flex: 1" />
    <el-input :modelValue="data.parse" @update:modelValue="update('parse', $event)" placeholder="parse（预留）" style="flex: 1" />
  </div>
</template>

<script setup>
const props = defineProps({ data: Object })
const emit = defineEmits(["update:data"])
const update = (key, val) => emit("update:data", { ...props.data, [key]: val })
</script>
