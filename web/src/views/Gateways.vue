<template>
  <div>
    <el-card>
      <template #header>
        <div style="display: flex; justify-content: space-between; align-items: center">
          <span>网关管理</span>
          <el-button type="primary" @click="openAdd">添加网关</el-button>
        </div>
      </template>
      <el-table :data="gateways" stripe v-loading="loading">
        <el-table-column prop="name" label="名称" />
        <el-table-column prop="type" label="类型" width="100">
          <template #default="{ row }">
            <el-tag :type="typeTag(row.type)">{{ row.type.toUpperCase() }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="配置" min-width="250">
          <template #default="{ row }">
            <span style="font-size: 12px; color: #666">{{ formatConfig(row) }}</span>
          </template>
        </el-table-column>
        <el-table-column label="状态" width="100">
          <template #default="{ row }">
            <el-tag :type="row.status === 'running' ? 'success' : row.status === 'error' ? 'danger' : 'info'" size="small">
              {{ row.status === 'running' ? '运行中' : row.status === 'error' ? '异常' : '已停止' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="220">
          <template #default="{ row }">
            <el-button v-if="row.status !== 'running'" size="small" type="success" @click="handleStart(row.id)">启动</el-button>
            <el-button v-else size="small" type="warning" @click="handleStop(row.id)">停止</el-button>
            <el-button size="small" @click="openEdit(row)">编辑</el-button>
            <el-button size="small" type="danger" @click="handleDelete(row.id)">删除</el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-card>

    <!-- 添加/编辑网关对话框 -->
    <el-dialog v-model="showDialog" :title="editing ? '编辑网关' : '添加网关'" width="550px">
      <el-form :model="form" label-width="100px">
        <el-form-item label="名称">
          <el-input v-model="form.name" />
        </el-form-item>
        <el-form-item label="类型">
          <el-select v-model="form.type" style="width: 100%" @change="onTypeChange">
            <el-option label="MQTT" value="mqtt" />
            <el-option label="TCP" value="tcp" />
            <el-option label="Modbus" value="modbus" />
            <el-option label="OPC UA" value="opcua" />
          </el-select>
        </el-form-item>

        <!-- MQTT 配置 -->
        <template v-if="form.type === 'mqtt'">
          <el-form-item label="端口">
            <el-input-number v-model="mqttConfig.port" :min="1" :max="65535" />
          </el-form-item>
          <el-form-item label="最大连接">
            <el-input-number v-model="mqttConfig.max_connection" :min="1" />
          </el-form-item>
          <el-form-item label="KeepAlive(s)">
            <el-input-number v-model="mqttConfig.keep_alive" :min="10" />
          </el-form-item>
        </template>

        <!-- TCP 配置 -->
        <template v-if="form.type === 'tcp'">
          <el-form-item label="端口">
            <el-input-number v-model="tcpConfig.port" :min="1" :max="65535" />
          </el-form-item>
          <el-form-item label="最大连接">
            <el-input-number v-model="tcpConfig.max_connection" :min="1" />
          </el-form-item>
          <el-form-item label="心跳(s)">
            <el-input-number v-model="tcpConfig.heartbeat" :min="5" />
          </el-form-item>
        </template>

        <!-- Modbus 配置 -->
        <template v-if="form.type === 'modbus'">
          <el-form-item label="端口">
            <el-input-number v-model="modbusConfig.port" :min="1" :max="65535" />
          </el-form-item>
          <el-form-item label="轮询间隔(s)">
            <el-input-number v-model="modbusConfig.poll_interval" :min="1" />
          </el-form-item>
          <el-form-item label="Slave IDs">
            <el-input v-model="modbusConfig.slave_ids_str" placeholder="逗号分隔，如 1,2,3" />
          </el-form-item>
        </template>

        <!-- OPC UA 配置 -->
        <template v-if="form.type === 'opcua'">
          <el-form-item label="Endpoint">
            <el-input v-model="opcuaConfig.endpoint" placeholder="opc.tcp://192.168.1.100:4840" />
          </el-form-item>
          <el-form-item label="轮询间隔(s)">
            <el-input-number v-model="opcuaConfig.poll_interval" :min="1" />
          </el-form-item>
          <el-form-item label="Node IDs">
            <el-input v-model="opcuaConfig.node_ids_str" placeholder="逗号分隔，如 ns=2;s=Temp,ns=2;s=Humidity" />
          </el-form-item>
          <el-form-item label="设备ID">
            <el-input v-model="opcuaConfig.device_id" />
          </el-form-item>
          <el-form-item label="业务域">
            <el-input v-model="opcuaConfig.domain_id" />
          </el-form-item>
        </template>
      </el-form>
      <template #footer>
        <el-button @click="showDialog = false">取消</el-button>
        <el-button type="primary" @click="handleSave">确定</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, onMounted } from "vue"
import { gatewayApi } from "../api"
import { ElMessage, ElMessageBox } from "element-plus"

const gateways = ref([])
const loading = ref(false)
const showDialog = ref(false)
const editing = ref(null)

const form = ref({ name: "", type: "mqtt" })
const mqttConfig = ref({ port: 1883, max_connection: 100, keep_alive: 60 })
const tcpConfig = ref({ port: 9000, max_connection: 100, heartbeat: 30 })
const modbusConfig = ref({ port: 502, poll_interval: 10, slave_ids_str: "1" })
const opcuaConfig = ref({ endpoint: "opc.tcp://localhost:4840", poll_interval: 5, node_ids_str: "ns=2;s=Temperature", device_id: "opcua-001", domain_id: "default" })

const typeTag = (t) => ({ mqtt: "", tcp: "success", modbus: "warning", opcua: "danger" }[t] || "info")

const formatConfig = (row) => {
  try {
    const cfg = JSON.parse(row.config)
    if (row.type === "mqtt") return `端口:${cfg.port} 最大连接:${cfg.max_connection}`
    if (row.type === "tcp") return `端口:${cfg.port} 心跳:${cfg.heartbeat}s`
    if (row.type === "modbus") return `端口:${cfg.port} 轮询:${cfg.poll_interval}s`
    if (row.type === "opcua") return `${cfg.endpoint} 轮询:${cfg.poll_interval}s`
  } catch { return row.config }
}

const loadGateways = async () => {
  loading.value = true
  try { const res = await gatewayApi.list(); gateways.value = res.data.data || [] }
  finally { loading.value = false }
}

onMounted(loadGateways)

const onTypeChange = (type) => {
  if (type === 'mqtt') mqttConfig.value = { port: 1883, max_connection: 100, keep_alive: 60 }
  if (type === 'tcp') tcpConfig.value = { port: 9000, max_connection: 100, heartbeat: 30 }
  if (type === 'modbus') modbusConfig.value = { port: 502, poll_interval: 10, slave_ids_str: "1" }
  if (type === 'opcua') opcuaConfig.value = { endpoint: "opc.tcp://localhost:4840", poll_interval: 5, node_ids_str: "ns=2;s=Temperature", device_id: "opcua-001", domain_id: "default" }
}

const openAdd = () => {
  editing.value = null
  form.value = { name: "", type: "mqtt" }
  mqttConfig.value = { port: 1883, max_connection: 100, keep_alive: 60 }
  tcpConfig.value = { port: 9000, max_connection: 100, heartbeat: 30 }
  modbusConfig.value = { port: 502, poll_interval: 10, slave_ids_str: "1" }
  opcuaConfig.value = { endpoint: "opc.tcp://localhost:4840", poll_interval: 5, node_ids_str: "ns=2;s=Temperature", device_id: "opcua-001", domain_id: "default" }
  showDialog.value = true
}

const openEdit = (row) => {
  editing.value = row
  form.value = { name: row.name, type: row.type }
  try {
    const cfg = JSON.parse(row.config)
    if (row.type === "mqtt") mqttConfig.value = { ...cfg }
    if (row.type === "tcp") tcpConfig.value = { ...cfg }
    if (row.type === "modbus") modbusConfig.value = { ...cfg, slave_ids_str: (cfg.slave_ids || []).join(",") }
    if (row.type === "opcua") opcuaConfig.value = { ...cfg, node_ids_str: (cfg.node_ids || []).join(",") }
  } catch {}
  showDialog.value = true
}

const buildConfig = () => {
  if (form.value.type === "mqtt") return { ...mqttConfig.value }
  if (form.value.type === "tcp") return { ...tcpConfig.value }
  if (form.value.type === "modbus") return { ...modbusConfig.value, slave_ids: modbusConfig.value.slave_ids_str.split(",").map(Number).filter(Boolean) }
  if (form.value.type === "opcua") return { ...opcuaConfig.value, node_ids: opcuaConfig.value.node_ids_str.split(",").map(s => s.trim()).filter(Boolean) }
}

const handleSave = async () => {
  const payload = { ...form.value, config: buildConfig() }
  if (editing.value) {
    await gatewayApi.update(editing.value.id, payload)
  } else {
    await gatewayApi.create(payload)
  }
  showDialog.value = false
  await loadGateways()
  ElMessage.success(editing.value ? "更新成功" : "创建成功")
}

const handleStart = async (id) => {
  try {
    await gatewayApi.start(id)
    ElMessage.success("网关已启动")
    await loadGateways()
  } catch (e) {
    ElMessage.error("启动失败: " + (e.response?.data?.error || e.message))
  }
}

const handleStop = async (id) => {
  try {
    await gatewayApi.stop(id)
    ElMessage.success("网关已停止")
    await loadGateways()
  } catch (e) {
    ElMessage.error("停止失败")
  }
}

const handleDelete = async (id) => {
  await ElMessageBox.confirm("确定删除该网关配置?", "提示", { type: "warning" })
  await gatewayApi.delete(id)
  gateways.value = gateways.value.filter(g => g.id !== id)
  ElMessage.success("删除成功")
}
</script>
