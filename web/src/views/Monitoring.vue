<template>
  <div v-loading="loading">
    <!-- 网关状态 -->
    <el-card style="margin-bottom: 20px">
      <template #header>
        <div style="display: flex; justify-content: space-between; align-items: center">
          <span>网关状态</span>
          <el-tag type="info">{{ gatewayStats.running }} / {{ gatewayStats.total }} 运行中</el-tag>
        </div>
      </template>
      <el-row :gutter="16">
        <el-col :span="6" v-for="gw in gateways" :key="gw.id">
          <el-card shadow="hover" style="margin-bottom: 12px">
            <div style="display: flex; justify-content: space-between; align-items: center">
              <div>
                <div style="font-weight: 600">{{ gw.name }}</div>
                <el-tag size="small" style="margin-top: 4px">{{ gw.type }}</el-tag>
              </div>
              <el-tag :type="gw.status === 'running' ? 'success' : gw.status === 'error' ? 'danger' : 'info'" effect="dark">
                {{ gw.status }}
              </el-tag>
            </div>
          </el-card>
        </el-col>
        <el-col v-if="gateways.length === 0" :span="24">
          <el-empty description="暂无网关配置" :image-size="60" />
        </el-col>
      </el-row>
    </el-card>

    <!-- 设备连接概览 -->
    <el-card style="margin-bottom: 20px">
      <template #header>设备连接概览</template>
      <el-row :gutter="20">
        <el-col :span="12">
          <div style="margin-bottom: 8px; color: #666">在线率</div>
          <el-progress
            :percentage="onlineRate"
            :color="onlineRate > 80 ? '#67c23a' : onlineRate > 50 ? '#e6a23c' : '#f56c6c'"
            :stroke-width="20"
            text-inside
          />
          <div style="margin-top: 8px; color: #999; font-size: 13px">
            {{ deviceStats.online }} / {{ deviceStats.total }} 设备在线
          </div>
        </el-col>
        <el-col :span="12">
          <el-row :gutter="16">
            <el-col :span="12">
              <el-statistic title="设备总数" :value="deviceStats.total" />
            </el-col>
            <el-col :span="12">
              <el-statistic title="在线设备" :value="deviceStats.online" />
            </el-col>
          </el-row>
        </el-col>
      </el-row>
    </el-card>
  </div>
</template>

<script setup>
import { ref, reactive, computed, onMounted } from "vue"
import { statsApi } from "../api"

const loading = ref(false)
const gateways = ref([])
const gatewayStats = reactive({ running: 0, total: 0 })
const deviceStats = reactive({ online: 0, total: 0 })

const onlineRate = computed(() => {
  if (deviceStats.total === 0) return 0
  return Math.round((deviceStats.online / deviceStats.total) * 100)
})

onMounted(async () => {
  loading.value = true
  try {
    const res = await statsApi.getMonitoring()
    const d = res.data.data
    gateways.value = d.gateways || []
    gatewayStats.running = d.gateway_running
    gatewayStats.total = d.gateway_total
    deviceStats.online = d.device_online
    deviceStats.total = d.device_total
  } catch (e) {
    console.error("加载监控数据失败:", e)
  } finally {
    loading.value = false
  }
})
</script>
