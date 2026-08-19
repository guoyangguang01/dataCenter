<template>
  <div>
    <el-card>
      <template #header>
        <div style="display: flex; justify-content: space-between; align-items: center">
          <span>规则引擎</span>
          <div>
            <el-select v-model="domainFilter" placeholder="选择域" clearable style="margin-right: 10px; width: 160px" @change="loadRules">
              <el-option v-for="d in domains" :key="d.id" :label="d.name" :value="d.id" />
            </el-select>
            <el-button type="primary" @click="openAdd">添加规则</el-button>
          </div>
        </div>
      </template>
      <el-table :data="rules" stripe v-loading="loading">
        <el-table-column prop="id" label="规则ID" width="180" />
        <el-table-column prop="name" label="名称" />
        <el-table-column prop="topic" label="Topic" width="200" />
        <el-table-column prop="domain_id" label="业务域" width="120" />
        <el-table-column label="节点数" width="80">
          <template #default="{ row }">{{ parseChain(row.chain).length }}</template>
        </el-table-column>
        <el-table-column label="状态" width="100">
          <template #default="{ row }">
            <el-switch v-model="row.enabled" @change="handleToggle(row)" />
          </template>
        </el-table-column>
        <el-table-column label="操作" width="150">
          <template #default="{ row }">
            <el-button size="small" type="danger" @click="handleDelete(row.id)">删除</el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-card>

    <!-- 添加/编辑规则对话框 -->
    <el-dialog v-model="showDialog" :title="editingRule ? '编辑规则' : '添加规则'" width="650px">
      <el-form :model="form" label-width="80px">
        <el-form-item label="名称">
          <el-input v-model="form.name" />
        </el-form-item>
        <el-form-item label="业务域">
          <el-input v-model="form.domain_id" />
        </el-form-item>
        <el-form-item label="Topic">
          <el-input v-model="form.topic" placeholder="NATS subject" />
        </el-form-item>
        <el-form-item label="启用">
          <el-switch v-model="form.enabled" />
        </el-form-item>
        <el-divider>节点链</el-divider>
        <div v-for="(node, i) in form.chain" :key="i" style="border: 1px solid #eee; padding: 10px; margin-bottom: 10px; border-radius: 4px">
          <div style="display: flex; gap: 8px; margin-bottom: 8px">
            <el-input v-model="node.id" placeholder="节点ID" style="width: 120px" />
            <el-select v-model="node.type" placeholder="类型" style="width: 120px">
              <el-option label="filter" value="filter" />
              <el-option label="transform" value="transform" />
              <el-option label="condition" value="condition" />
              <el-option label="aggregate" value="aggregate" />
              <el-option label="script" value="script" />
            </el-select>
            <el-button type="danger" :icon="Delete" circle @click="form.chain.splice(i, 1)" />
          </div>
          <div v-if="node.type === 'filter'" style="display: flex; gap: 8px">
            <el-input v-model="node.config.field" placeholder="field" style="width: 100px" />
            <el-select v-model="node.config.operator" placeholder="op" style="width: 100px">
              <el-option label="eq" value="eq" />
              <el-option label="contains" value="contains" />
              <el-option label="prefix" value="prefix" />
            </el-select>
            <el-input v-model="node.config.value" placeholder="value" style="width: 120px" />
          </div>
          <div v-else-if="node.type === 'condition'">
            <el-input v-model="node.config.expression" placeholder="表达式" />
          </div>
          <div v-else-if="node.type === 'aggregate'" style="display: flex; gap: 8px">
            <el-input-number v-model="node.config.window_size" :min="1" placeholder="窗口大小" />
            <el-select v-model="node.config.function" placeholder="函数" style="width: 120px">
              <el-option label="avg" value="avg" />
              <el-option label="sum" value="sum" />
              <el-option label="min" value="min" />
              <el-option label="max" value="max" />
              <el-option label="count" value="count" />
            </el-select>
          </div>
          <div v-else-if="node.type === 'script'">
            <el-input v-model="node.config.script" type="textarea" :rows="3" placeholder="JavaScript 脚本" />
          </div>
        </div>
        <el-button size="small" @click="addNode">+ 添加节点</el-button>
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
import { ruleApi, domainApi } from "../api"
import { ElMessage, ElMessageBox } from "element-plus"
import { Delete } from "@element-plus/icons-vue"

const rules = ref([])
const domains = ref([])
const loading = ref(false)
const showDialog = ref(false)
const editingRule = ref(null)
const domainFilter = ref("")
const form = ref({ name: "", domain_id: "", topic: "", enabled: true, chain: [] })

const parseChain = (chain) => {
  if (!chain) return []
  if (Array.isArray(chain)) return chain
  try { return JSON.parse(chain) } catch { return [] }
}

const loadRules = async () => {
  loading.value = true
  try {
    const res = await ruleApi.list(domainFilter.value)
    rules.value = (res.data.data || []).map(r => ({ ...r, chain: parseChain(r.chain) }))
  } finally { loading.value = false }
}

onMounted(async () => {
  const dRes = await domainApi.list(); domains.value = dRes.data.data || []
  await loadRules()
})

const openAdd = () => {
  editingRule.value = null
  form.value = { name: "", domain_id: "", topic: "", enabled: true, chain: [] }
  showDialog.value = true
}

const addNode = () => {
  form.value.chain.push({ id: `node_${Date.now()}`, type: "filter", config: { field: "topic", operator: "eq", value: "" } })
}

const handleSave = async () => {
  const payload = { ...form.value, chain: form.value.chain.map(n => ({ ...n, config: { ...n.config } })) }
  if (editingRule.value) {
    await ruleApi.update(editingRule.value.id, payload)
  } else {
    await ruleApi.create(payload)
  }
  showDialog.value = false
  await loadRules()
  ElMessage.success(editingRule.value ? "规则更新成功" : "规则创建成功")
}

const handleToggle = async (row) => {
  await ruleApi.toggle(row.id)
  ElMessage.success(row.enabled ? "规则已启用" : "规则已禁用")
}

const handleDelete = async (id) => {
  await ElMessageBox.confirm("确定删除该规则?", "提示", { type: "warning" })
  await ruleApi.delete(id)
  rules.value = rules.value.filter(r => r.id !== id)
  ElMessage.success("删除成功")
}
</script>
