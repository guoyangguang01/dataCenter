<template>
  <div>
    <el-card>
      <template #header>
        <div style="display: flex; justify-content: space-between; align-items: center">
          <span>数据维度</span>
          <div style="display: flex; gap: 12px; align-items: center">
            <el-select v-model="domainFilter" placeholder="选择域" clearable style="width: 160px" @change="onDomainChange">
              <el-option v-for="d in domains" :key="d.id" :label="d.name" :value="d.id" />
            </el-select>
            <el-tag type="info">共 {{ metricKeys.length }} 个指标</el-tag>
            <el-switch v-model="autoRefresh" active-text="自动刷新" @change="toggleAutoRefresh" />
            <el-button type="primary" @click="loadData" :loading="loading">刷新</el-button>
          </div>
        </div>
      </template>

      <div v-if="loading && metricKeys.length === 0" style="text-align: center; padding: 60px 0">
        <el-icon class="is-loading" :size="32"><Loading /></el-icon>
        <div style="margin-top: 8px; color: #999">加载中...</div>
      </div>

      <div v-else-if="metricKeys.length === 0" style="text-align: center; padding: 60px 0">
        <el-empty description="暂无实时数据" />
      </div>

      <el-row :gutter="16" v-else>
        <el-col :xs="24" :sm="12" :lg="8" v-for="key in metricKeys" :key="key" style="margin-bottom: 16px">
          <el-card shadow="hover">
            <template #header>
              <div style="display: flex; justify-content: space-between; align-items: center">
                <span style="font-weight: 600">{{ metricData[key].name }}</span>
                <el-tag size="small" type="info">{{ metricData[key].devices.length }} 台设备</el-tag>
              </div>
            </template>

            <!-- 汇总 -->
            <el-row :gutter="8" style="margin-bottom: 12px">
              <el-col :span="8">
                <div style="text-align: center">
                  <div style="font-size: 11px; color: #999">平均</div>
                  <div style="font-size: 16px; font-weight: 600; color: #409eff">{{ avg(key) }}</div>
                </div>
              </el-col>
              <el-col :span="8">
                <div style="text-align: center">
                  <div style="font-size: 11px; color: #999">最大</div>
                  <div style="font-size: 16px; font-weight: 600; color: #f56c6c">{{ max(key) }}</div>
                </div>
              </el-col>
              <el-col :span="8">
                <div style="text-align: center">
                  <div style="font-size: 11px; color: #999">最小</div>
                  <div style="font-size: 16px; font-weight: 600; color: #67c23a">{{ min(key) }}</div>
                </div>
              </el-col>
            </el-row>

            <!-- 设备列表 -->
            <el-table :data="metricData[key].devices" size="small" max-height="200" :show-header="false">
              <el-table-column>
                <template #default="{ row }">
                  <div style="display: flex; justify-content: space-between; align-items: center">
                    <span style="color: #666; font-size: 13px">{{ deviceName(row.device_id) }}</span>
                    <span>
                      <span style="font-weight: 600; color: #303133">{{ formatVal(row.value) }}</span>
                      <span v-if="metricData[key].unit" style="color: #999; font-size: 12px; margin-left: 2px">{{ metricData[key].unit }}</span>
                    </span>
                  </div>
                </template>
              </el-table-column>
            </el-table>
          </el-card>
        </el-col>
      </el-row>
    </el-card>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, onUnmounted, watch } from "vue"
import { dataApi, deviceApi, domainApi } from "../api"
import { useAppStore } from "../stores/app"
import { Loading } from "@element-plus/icons-vue"

const store = useAppStore()
const loading = ref(false)
const metricData = ref({})
const deviceNameMap = ref({})
const domains = ref([])
const domainFilter = ref("")

// 自动刷新
const autoRefresh = ref(true)
const refreshInterval = ref(5)
let timer = null

// 指标 key 列表（排序）
const metricKeys = computed(() => Object.keys(metricData.value).sort())

// 设备名称
const deviceName = (id) => deviceNameMap.value[id] || id

// 格式化数值
const formatVal = (v) => {
  if (typeof v === "number") return v.toFixed(1)
  return v
}

// 汇总计算
const avg = (key) => {
  const devices = metricData.value[key]?.devices || []
  if (devices.length === 0) return "-"
  const sum = devices.reduce((a, d) => a + d.value, 0)
  return (sum / devices.length).toFixed(1)
}
const max = (key) => {
  const devices = metricData.value[key]?.devices || []
  if (devices.length === 0) return "-"
  return Math.max(...devices.map(d => d.value)).toFixed(1)
}
const min = (key) => {
  const devices = metricData.value[key]?.devices || []
  if (devices.length === 0) return "-"
  return Math.min(...devices.map(d => d.value)).toFixed(1)
}

// 加载设备名称
const loadDeviceNames = async () => {
  try {
    const params = { page: 1, page_size: 500 }
    if (domainFilter.value) params.domain_id = domainFilter.value
    const res = await deviceApi.list(params)
    const devices = res.data.data || []
    const map = {}
    devices.forEach(d => { map[d.id] = d.name })
    deviceNameMap.value = map
  } catch {}
}

// 加载数据
const loadData = async () => {
  loading.value = true
  try {
    const res = await dataApi.getAllLatest(domainFilter.value || undefined)
    metricData.value = res.data.data || {}
  } catch {
    metricData.value = {}
  } finally {
    loading.value = false
  }
}

// 域筛选变化
const onDomainChange = () => {
  stopAutoRefresh()
  loadDeviceNames()
  loadData()
  startAutoRefresh()
}

// 自动刷新
const startAutoRefresh = () => {
  stopAutoRefresh()
  if (autoRefresh.value) {
    timer = setInterval(() => loadData(), refreshInterval.value * 1000)
  }
}

const stopAutoRefresh = () => {
  if (timer) {
    clearInterval(timer)
    timer = null
  }
}

const toggleAutoRefresh = () => {
  if (autoRefresh.value) {
    startAutoRefresh()
  } else {
    stopAutoRefresh()
  }
}

// 全局域选择器变化时联动
watch(() => store.currentDomain, (val) => {
  domainFilter.value = val
  onDomainChange()
})

onMounted(async () => {
  domains.value = store.domains
  if (store.currentDomain) domainFilter.value = store.currentDomain
  await loadDeviceNames()
  await loadData()
  startAutoRefresh()
})

onUnmounted(() => stopAutoRefresh())
</script>
