<template>
  <div>
    <el-card>
      <template #header>
        <div style="display: flex; justify-content: space-between; align-items: center">
          <span>域管理</span>
          <el-button type="primary" @click="showAddDomain = true">添加域</el-button>
        </div>
      </template>
      <el-table :data="domains" stripe v-loading="loading" @row-click="handleRowClick" highlight-current-row>
        <el-table-column prop="id" label="域ID" width="150" />
        <el-table-column prop="name" label="名称" />
        <el-table-column prop="description" label="描述" />
        <el-table-column prop="created_at" label="创建时间" width="180">
          <template #default="{ row }">{{ formatTime(row.created_at) }}</template>
        </el-table-column>
        <el-table-column label="操作" width="150">
          <template #default="{ row }">
            <el-button size="small" type="danger" @click.stop="handleDeleteDomain(row.id)">删除</el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-card>

    <!-- 成员管理 -->
    <el-card v-if="selectedDomain" style="margin-top: 20px">
      <template #header>
        <div style="display: flex; justify-content: space-between; align-items: center">
          <span>{{ selectedDomain.name }} - 成员管理</span>
          <el-button type="primary" size="small" @click="showAddMember = true">添加成员</el-button>
        </div>
      </template>
      <el-table :data="members" stripe v-loading="membersLoading">
        <el-table-column prop="user_id" label="用户ID" />
        <el-table-column prop="role" label="角色" width="150">
          <template #default="{ row }">
            <el-tag :type="row.role === 'super_admin' ? 'danger' : row.role === 'admin' ? 'warning' : 'info'">
              {{ row.role }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="120">
          <template #default="{ row }">
            <el-button size="small" type="danger" @click="handleRemoveMember(row.user_id)">移除</el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-card>

    <!-- 添加域对话框 -->
    <el-dialog v-model="showAddDomain" title="添加域" width="400px">
      <el-form :model="domainForm" label-width="80px">
        <el-form-item label="域ID">
          <el-input v-model="domainForm.id" />
        </el-form-item>
        <el-form-item label="名称">
          <el-input v-model="domainForm.name" />
        </el-form-item>
        <el-form-item label="描述">
          <el-input v-model="domainForm.description" type="textarea" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="showAddDomain = false">取消</el-button>
        <el-button type="primary" @click="handleCreateDomain">确定</el-button>
      </template>
    </el-dialog>

    <!-- 添加成员对话框 -->
    <el-dialog v-model="showAddMember" title="添加成员" width="400px">
      <el-form :model="memberForm" label-width="80px">
        <el-form-item label="用户ID">
          <el-input v-model="memberForm.user_id" />
        </el-form-item>
        <el-form-item label="角色">
          <el-select v-model="memberForm.role" style="width: 100%">
            <el-option label="超级管理员" value="super_admin" />
            <el-option label="管理员" value="admin" />
            <el-option label="操作员" value="operator" />
            <el-option label="观察者" value="viewer" />
          </el-select>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="showAddMember = false">取消</el-button>
        <el-button type="primary" @click="handleAddMember">确定</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, onMounted } from "vue"
import { domainApi } from "../api"
import { ElMessage, ElMessageBox } from "element-plus"

const domains = ref([])
const loading = ref(false)
const selectedDomain = ref(null)
const members = ref([])
const membersLoading = ref(false)
const showAddDomain = ref(false)
const showAddMember = ref(false)
const domainForm = ref({ id: "", name: "", description: "" })
const memberForm = ref({ user_id: "", role: "viewer" })

const formatTime = (t) => t ? new Date(t).toLocaleString("zh-CN") : ""

onMounted(async () => {
  loading.value = true
  try { const res = await domainApi.list(); domains.value = res.data.data || [] }
  finally { loading.value = false }
})

const handleRowClick = async (row) => {
  selectedDomain.value = row
  membersLoading.value = true
  try { const res = await domainApi.listMembers(row.id); members.value = res.data.data || [] }
  finally { membersLoading.value = false }
}

const handleCreateDomain = async () => {
  await domainApi.create(domainForm.value)
  showAddDomain.value = false
  domainForm.value = { id: "", name: "", description: "" }
  const res = await domainApi.list(); domains.value = res.data.data || []
  ElMessage.success("域创建成功")
}

const handleDeleteDomain = async (id) => {
  await ElMessageBox.confirm("确定删除该域?", "提示", { type: "warning" })
  await domainApi.delete(id)
  domains.value = domains.value.filter(d => d.id !== id)
  if (selectedDomain.value?.id === id) { selectedDomain.value = null; members.value = [] }
  ElMessage.success("删除成功")
}

const handleAddMember = async () => {
  if (!selectedDomain.value) return
  await domainApi.addMember(selectedDomain.value.id, memberForm.value)
  showAddMember.value = false
  memberForm.value = { user_id: "", role: "viewer" }
  const res = await domainApi.listMembers(selectedDomain.value.id); members.value = res.data.data || []
  ElMessage.success("成员添加成功")
}

const handleRemoveMember = async (userId) => {
  if (!selectedDomain.value) return
  await ElMessageBox.confirm("确定移除该成员?", "提示", { type: "warning" })
  await domainApi.removeMember(selectedDomain.value.id, userId)
  members.value = members.value.filter(m => m.user_id !== userId)
  ElMessage.success("成员已移除")
}
</script>
