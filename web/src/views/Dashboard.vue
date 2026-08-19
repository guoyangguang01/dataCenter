<template>
  <div>
    <el-row :gutter="20" style="margin-bottom: 20px">
      <el-col :span="6"><el-card shadow="hover"><el-statistic title="设备总数" :value="320" /></el-card></el-col>
      <el-col :span="6"><el-card shadow="hover"><el-statistic title="在线设备" :value="287" /></el-card></el-col>
      <el-col :span="6"><el-card shadow="hover"><el-statistic title="今日上报" :value="1203456" /></el-card></el-col>
      <el-col :span="6"><el-card shadow="hover"><el-statistic title="活跃告警" :value="12" /></el-card></el-col>
    </el-row>
    <el-card><template #header>最近告警</template>
      <el-table :data="alerts" stripe>
        <el-table-column prop="time" label="时间" width="180" />
        <el-table-column prop="device" label="设备" />
        <el-table-column prop="message" label="内容" />
        <el-table-column prop="level" label="级别" width="100">
          <template #default="{ row }">
            <el-tag :type="row.level === 'critical' ? 'danger' : 'warning'">{{ row.level }}</el-tag>
          </template>
        </el-table-column>
      </el-table>
    </el-card>
  </div>
</template>

<script setup>
import { ref } from "vue"
const alerts = ref([
  { time: '14:23', device: 'sensor-001', message: '温度异常 35.6°C', level: 'critical' },
  { time: '14:20', device: 'ctrl-05', message: '设备离线', level: 'warning' },
])
</script>
