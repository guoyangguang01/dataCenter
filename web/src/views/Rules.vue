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
        <el-table-column label="操作" width="180">
          <template #default="{ row }">
            <el-button size="small" @click="openEdit(row)">编辑</el-button>
            <el-button size="small" type="danger" @click="handleDelete(row.id)">删除</el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-card>

    <!-- 添加/编辑规则对话框 -->
    <el-dialog v-model="showDialog" :title="editingRule ? '编辑规则' : '添加规则'" width="900px">
      <el-form ref="formRef" :model="form" :rules="formRules" label-width="80px">
        <el-form-item label="名称" prop="name">
          <el-input v-model="form.name" />
        </el-form-item>
        <el-form-item label="业务域" prop="domain_id">
          <el-input v-model="form.domain_id" />
        </el-form-item>
        <el-form-item label="Topic">
          <el-input v-model="form.topic" placeholder="NATS subject" />
        </el-form-item>
        <el-form-item label="启用">
          <el-switch v-model="form.enabled" />
        </el-form-item>
      </el-form>

      <el-tabs v-model="editorMode" type="border-card">
        <!-- 表单模式 -->
        <el-tab-pane label="表单模式" name="form">
          <div style="max-height: 400px; overflow-y: auto; padding: 4px 0">
            <div v-for="(node, i) in form.chain" :key="i" style="border: 1px solid #eee; margin-bottom: 10px; border-radius: 4px; overflow: hidden">
              <div style="display: flex; align-items: center; gap: 8px; padding: 8px 10px; background: #fafafa; cursor: pointer" @click="toggleNode(i)">
                <el-icon style="transition: transform 0.2s" :style="{ transform: collapsedNodes.has(i) ? 'rotate(-90deg)' : 'rotate(0deg)' }"><ArrowDown /></el-icon>
                <el-tag size="small" :type="nodeTagType(node.type)">{{ node.type }}</el-tag>
                <span style="font-size: 13px; color: #666; flex: 1">{{ node.id }}</span>
                <span style="font-size: 12px; color: #999">{{ nodeSummary(node) }}</span>
                <el-button size="small" :disabled="i === 0" @click.stop="moveNode(i, -1)" title="上移">↑</el-button>
                <el-button size="small" :disabled="i === form.chain.length - 1" @click.stop="moveNode(i, 1)" title="下移">↓</el-button>
                <el-button size="small" @click.stop="copyNode(i)" title="复制">📋</el-button>
                <el-button size="small" type="danger" :icon="Delete" circle @click.stop="form.chain.splice(i, 1)" />
              </div>
              <div v-show="!collapsedNodes.has(i)" style="padding: 10px">
                <div style="display: flex; gap: 8px; margin-bottom: 8px">
                  <el-input v-model="node.id" placeholder="节点ID" style="width: 140px" />
                  <el-select v-model="node.type" placeholder="类型" style="width: 130px" @change="onNodeTypeChange(node)">
                    <el-option label="filter" value="filter" />
                    <el-option label="transform" value="transform" />
                    <el-option label="condition" value="condition" />
                    <el-option label="aggregate" value="aggregate" />
                    <el-option label="action" value="action" />
                    <el-option label="script" value="script" />
                  </el-select>
                </div>
                <div v-if="node.type === 'filter'" style="display: flex; gap: 8px">
                  <el-input v-model="node.config.field" placeholder="field" style="width: 100px" />
                  <el-select v-model="node.config.operator" placeholder="op" style="width: 100px">
                    <el-option label="eq" value="eq" />
                    <el-option label="contains" value="contains" />
                    <el-option label="prefix" value="prefix" />
                  </el-select>
                  <el-input v-model="node.config.value" placeholder="value" style="width: 160px" />
                </div>
                <div v-else-if="node.type === 'transform'" style="display: flex; gap: 8px">
                  <el-input v-model="node.config.extract" placeholder="extract topic" style="width: 220px" />
                  <el-input v-model="node.config.parse" placeholder="parse（预留）" style="width: 160px" />
                </div>
                <div v-else-if="node.type === 'condition'">
                  <el-input v-model="node.config.expression" placeholder="表达式，如 temp > 30" />
                  <div style="display: flex; gap: 8px; margin-top: 8px">
                    <el-input v-model="node.config.true_branch" placeholder="true_branch（可选）" style="flex: 1" />
                    <el-input v-model="node.config.false_branch" placeholder="false_branch（可选）" style="flex: 1" />
                  </div>
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
                  <el-input v-model="node.config.script" type="textarea" :rows="4" placeholder="JavaScript 脚本" />
                  <el-input-number v-model="node.config.timeout_ms" :min="1000" :max="30000" placeholder="超时(ms)" style="margin-top: 8px" />
                </div>
                <div v-else-if="node.type === 'action'">
                  <el-select v-model="node.config.type" placeholder="动作类型" style="width: 140px; margin-bottom: 8px">
                    <el-option label="publish" value="publish" />
                    <el-option label="webhook" value="webhook" />
                  </el-select>
                  <el-input v-model="node.config.topic_template" placeholder="topic_template" style="margin-bottom: 8px" />
                  <el-input v-model="node.config.payload_template_str" type="textarea" :rows="2" placeholder='payload_template JSON, 如 {"alert": "high temp"}' />
                </div>
              </div>
            </div>
            <el-button size="small" @click="addNode">+ 添加节点</el-button>
            <el-button size="small" @click="exportJson">导出 JSON</el-button>
            <el-button size="small" @click="showImportJson = true">导入 JSON</el-button>
          </div>
        </el-tab-pane>

        <!-- 画布模式 -->
        <el-tab-pane label="画布模式" name="canvas">
          <RuleCanvas :chain="form.chain" @update:chain="onCanvasChainUpdate" />
        </el-tab-pane>
      </el-tabs>

      <template #footer>
        <el-button @click="showDialog = false">取消</el-button>
        <el-button type="primary" @click="handleSave">确定</el-button>
      </template>
    </el-dialog>

    <!-- 导入 JSON 对话框 -->
    <el-dialog v-model="showImportJson" title="导入节点链 JSON" width="600px">
      <el-input v-model="importJsonText" type="textarea" :rows="12" placeholder='粘贴节点链 JSON 数组，如 [{"id":"f1","type":"filter","config":{"field":"topic","operator":"eq","value":"temp"}}]' />
      <template #footer>
        <el-button @click="showImportJson = false">取消</el-button>
        <el-button type="primary" @click="importJson">导入</el-button>
      </template>
    </el-dialog>

    <!-- 导出 JSON 对话框 -->
    <el-dialog v-model="showExportJson" title="导出节点链 JSON" width="600px">
      <el-input v-model="exportJsonText" type="textarea" :rows="12" readonly />
      <template #footer>
        <el-button type="primary" @click="copyExportJson">复制</el-button>
        <el-button @click="showExportJson = false">关闭</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, reactive, watch, onMounted } from "vue"
import { ruleApi } from "../api"
import { useAppStore } from "../stores/app"
import { ElMessage, ElMessageBox } from "element-plus"
import { Delete, ArrowDown } from "@element-plus/icons-vue"
import RuleCanvas from "../components/flow/RuleCanvas.vue"

const store = useAppStore()
const rules = ref([])
const domains = ref([])
const loading = ref(false)
const showDialog = ref(false)
const editingRule = ref(null)
const formRef = ref(null)

const formRules = {
  name: [
    { required: true, message: "请输入规则名称", trigger: "blur" },
    { min: 2, max: 64, message: "长度 2-64 个字符", trigger: "blur" }
  ],
  domain_id: [{ required: true, message: "请输入业务域", trigger: "blur" }]
}
const domainFilter = ref(store.currentDomain)
const editorMode = ref("form")
const collapsedNodes = reactive(new Set())
const showImportJson = ref(false)
const importJsonText = ref("")
const showExportJson = ref(false)
const exportJsonText = ref("")
const form = ref({ name: "", domain_id: "", topic: "", enabled: true, chain: [] })

const parseChain = (chain) => {
  if (!chain) return []
  if (Array.isArray(chain)) return chain
  try { return JSON.parse(chain) } catch { return [] }
}

const nodeTagType = (type) => ({
  filter: "", transform: "success", condition: "warning", aggregate: "info", action: "danger", script: "primary"
}[type] || "")

const nodeSummary = (node) => {
  const c = node.config || {}
  if (node.type === "filter") return `${c.field || "?"} ${c.operator || "?"} ${c.value || "?"}`
  if (node.type === "transform") return `extract: ${c.extract || "?"}`
  if (node.type === "condition") return c.expression || "未配置"
  if (node.type === "aggregate") return `${c.function || "avg"}(window=${c.window_size || 10})`
  if (node.type === "action") return `${c.type || "?"} → ${c.topic_template || "?"}`
  if (node.type === "script") return (c.script || "").substring(0, 40) + ((c.script || "").length > 40 ? "..." : "")
  return ""
}

const toggleNode = (i) => {
  if (collapsedNodes.has(i)) collapsedNodes.delete(i)
  else collapsedNodes.add(i)
}

const moveNode = (i, dir) => {
  const chain = form.value.chain
  const j = i + dir
  if (j < 0 || j >= chain.length) return
  ;[chain[i], chain[j]] = [chain[j], chain[i]]
}

const copyNode = (i) => {
  const src = form.value.chain[i]
  const copy = { ...src, id: `${src.id}_copy`, config: { ...src.config } }
  form.value.chain.splice(i + 1, 0, copy)
}

const exportJson = () => {
  exportJsonText.value = JSON.stringify(form.value.chain, null, 2)
  showExportJson.value = true
}

const copyExportJson = () => {
  navigator.clipboard.writeText(exportJsonText.value)
  ElMessage.success("已复制到剪贴板")
}

const importJson = () => {
  try {
    const parsed = JSON.parse(importJsonText.value)
    if (!Array.isArray(parsed)) throw new Error("必须是 JSON 数组")
    form.value.chain = parsed.map(n => {
      const config = { ...n.config }
      if (n.type === "action" && config.payload_template && typeof config.payload_template === "object") {
        config.payload_template_str = JSON.stringify(config.payload_template)
      }
      return { id: n.id || `node_${Date.now()}`, type: n.type || "filter", config }
    })
    collapsedNodes.clear()
    showImportJson.value = false
    importJsonText.value = ""
    ElMessage.success(`已导入 ${parsed.length} 个节点`)
  } catch (e) {
    ElMessage.error("JSON 解析失败: " + e.message)
  }
}

const loadRules = async () => {
  loading.value = true
  try {
    const res = await ruleApi.list(domainFilter.value)
    rules.value = (res.data.data || []).map(r => ({ ...r, chain: parseChain(r.chain) }))
  } finally { loading.value = false }
}

onMounted(async () => {
  domains.value = store.domains
  await loadRules()
})

watch(() => store.currentDomain, (val) => {
  domainFilter.value = val
  loadRules()
})

const openAdd = () => {
  editingRule.value = null
  form.value = { name: "", domain_id: "", topic: "", enabled: true, chain: [] }
  showDialog.value = true
}

const openEdit = (row) => {
  editingRule.value = row
  const chain = (Array.isArray(row.chain) ? row.chain : parseChain(row.chain)).map(n => {
    const config = { ...n.config }
    if (n.type === "action" && config.payload_template && typeof config.payload_template === "object") {
      config.payload_template_str = JSON.stringify(config.payload_template)
    }
    return { ...n, config }
  })
  form.value = { name: row.name, domain_id: row.domain_id, topic: row.topic || "", enabled: row.enabled, chain }
  showDialog.value = true
}

const defaultConfig = (type) => {
  const configs = {
    filter: { field: "topic", operator: "eq", value: "" },
    transform: { extract: "", parse: "" },
    condition: { expression: "", true_branch: "", false_branch: "" },
    aggregate: { window_size: 10, function: "avg" },
    action: { type: "publish", topic_template: "", payload_template_str: "{}" },
    script: { script: "", timeout_ms: 5000 }
  }
  return { ...(configs[type] || {}) }
}

const onNodeTypeChange = (node) => {
  node.config = defaultConfig(node.type)
}

const addNode = () => {
  form.value.chain.push({ id: `node_${Date.now()}`, type: "filter", config: defaultConfig("filter") })
}

const onCanvasChainUpdate = (newChain) => {
  form.value.chain = newChain
}

const handleSave = async () => {
  if (formRef.value) await formRef.value.validate()
  if (form.value.chain.length === 0) {
    ElMessage.warning("请至少添加一个节点")
    return
  }
  for (const node of form.value.chain) {
    if (!node.id) { ElMessage.warning("所有节点必须填写节点ID"); return }
  }
  const payload = {
    ...form.value,
    chain: form.value.chain.map(n => {
      const config = { ...n.config }
      if (n.type === "action" && config.payload_template_str) {
        try { config.payload_template = JSON.parse(config.payload_template_str) } catch { config.payload_template = {} }
        delete config.payload_template_str
      }
      return { ...n, config }
    })
  }
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
