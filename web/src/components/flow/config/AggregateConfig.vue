<template>
  <div style="display: flex; gap: 8px">
    <el-input-number :modelValue="data.window_size" @update:modelValue="update('window_size', $event)" :min="1" placeholder="窗口大小" />
    <el-select :modelValue="data.function" @update:modelValue="update('function', $event)" placeholder="函数" style="width: 120px">
      <el-option label="avg" value="avg" />
      <el-option label="sum" value="sum" />
      <el-option label="min" value="min" />
      <el-option label="max" value="max" />
      <el-option label="count" value="count" />
    </el-select>
  </div>
</template>

<script setup>
const props = defineProps({ data: Object })
const emit = defineEmits(["update:data"])
const update = (key, val) => emit("update:data", { ...props.data, [key]: val })
</script>
