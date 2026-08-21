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
        <el-table-column label="操作" width="180">
          <template #default="{ row }">
            <el-button size="small" @click="openEdit(row)">编辑</el-button>
            <el-button size="small" type="danger" @click="handleDelete(row.id)">删除</el-button>
          </template>
        </el-table-column>
      </el-table>
      <div style="display: flex; justify-content: flex-end; margin-top: 16px">
        <el-pagination
          v-model:current-page="page"
          v-model:page-size="pageSize"
          :page-sizes="[10, 20, 50]"
          :total="total"
          layout="total, sizes, prev, pager, next, jumper"
          @size-change="loadDevices"
          @current-change="loadDevices"
        />
      </div>
    </el-card>

    <!-- 添加/编辑设备对话框 -->
    <el-dialog v-model="showDialog" :title="editing ? '编辑设备' : '添加设备'" width="500px" :before-close="handleBeforeClose">
      <el-form ref="formRef" :model="form" :rules="formRules" label-width="80px">
        <el-form-item label="设备ID" prop="id">
          <el-input v-model="form.id" placeholder="如 sensor-001" :disabled="!!editing" />
        </el-form-item>
        <el-form-item label="名称" prop="name">
          <el-input v-model="form.name" placeholder="如 车间A温度传感器" />
        </el-form-item>
        <el-form-item label="设备类型" prop="device_type">
          <el-select v-model="form.device_type" style="width: 100%">
            <el-option label="sensor" value="sensor" />
            <el-option label="actuator" value="actuator" />
            <el-option label="gateway" value="gateway" />
            <el-option label="plc" value="plc" />
          </el-select>
        </el-form-item>
        <el-form-item label="协议" prop="protocol">
          <el-select v-model="form.protocol" style="width: 100%" :disabled="!!editing">
            <el-option label="MQTT" value="mqtt" />
            <el-option label="TCP" value="tcp" />
            <el-option label="Modbus" value="modbus" />
            <el-option label="OPC UA" value="opcua" />
          </el-select>
        </el-form-item>
        <el-form-item label="业务域" prop="domain_id">
          <el-input v-model="form.domain_id" placeholder="如 factory-a" />
        </el-form-item>
        <el-form-item label="地区">
          <el-input v-model="form.region" placeholder="如 cn-east（可选）" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="showDialog = false">取消</el-button>
        <el-button type="primary" @click="handleSave">确定</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, watch, onMounted } from "vue"
import { deviceApi } from "../api"
import { ElMessage, ElMessageBox } from "element-plus"

const devices = ref([])
const loading = ref(false)
const showDialog = ref(false)
const editing = ref(null)
const formRef = ref(null)
const formDirty = ref(false)
const form = ref({ id: "", name: "", device_type: "sensor", protocol: "mqtt", domain_id: "", region: "" })
const page = ref(1)
const pageSize = ref(20)
const total = ref(0)

const formRules = {
  id: [{ required: true, message: "请输入设备ID", trigger: "blur" }],
  name: [
    { required: true, message: "请输入设备名称", trigger: "blur" },
    { min: 2, max: 64, message: "长度 2-64 个字符", trigger: "blur" }
  ],
  device_type: [{ required: true, message: "请选择设备类型", trigger: "change" }],
  protocol: [{ required: true, message: "请选择协议", trigger: "change" }],
  domain_id: [{ required: true, message: "请输入业务域", trigger: "blur" }]
}

const loadDevices = async () => {
  loading.value = true
  try {
    const res = await deviceApi.list({ page: page.value, page_size: pageSize.value })
    devices.value = res.data.data || []
    total.value = res.data.total || 0
  } finally { loading.value = false }
}

onMounted(loadDevices)

const openAdd = () => {
  editing.value = null
  form.value = { id: "", name: "", device_type: "sensor", protocol: "mqtt", domain_id: "", region: "" }
  formDirty.value = false
  showDialog.value = true
}

const openEdit = (row) => {
  editing.value = row
  form.value = { id: row.id, name: row.name, device_type: row.device_type, protocol: row.protocol, domain_id: row.domain_id, region: row.region || "" }
  formDirty.value = false
  showDialog.value = true
}

const handleSave = async () => {
  if (!formRef.value) return
  await formRef.value.validate()
  if (editing.value) {
    const { id, protocol, ...updateData } = form.value
    await deviceApi.update(editing.value.id, updateData)
  } else {
    const res = await deviceApi.create(form.value)
    ElMessage.success("设备创建成功，Token: " + res.data.token.substring(0, 16) + "...")
  }
  showDialog.value = false
  page.value = 1
  await loadDevices()
  if (editing.value) ElMessage.success("设备更新成功")
}

const handleBeforeClose = (done) => {
  if (formDirty.value) {
    ElMessageBox.confirm("有未保存的更改，确定关闭？", "提示", { type: "warning" })
      .then(() => done())
      .catch(() => {})
  } else {
    done()
  }
}

watch(form, () => { if (showDialog.value) formDirty.value = true }, { deep: true })

const handleDelete = async (id) => {
  await ElMessageBox.confirm("确定删除该设备?", "提示", { type: "warning" })
  await deviceApi.delete(id)
  await loadDevices()
  ElMessage.success("删除成功")
}
</script>
