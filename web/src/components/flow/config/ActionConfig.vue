<template>
  <div>
    <el-select :modelValue="data.type" @update:modelValue="update('type', $event)" placeholder="动作类型" style="width: 100%; margin-bottom: 8px">
      <el-option label="publish" value="publish" />
      <el-option label="webhook" value="webhook" />
    </el-select>
    <el-input :modelValue="data.topic_template" @update:modelValue="update('topic_template', $event)" placeholder="topic_template" style="margin-bottom: 8px" />
    <el-input :modelValue="data.payload_template_str" @update:modelValue="update('payload_template_str', $event)" type="textarea" :rows="3" placeholder='payload_template JSON' />
  </div>
</template>

<script setup>
const props = defineProps({ data: Object })
const emit = defineEmits(["update:data"])
const update = (key, val) => emit("update:data", { ...props.data, [key]: val })
</script>
