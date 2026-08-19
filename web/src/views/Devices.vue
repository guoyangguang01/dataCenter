<template>
  <div>
    <el-card>
      <template #header>
        <div style="display: flex; justify-content: space-between; align-items: center">
          <span>设备列表</span>
          <el-button type="primary" @click="showAdd = true">添加设备</el-button>
        </div>
      </template>
      <el-table :data="devices" stripe v-loading="loading">
        <el-table-column prop="id" label="设备ID" width="150" />
        <el-table-column prop="name" label="名称" />
        <el-table-column prop="device_type" label="类型" width="100" />
        <el-table-column prop="protocol" label="协议" width="100" />
        <el-table-column prop="domain_id" label="业务域" width="120" />
        <el-table-column label="状态" width="100">
          <template #default="{ row }">
            <el-tag :type="row.online ? 'success' : 'info'">{{ row.online ? '在线' : '离线' }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="150">
          <template #default="{ row }">
            <el-button size="small" type="danger" @click="handleDelete(row.id)">删除</el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-card>
  </div>
</template>

<script setup>
import { ref, onMounted } from "vue"
import { deviceApi } from "../api"
const devices = ref([])
const loading = ref(false)
const showAdd = ref(false)

onMounted(async () => {
  loading.value = true
  try {
    const res = await deviceApi.list({ page: 1, page_size: 20 })
    devices.value = res.data.data || []
  } finally { loading.value = false }
})

const handleDelete = async (id) => {
  await deviceApi.delete(id)
  devices.value = devices.value.filter(d => d.id !== id)
}
</script>
