<template>
  <div>
    <el-card>
      <template #header>
        <div style="display: flex; justify-content: space-between; align-items: center">
          <span>设备列表</span>
          <el-button type="primary" @click="openAdd">添加设备</el-button>
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
        <el-table-column label="Token" width="200">
          <template #default="{ row }">
            <span style="font-size: 12px; color: #999" :title="row.token">{{ row.token ? row.token.substring(0, 16) + '...' : '-' }}</span>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="150">
          <template #default="{ row }">
            <el-button size="small" type="danger" @click="handleDelete(row.id)">删除</el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-card>

    <!-- 添加设备对话框 -->
    <el-dialog v-model="showAdd" title="添加设备" width="500px">
      <el-form :model="form" label-width="80px">
        <el-form-item label="设备ID">
          <el-input v-model="form.id" placeholder="如 sensor-001" />
        </el-form-item>
        <el-form-item label="名称">
          <el-input v-model="form.name" placeholder="如 车间A温度传感器" />
        </el-form-item>
        <el-form-item label="设备类型">
          <el-select v-model="form.device_type" style="width: 100%">
            <el-option label="sensor" value="sensor" />
            <el-option label="actuator" value="actuator" />
            <el-option label="gateway" value="gateway" />
            <el-option label="plc" value="plc" />
          </el-select>
        </el-form-item>
        <el-form-item label="协议">
          <el-select v-model="form.protocol" style="width: 100%">
            <el-option label="MQTT" value="mqtt" />
            <el-option label="TCP" value="tcp" />
            <el-option label="Modbus" value="modbus" />
            <el-option label="OPC UA" value="opcua" />
          </el-select>
        </el-form-item>
        <el-form-item label="业务域">
          <el-input v-model="form.domain_id" placeholder="如 factory-a" />
        </el-form-item>
        <el-form-item label="地区">
          <el-input v-model="form.region" placeholder="如 cn-east（可选）" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="showAdd = false">取消</el-button>
        <el-button type="primary" @click="handleCreate">确定</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, onMounted } from "vue"
import { deviceApi } from "../api"
import { ElMessage, ElMessageBox } from "element-plus"

const devices = ref([])
const loading = ref(false)
const showAdd = ref(false)
const form = ref({ id: "", name: "", device_type: "sensor", protocol: "mqtt", domain_id: "", region: "" })

onMounted(async () => {
  loading.value = true
  try {
    const res = await deviceApi.list({ page: 1, page_size: 20 })
    devices.value = res.data.data || []
  } finally { loading.value = false }
})

const openAdd = () => {
  form.value = { id: "", name: "", device_type: "sensor", protocol: "mqtt", domain_id: "", region: "" }
  showAdd.value = true
}

const handleCreate = async () => {
  if (!form.value.id || !form.value.name || !form.value.domain_id) {
    ElMessage.warning("请填写必填项")
    return
  }
  const res = await deviceApi.create(form.value)
  devices.value.unshift(res.data)
  showAdd.value = false
  ElMessage.success("设备创建成功，Token: " + res.data.token.substring(0, 16) + "...")
}

const handleDelete = async (id) => {
  await ElMessageBox.confirm("确定删除该设备?", "提示", { type: "warning" })
  await deviceApi.delete(id)
  devices.value = devices.value.filter(d => d.id !== id)
  ElMessage.success("删除成功")
}
</script>
