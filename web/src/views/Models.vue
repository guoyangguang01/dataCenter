<template>
  <div>
    <el-card>
      <template #header>
        <div style="display: flex; justify-content: space-between; align-items: center">
          <span>物模型</span>
          <div>
            <el-select v-model="domainFilter" placeholder="选择域" clearable style="margin-right: 10px; width: 160px" @change="loadModels">
              <el-option v-for="d in domains" :key="d.id" :label="d.name" :value="d.id" />
            </el-select>
            <el-button type="primary" @click="openAdd">添加模型</el-button>
          </div>
        </div>
      </template>
      <el-table :data="models" stripe v-loading="loading">
        <el-table-column prop="id" label="模型ID" width="150" />
        <el-table-column prop="name" label="名称" />
        <el-table-column prop="domain_id" label="业务域" width="120" />
        <el-table-column label="属性数" width="80">
          <template #default="{ row }">{{ (row.properties || []).length }}</template>
        </el-table-column>
        <el-table-column label="命令数" width="80">
          <template #default="{ row }">{{ (row.commands || []).length }}</template>
        </el-table-column>
        <el-table-column label="事件数" width="80">
          <template #default="{ row }">{{ (row.events || []).length }}</template>
        </el-table-column>
        <el-table-column label="操作" width="180">
          <template #default="{ row }">
            <el-button size="small" @click="openEdit(row)">编辑</el-button>
            <el-button size="small" type="danger" @click="handleDelete(row.id)">删除</el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-card>

    <!-- 添加/编辑模型对话框 -->
    <el-dialog v-model="showDialog" :title="editingModel ? '编辑物模型' : '添加物模型'" width="600px">
      <el-form :model="form" label-width="80px">
        <el-form-item label="模型ID">
          <el-input v-model="form.id" :disabled="!!editingModel" />
        </el-form-item>
        <el-form-item label="名称">
          <el-input v-model="form.name" />
        </el-form-item>
        <el-form-item label="业务域">
          <el-input v-model="form.domain_id" />
        </el-form-item>
        <el-divider>属性定义</el-divider>
        <div v-for="(prop, i) in form.properties" :key="i" style="display: flex; gap: 8px; margin-bottom: 8px">
          <el-input v-model="prop.id" placeholder="属性ID" style="width: 100px" />
          <el-input v-model="prop.name" placeholder="名称" style="width: 100px" />
          <el-select v-model="prop.data_type" placeholder="类型" style="width: 100px">
            <el-option label="float" value="float" />
            <el-option label="int" value="int" />
            <el-option label="string" value="string" />
            <el-option label="bool" value="bool" />
            <el-option label="enum" value="enum" />
          </el-select>
          <el-input v-model="prop.unit" placeholder="单位" style="width: 80px" />
          <el-button type="danger" :icon="Delete" circle @click="form.properties.splice(i, 1)" />
        </div>
        <el-button size="small" @click="form.properties.push({ id: '', name: '', data_type: 'float', unit: '', range: [0, 100], required: false, access_mode: 'r' })">+ 添加属性</el-button>
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
import { modelApi, domainApi } from "../api"
import { ElMessage, ElMessageBox } from "element-plus"
import { Delete } from "@element-plus/icons-vue"

const models = ref([])
const domains = ref([])
const loading = ref(false)
const showDialog = ref(false)
const editingModel = ref(null)
const domainFilter = ref("")
const form = ref({ id: "", name: "", domain_id: "", properties: [], commands: [], events: [] })

const loadModels = async () => {
  loading.value = true
  try {
    const res = await modelApi.list(domainFilter.value)
    models.value = res.data.data || []
  } finally { loading.value = false }
}

onMounted(async () => {
  const dRes = await domainApi.list(); domains.value = dRes.data.data || []
  await loadModels()
})

const openAdd = () => {
  editingModel.value = null
  form.value = { id: "", name: "", domain_id: "", properties: [], commands: [], events: [] }
  showDialog.value = true
}

const openEdit = (row) => {
  editingModel.value = row
  form.value = {
    id: row.id,
    name: row.name,
    domain_id: row.domain_id,
    properties: (row.properties || []).map(p => ({ ...p })),
    commands: (row.commands || []).map(c => ({ ...c })),
    events: (row.events || []).map(e => ({ ...e }))
  }
  showDialog.value = true
}

const handleSave = async () => {
  if (editingModel.value) {
    const { id, ...data } = form.value
    await modelApi.update(editingModel.value.id, data)
  } else {
    await modelApi.create(form.value)
  }
  showDialog.value = false
  await loadModels()
  ElMessage.success(editingModel.value ? "模型更新成功" : "模型创建成功")
}

const handleDelete = async (id) => {
  await ElMessageBox.confirm("确定删除该模型?", "提示", { type: "warning" })
  await modelApi.delete(id)
  models.value = models.value.filter(m => m.id !== id)
  ElMessage.success("删除成功")
}
</script>
