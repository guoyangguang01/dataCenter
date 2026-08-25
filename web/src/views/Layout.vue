<template>
  <el-container style="height: 100vh">
    <el-aside :width="collapsed ? '64px' : '220px'" style="background: #001529; transition: width 0.3s">
      <div style="height: 60px; display: flex; align-items: center; justify-content: center; color: #fff; font-size: 18px">
        <span v-if="!collapsed">数据中台</span>
        <span v-else>DC</span>
      </div>
      <el-menu :collapse="collapsed" background-color="#001529" text-color="#ffffffa6" active-text-color="#fff" router>
        <el-menu-item index="/dashboard"><el-icon><Odometer /></el-icon><span>概览</span></el-menu-item>
        <el-menu-item index="/devices"><el-icon><Monitor /></el-icon><span>设备管理</span></el-menu-item>
        <el-menu-item index="/device-data"><el-icon><TrendCharts /></el-icon><span>设备数据</span></el-menu-item>
        <el-menu-item index="/metric-data"><el-icon><DataLine /></el-icon><span>数据维度</span></el-menu-item>
        <el-menu-item index="/rules"><el-icon><Setting /></el-icon><span>规则引擎</span></el-menu-item>
        <el-menu-item index="/models"><el-icon><Box /></el-icon><span>物模型</span></el-menu-item>
        <el-menu-item index="/alerts"><el-icon><Bell /></el-icon><span>告警中心</span></el-menu-item>
        <el-menu-item index="/domains"><el-icon><OfficeBuilding /></el-icon><span>域管理</span></el-menu-item>
        <el-menu-item index="/gateways"><el-icon><Connection /></el-icon><span>网关管理</span></el-menu-item>
        <el-menu-item index="/monitoring"><el-icon><DataLine /></el-icon><span>系统监控</span></el-menu-item>
      </el-menu>
    </el-aside>
    <el-container>
      <el-header style="display: flex; align-items: center; border-bottom: 1px solid #eee; padding: 0 20px">
        <el-icon style="cursor: pointer; font-size: 20px" @click="collapsed = !collapsed"><Fold /></el-icon>
        <el-select v-model="store.currentDomain" placeholder="全部业务域" clearable style="margin-left: 20px; width: 200px">
          <el-option label="全部业务域" value="" />
          <el-option v-for="d in store.domains" :key="d.id" :label="d.name" :value="d.id" />
        </el-select>
      </el-header>
      <el-main style="background: #f5f5f5"><router-view :key="store.currentDomain" /></el-main>
    </el-container>
  </el-container>
</template>

<script setup>
import { ref, onMounted } from "vue"
import { useAppStore } from "../stores/app"
import { Odometer, Monitor, Setting, Box, Bell, OfficeBuilding, Connection, Fold, DataLine, TrendCharts } from "@element-plus/icons-vue"

const store = useAppStore()
const collapsed = ref(false)

onMounted(() => {
  store.loadDomains()
})
</script>
