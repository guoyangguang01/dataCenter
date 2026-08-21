<template>
  <div v-loading="loading">
    <el-row :gutter="20" style="margin-bottom: 20px">
      <el-col :span="6"><el-card shadow="hover"><el-statistic title="设备总数" :value="stats.device_total" /></el-card></el-col>
      <el-col :span="6"><el-card shadow="hover"><el-statistic title="在线设备" :value="stats.device_online" /></el-card></el-col>
      <el-col :span="6"><el-card shadow="hover"><el-statistic title="今日上报" :value="stats.message_today" /></el-card></el-col>
      <el-col :span="6"><el-card shadow="hover"><el-statistic title="24h 告警" :value="stats.alert_count" /></el-card></el-col>
    </el-row>
    <el-card><template #header>最近告警</template>
      <el-table :data="alerts" stripe>
        <el-table-column prop="created_at" label="时间" width="200">
          <template #default="{ row }">{{ formatTime(row.created_at) }}</template>
        </el-table-column>
        <el-table-column prop="device_id" label="设备" />
        <el-table-column prop="title" label="内容" />
        <el-table-column prop="status" label="状态" width="120">
          <template #default="{ row }">
            <el-tag :type="row.status === 'sent' ? 'success' : row.status === 'failed' ? 'danger' : 'warning'">
              {{ row.status }}
            </el-tag>
          </template>
        </el-table-column>
      </el-table>
    </el-card>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted } from "vue"
import { statsApi } from "../api"

const loading = ref(false)
const stats = reactive({
  device_total: 0,
  device_online: 0,
  message_today: 0,
  alert_count: 0,
})
const alerts = ref([])

const formatTime = (t) => {
  if (!t) return ""
  return new Date(t).toLocaleString("zh-CN")
}

onMounted(async () => {
  loading.value = true
  try {
    const res = await statsApi.getDashboard()
    const d = res.data.data
    stats.device_total = d.device_total
    stats.device_online = d.device_online
    stats.message_today = d.message_today
    stats.alert_count = d.alert_count
    alerts.value = d.recent_alerts || []
  } catch (e) {
    console.error("加载仪表盘数据失败:", e)
  } finally {
    loading.value = false
  }
})
</script>
