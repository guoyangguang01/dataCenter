<template>
  <div>
    <el-input :modelValue="data.script" @update:modelValue="update('script', $event)" type="textarea" :rows="5" placeholder="JavaScript 脚本" />
    <el-input-number :modelValue="data.timeout_ms" @update:modelValue="update('timeout_ms', $event)" :min="1000" :max="30000" placeholder="超时(ms)" style="margin-top: 8px" />
  </div>
</template>

<script setup>
const props = defineProps({ data: Object })
const emit = defineEmits(["update:data"])
const update = (key, val) => emit("update:data", { ...props.data, [key]: val })
</script>
