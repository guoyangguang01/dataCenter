<template>
  <div>
    <el-card>
      <template #header>
        <div style="display: flex; justify-content: space-between; align-items: center">
          <span>告警中心</span>
          <el-button type="primary" @click="openAdd">添加 Webhook</el-button>
        </div>
      </template>
      <el-tabs v-model="activeTab">
        <!-- Webhook 配置 -->
        <el-tab-pane label="Webhook 配置" name="webhooks">
          <el-table :data="webhooks" stripe v-loading="loading">
            <el-table-column prop="id" label="ID" width="180" />
            <el-table-column prop="name" label="名称" />
            <el-table-column prop="url" label="URL" />
            <el-table-column prop="method" label="方法" width="80" />
            <el-table-column prop="domain_id" label="业务域" width="120" />
            <el-table-column label="操作" width="240">
              <template #default="{ row }">
                <el-button size="small" @click="openEdit(row)">编辑</el-button>
                <el-button size="small" type="success" @click="handleTest(row.id)">测试</el-button>
                <el-button size="small" type="danger" @click="handleDeleteWebhook(row.id)">删除</el-button>
              </template>
            </el-table-column>
          </el-table>
        </el-tab-pane>

        <!-- 告警日志 -->
        <el-tab-pane label="告警日志" name="logs">
          <el-table :data="logs" stripe v-loading="logsLoading">
            <el-table-column prop="alert_id" label="告警ID" width="180" />
            <el-table-column prop="webhook_id" label="Webhook" width="180" />
            <el-table-column prop="status" label="状态" width="100">
              <template #default="{ row }">
                <el-tag :type="row.status === 'sent' ? 'success' : row.status === 'rate_limited' ? 'warning' : 'danger'">
                  {{ row.status }}
                </el-tag>
              </template>
            </el-table-column>
            <el-table-column prop="response" label="响应" show-overflow-tooltip />
            <el-table-column prop="created_at" label="时间" width="180">
              <template #default="{ row }">{{ formatTime(row.created_at) }}</template>
            </el-table-column>
          </el-table>
        </el-tab-pane>
      </el-tabs>
    </el-card>

    <!-- 添加/编辑 Webhook 对话框 -->
    <el-dialog v-model="showDialog" :title="editingWebhook ? '编辑 Webhook' : '添加 Webhook'" width="500px">
      <el-form :model="form" label-width="100px">
        <el-form-item label="名称">
          <el-input v-model="form.name" />
        </el-form-item>
        <el-form-item label="业务域">
          <el-input v-model="form.domain_id" />
        </el-form-item>
        <el-form-item label="URL">
          <el-input v-model="form.url" />
        </el-form-item>
        <el-form-item label="请求方法">
          <el-select v-model="form.method" style="width: 100%">
            <el-option label="POST" value="POST" />
            <el-option label="GET" value="GET" />
          </el-select>
        </el-form-item>
        <el-form-item label="告警级别">
          <el-checkbox-group v-model="selectedLevels">
            <el-checkbox label="critical">严重</el-checkbox>
            <el-checkbox label="warning">警告</el-checkbox>
            <el-checkbox label="info">信息</el-checkbox>
          </el-checkbox-group>
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
import { ref, onMounted } from "vue"
import { alertApi } from "../api"
import { ElMessage, ElMessageBox } from "element-plus"

const activeTab = ref("webhooks")
const webhooks = ref([])
const logs = ref([])
const loading = ref(false)
const logsLoading = ref(false)
const showDialog = ref(false)
const editingWebhook = ref(null)
const selectedLevels = ref(["critical", "warning"])
const form = ref({ name: "", domain_id: "", url: "", method: "POST", filter: { levels: [], device_types: [] }, rate_limit: { max_per_minute: 10, dedup_window: 300 } })

const formatTime = (t) => t ? new Date(t).toLocaleString("zh-CN") : ""

const loadWebhooks = async () => {
  loading.value = true
  try { const res = await alertApi.listWebhooks(); webhooks.value = res.data.data || [] }
  finally { loading.value = false }
}

const loadLogs = async () => {
  logsLoading.value = true
  try { const res = await alertApi.listLogs(); logs.value = res.data.data || [] }
  finally { logsLoading.value = false }
}

onMounted(async () => {
  await loadWebhooks()
  await loadLogs()
})

const openAdd = () => {
  editingWebhook.value = null
  form.value = { name: "", domain_id: "", url: "", method: "POST", filter: { levels: [], device_types: [] }, rate_limit: { max_per_minute: 10, dedup_window: 300 } }
  selectedLevels.value = ["critical", "warning"]
  showDialog.value = true
}

const openEdit = (row) => {
  editingWebhook.value = row
  form.value = {
    name: row.name,
    domain_id: row.domain_id || "",
    url: row.url,
    method: row.method || "POST",
    filter: { ...(row.filter || {}), levels: (row.filter?.levels || []), device_types: (row.filter?.device_types || []) },
    rate_limit: { ...(row.rate_limit || { max_per_minute: 10, dedup_window: 300 }) }
  }
  selectedLevels.value = [...(row.filter?.levels || [])]
  showDialog.value = true
}

const handleSave = async () => {
  form.value.filter.levels = selectedLevels.value
  if (editingWebhook.value) {
    await alertApi.updateWebhook(editingWebhook.value.id, form.value)
  } else {
    await alertApi.createWebhook(form.value)
  }
  showDialog.value = false
  await loadWebhooks()
  ElMessage.success(editingWebhook.value ? "Webhook 更新成功" : "Webhook 创建成功")
}

const handleTest = async (id) => {
  try {
    await alertApi.testWebhook(id)
    ElMessage.success("测试发送成功")
  } catch {
    ElMessage.error("测试发送失败")
  }
}

const handleDeleteWebhook = async (id) => {
  await ElMessageBox.confirm("确定删除该 Webhook?", "提示", { type: "warning" })
  await alertApi.deleteWebhook(id)
  webhooks.value = webhooks.value.filter(w => w.id !== id)
  ElMessage.success("删除成功")
}
</script>
