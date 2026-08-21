<template>
  <div style="display: flex; gap: 8px">
    <el-input :modelValue="data.field" @update:modelValue="update('field', $event)" placeholder="field" style="width: 100px" />
    <el-select :modelValue="data.operator" @update:modelValue="update('operator', $event)" placeholder="op" style="width: 100px">
      <el-option label="eq" value="eq" />
      <el-option label="contains" value="contains" />
      <el-option label="prefix" value="prefix" />
    </el-select>
    <el-input :modelValue="data.value" @update:modelValue="update('value', $event)" placeholder="value" style="flex: 1" />
  </div>
</template>

<script setup>
const props = defineProps({ data: Object })
const emit = defineEmits(["update:data"])
const update = (key, val) => emit("update:data", { ...props.data, [key]: val })
</script>
