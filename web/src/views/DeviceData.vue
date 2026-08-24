<template>
  <div>
    <el-card>
      <template #header>
        <div style="display: flex; justify-content: space-between; align-items: center">
          <span>设备实时数据</span>
          <div style="display: flex; gap: 12px; align-items: center">
            <el-select v-model="domainFilter" placeholder="选择域" clearable style="width: 160px" @change="loadDevices">
              <el-option v-for="d in domains" :key="d.id" :label="d.name" :value="d.id" />
            </el-select>
            <el-select v-model="selectedDevice" placeholder="选择设备" style="width: 240px" @change="loadData">
              <el-option v-for="d in devices" :key="d.id" :label="d.name + ' (' + d.id + ')'" :value="d.id" />
            </el-select>
            <el-select v-model="timeRange" style="width: 120px" @change="loadData">
              <el-option label="最近1小时" :value="1" />
              <el-option label="最近6小时" :value="6" />
              <el-option label="最近24小时" :value="24" />
              <el-option label="最近7天" :value="168" />
            </el-select>
            <el-switch v-model="autoRefresh" active-text="自动刷新" @change="toggleAutoRefresh" style="margin-left: 8px" />
            <el-button type="primary" @click="loadData" :loading="loading">刷新</el-button>
          </div>
        </div>
      </template>

      <div v-if="!selectedDevice" style="text-align: center; padding: 60px 0">
        <el-empty description="请先选择一个设备" />
      </div>

      <template v-else>
        <!-- 最新数据卡片 -->
        <el-row :gutter="16" style="margin-bottom: 20px">
          <el-col :span="6" v-for="(item, i) in latestMetrics" :key="i">
            <el-card shadow="hover">
              <el-statistic :title="item.name" :value="item.value" :suffix="item.unit" />
              <div style="font-size: 12px; color: #999; margin-top: 4px">{{ item.time }}</div>
            </el-card>
          </el-col>
          <el-col v-if="latestMetrics.length === 0" :span="24">
            <el-empty description="暂无数据" :image-size="60" />
          </el-col>
        </el-row>

        <!-- 趋势图 -->
        <el-card style="margin-bottom: 20px">
          <template #header>数据趋势</template>
          <div ref="chartRef" style="height: 350px"></div>
        </el-card>

        <!-- 数据表格 -->
        <el-card>
          <template #header>
            <div style="display: flex; justify-content: space-between; align-items: center">
              <span>数据明细</span>
              <el-tag type="info">共 {{ tableData.length }} 条</el-tag>
            </div>
          </template>
          <el-table :data="tableData" stripe v-loading="loading" max-height="400">
            <el-table-column prop="ts" label="时间" width="200" />
            <el-table-column prop="topic" label="Topic" width="200" />
            <el-table-column prop="value" label="值" />
            <el-table-column prop="quality" label="质量" width="100">
              <template #default="{ row }">
                <el-tag :type="row.quality === 0 ? 'success' : 'warning'" size="small">
                  {{ row.quality === 0 ? '正常' : '异常' }}
                </el-tag>
              </template>
            </el-table-column>
          </el-table>
        </el-card>
      </template>
    </el-card>
  </div>
</template>

<script setup>
import { ref, onMounted, onUnmounted, nextTick, watch } from "vue"
import { deviceApi, dataApi, domainApi, modelApi } from "../api"
import * as echarts from "echarts"

const domains = ref([])
const domainFilter = ref("")
const devices = ref([])
const selectedDevice = ref("")
const timeRange = ref(1)
const loading = ref(false)
const latestMetrics = ref([])
const tableData = ref([])
const chartRef = ref(null)
let chart = null

// 物模型属性映射：propertyId → { name, unit }
const propMap = ref({})

// 自动刷新
const autoRefresh = ref(true)
const refreshInterval = ref(5)
let timer = null
let lastTs = 0  // 追踪最后一条数据的时间戳

const loadDevices = async () => {
  const params = { page: 1, page_size: 100 }
  if (domainFilter.value) params.domain_id = domainFilter.value
  const res = await deviceApi.list(params)
  devices.value = res.data.data || []
  if (selectedDevice.value && !devices.value.find(d => d.id === selectedDevice.value)) {
    selectedDevice.value = ""
  }
}

onMounted(async () => {
  const res = await domainApi.list()
  domains.value = res.data.data || []
  await loadDevices()
})

// 加载设备的物模型，构建属性映射
const loadModel = async (deviceId) => {
  propMap.value = {}
  try {
    const res = await modelApi.getDeviceModel(deviceId)
    const model = res.data
    if (model && model.properties) {
      model.properties.forEach(p => {
        propMap.value[p.id] = { name: p.name, unit: p.unit || "" }
      })
    }
  } catch {}
}

// 获取属性显示名称
const propName = (key) => propMap.value[key]?.name || key
const propUnit = (key) => propMap.value[key]?.unit || ""

// 解析一行数据为对象
const parseRow = (row, columns) => {
  const obj = {}
  columns.forEach((col, i) => { obj[col] = row[i] })
  return obj
}

// 解析 val JSON
const parseVal = (obj) => {
  try { return JSON.parse(obj.val || obj["last_row(val)"] || "{}") } catch { return {} }
}

// 更新最新指标卡片
const updateLatestMetrics = async () => {
  if (!selectedDevice.value) return
  try {
    const res = await dataApi.getLatestData(selectedDevice.value, 20)
    const rows = res.data.data?.rows || []
    const columns = res.data.data?.columns || []
    if (rows.length > 0) {
      const obj = parseRow(rows[0], columns)
      const parsed = parseVal(obj)
      const ts = obj.ts || obj["last_row(ts)"]
      latestMetrics.value = Object.entries(parsed)
        .filter(([k]) => k !== "device_id" && k !== "timestamp" && typeof parsed[k] === "number")
        .map(([k, v]) => ({
          name: propName(k),
          value: typeof v === "number" ? v.toFixed(1) : v,
          unit: propUnit(k),
          time: ts ? new Date(ts).toLocaleString("zh-CN") : "-"
        }))
    }
  } catch {}
}

// 从历史数据解析表格行
const buildTableRow = (row, columns) => {
  const obj = parseRow(row, columns)
  const parsed = parseVal(obj)
  const displayVal = Object.entries(parsed)
    .filter(([k]) => k !== "device_id" && k !== "timestamp")
    .map(([k, v]) => {
      const unit = propUnit(k)
      return `${propName(k)}: ${v}${unit ? " " + unit : ""}`
    })
    .join(", ")
  return {
    ts: obj.ts ? new Date(obj.ts).toLocaleString("zh-CN") : "-",
    tsRaw: obj.ts || "",
    topic: parsed.device_id || "-",
    value: displayVal || obj.val || "-",
    quality: obj.quality ?? 0
  }
}

// 首次完整加载
const loadData = async () => {
  if (!selectedDevice.value) return
  loading.value = true
  try {
    await loadModel(selectedDevice.value)

    // 加载最新指标
    await updateLatestMetrics()

    // 加载历史数据
    const historyRes = await dataApi.getDeviceData(selectedDevice.value, timeRange.value)
    const historyRows = historyRes.data.data?.rows || []
    const historyColumns = historyRes.data.data?.columns || []

    tableData.value = historyRows.map(row => buildTableRow(row, historyColumns)).reverse()

    // 记录最后时间戳
    if (historyRows.length > 0) {
      const lastObj = parseRow(historyRows[historyRows.length - 1], historyColumns)
      lastTs = new Date(lastObj.ts).getTime()
    }

    // 渲染图表
    await nextTick()
    renderChart(historyRows, historyColumns)

    // 启动自动刷新
    startAutoRefresh()
  } catch (e) {
    console.error("加载数据失败:", e)
  } finally {
    loading.value = false
  }
}

// 增量追加新数据
const refreshData = async () => {
  if (!selectedDevice.value || !chart) return
  try {
    await updateLatestMetrics()

    const historyRes = await dataApi.getDeviceData(selectedDevice.value, timeRange.value)
    const historyRows = historyRes.data.data?.rows || []
    const historyColumns = historyRes.data.data?.columns || []

    // 筛选新数据（时间戳 > lastTs）
    const newRows = historyRows.filter(row => {
      const obj = parseRow(row, historyColumns)
      return new Date(obj.ts).getTime() > lastTs
    })

    if (newRows.length === 0) return

    // 更新 lastTs
    const newest = parseRow(newRows[newRows.length - 1], historyColumns)
    lastTs = new Date(newest.ts).getTime()

    // 追加表格数据（新数据在前）
    const newTableRows = newRows.map(row => buildTableRow(row, historyColumns))
    tableData.value = [...newTableRows.reverse(), ...tableData.value]

    // 追加图表数据
    appendChartData(newRows, historyColumns)
  } catch {}
}

// 追加数据到图表（不重建）
const appendChartData = (rows, columns) => {
  if (!chart) return
  const option = chart.getOption()
  const seriesMap = {}
  option.series.forEach((s, i) => { seriesMap[s.name] = i })

  rows.forEach(row => {
    const obj = parseRow(row, columns)
    const parsed = parseVal(obj)
    const ts = new Date(obj.ts).getTime()
    Object.entries(parsed).forEach(([key, val]) => {
      if (key === "device_id" || key === "timestamp" || typeof val !== "number") return
      const label = propName(key)
      const unit = propUnit(key)
      const seriesName = unit ? `${label}(${unit})` : label
      if (seriesMap[seriesName] !== undefined) {
        option.series[seriesMap[seriesName]].data.push([ts, val])
      }
    })
  })

  chart.setOption({ series: option.series })
}

// 自动刷新控制
const startAutoRefresh = () => {
  stopAutoRefresh()
  if (autoRefresh.value) {
    timer = setInterval(() => refreshData(), refreshInterval.value * 1000)
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

onUnmounted(() => stopAutoRefresh())

const renderChart = (rows, columns) => {
  if (!chartRef.value || rows.length === 0) return

  if (chart) chart.dispose()
  chart = echarts.init(chartRef.value)

  // 按指标分组（从 val JSON 中解析）
  const metricMap = new Map()
  rows.forEach(row => {
    const obj = {}
    columns.forEach((col, i) => { obj[col] = row[i] })
    let parsed = {}
    try { parsed = JSON.parse(obj.val || "{}") } catch {}
    const ts = new Date(obj.ts).getTime()
    Object.entries(parsed).forEach(([key, val]) => {
      if (key === "device_id" || key === "timestamp") return
      if (typeof val !== "number") return
      const label = propName(key)
      if (!metricMap.has(label)) metricMap.set(label, { key, data: [] })
      metricMap.get(label).data.push({ ts, value: val })
    })
  })

  const seriesList = []
  const colorMap = { "温度": "#f56c6c", "湿度": "#409eff", "电池电量": "#67c23a", "转速": "#e6a23c", "电压": "#7b68ee", "功率": "#e6a23c" }
  const colors = ["#409eff", "#67c23a", "#e6a23c", "#f56c6c", "#7b68ee"]
  let i = 0
  for (const [label, { key, data }] of metricMap) {
    const unit = propUnit(key)
    seriesList.push({
      name: unit ? `${label}(${unit})` : label,
      type: "line",
      smooth: true,
      data: data.map(d => [d.ts, d.value]),
      lineStyle: { width: 2 },
      itemStyle: { color: colorMap[label] || colors[i % colors.length] }
    })
    i++
  }

  chart.setOption({
    tooltip: { trigger: "axis", formatter: (params) => {
      let html = new Date(params[0].value[0]).toLocaleString("zh-CN") + "<br/>"
      params.forEach(p => { html += `${p.seriesName}: ${p.value[1]}<br/>` })
      return html
    }},
    legend: { data: seriesList.map(s => s.name), bottom: 0 },
    grid: { left: 60, right: 20, top: 20, bottom: 50 },
    xAxis: { type: "time" },
    yAxis: { type: "value", name: "数值" },
    series: seriesList
  })
}

watch(selectedDevice, () => { stopAutoRefresh(); if (selectedDevice.value) loadData() })
watch(timeRange, () => { if (selectedDevice.value) loadData() })
</script>
