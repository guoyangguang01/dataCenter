<template>
  <div>
    <el-card>
      <template #header>
        <div style="display: flex; justify-content: space-between; align-items: center">
          <span>设备实时数据</span>
          <div style="display: flex; gap: 12px; align-items: center">
            <el-select v-model="selectedDevice" placeholder="选择设备" style="width: 240px" @change="loadData">
              <el-option v-for="d in devices" :key="d.id" :label="d.name + ' (' + d.id + ')'" :value="d.id" />
            </el-select>
            <el-select v-model="timeRange" style="width: 120px" @change="loadData">
              <el-option label="最近1小时" :value="1" />
              <el-option label="最近6小时" :value="6" />
              <el-option label="最近24小时" :value="24" />
              <el-option label="最近7天" :value="168" />
            </el-select>
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
          <el-col :span="6" v-for="(item, i) in latestData" :key="i">
            <el-card shadow="hover">
              <el-statistic :title="item.topic" :value="item.value" :suffix="item.unit" />
              <div style="font-size: 12px; color: #999; margin-top: 4px">{{ item.time }}</div>
            </el-card>
          </el-col>
          <el-col v-if="latestData.length === 0" :span="24">
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
import { ref, onMounted, nextTick, watch } from "vue"
import { deviceApi, dataApi } from "../api"
import * as echarts from "echarts"

const devices = ref([])
const selectedDevice = ref("")
const timeRange = ref(1)
const loading = ref(false)
const latestData = ref([])
const tableData = ref([])
const chartRef = ref(null)
let chart = null

onMounted(async () => {
  const res = await deviceApi.list({ page: 1, page_size: 100 })
  devices.value = res.data.data || []
})

const loadData = async () => {
  if (!selectedDevice.value) return
  loading.value = true
  try {
    // 加载最新数据
    const latestRes = await dataApi.getLatestData(selectedDevice.value, 20)
    const latestRows = latestRes.data.data?.rows || []
    const columns = latestRes.data.data?.columns || []

    // 解析最新数据
    latestData.value = latestRows.map(row => {
      const obj = {}
      columns.forEach((col, i) => { obj[col] = row[i] })
      return {
        topic: obj.topic || "-",
        value: obj.value ?? "-",
        unit: "",
        time: obj.ts ? new Date(obj.ts).toLocaleString("zh-CN") : "-"
      }
    }).slice(0, 4)

    // 加载历史数据
    const historyRes = await dataApi.getDeviceData(selectedDevice.value, timeRange.value)
    const historyRows = historyRes.data.data?.rows || []
    const historyColumns = historyRes.data.data?.columns || []

    // 解析表格数据
    tableData.value = historyRows.map(row => {
      const obj = {}
      historyColumns.forEach((col, i) => { obj[col] = row[i] })
      return {
        ts: obj.ts ? new Date(obj.ts).toLocaleString("zh-CN") : "-",
        topic: obj.topic || "-",
        value: obj.value ?? "-",
        quality: obj.quality ?? 0
      }
    }).reverse()

    // 渲染图表
    await nextTick()
    renderChart(historyRows, historyColumns)
  } catch (e) {
    console.error("加载数据失败:", e)
  } finally {
    loading.value = false
  }
}

const renderChart = (rows, columns) => {
  if (!chartRef.value || rows.length === 0) return

  if (chart) chart.dispose()
  chart = echarts.init(chartRef.value)

  // 按 topic 分组
  const topicMap = new Map()
  rows.forEach(row => {
    const obj = {}
    columns.forEach((col, i) => { obj[col] = row[i] })
    const topic = obj.topic || "unknown"
    if (!topicMap.has(topic)) topicMap.set(topic, [])
    topicMap.get(topic).push({
      ts: new Date(obj.ts).getTime(),
      value: parseFloat(obj.value) || 0
    })
  })

  const seriesList = []
  const colors = ["#409eff", "#67c23a", "#e6a23c", "#f56c6c", "#7b68ee"]
  let i = 0
  for (const [topic, data] of topicMap) {
    seriesList.push({
      name: topic,
      type: "line",
      smooth: true,
      data: data.map(d => [d.ts, d.value]),
      lineStyle: { width: 2 },
      itemStyle: { color: colors[i % colors.length] }
    })
    i++
  }

  chart.setOption({
    tooltip: { trigger: "axis", formatter: (params) => {
      let html = new Date(params[0].value[0]).toLocaleString("zh-CN") + "<br/>"
      params.forEach(p => { html += `${p.seriesName}: ${p.value[1]}<br/>` })
      return html
    }},
    legend: { data: Array.from(topicMap.keys()), bottom: 0 },
    grid: { left: 60, right: 20, top: 20, bottom: 50 },
    xAxis: { type: "time" },
    yAxis: { type: "value", name: "数值" },
    series: seriesList
  })
}

watch(timeRange, () => { if (selectedDevice.value) loadData() })
</script>
