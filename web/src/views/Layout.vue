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
        <el-menu-item index="/rules"><el-icon><Setting /></el-icon><span>规则引擎</span></el-menu-item>
        <el-menu-item index="/models"><el-icon><Box /></el-icon><span>物模型</span></el-menu-item>
        <el-menu-item index="/alerts"><el-icon><Bell /></el-icon><span>告警中心</span></el-menu-item>
        <el-menu-item index="/domains"><el-icon><OfficeBuilding /></el-icon><span>域管理</span></el-menu-item>
      </el-menu>
    </el-aside>
    <el-container>
      <el-header style="display: flex; align-items: center; border-bottom: 1px solid #eee; padding: 0 20px">
        <el-icon style="cursor: pointer; font-size: 20px" @click="collapsed = !collapsed"><Fold /></el-icon>
        <el-select v-model="currentDomain" placeholder="选择业务域" style="margin-left: 20px; width: 200px">
          <el-option label="工厂A" value="factory-a" />
          <el-option label="工厂B" value="factory-b" />
        </el-select>
      </el-header>
      <el-main style="background: #f5f5f5"><router-view /></el-main>
    </el-container>
  </el-container>
</template>

<script setup>
import { ref } from "vue"
import { useAppStore } from "../stores/app"
const store = useAppStore()
const collapsed = ref(false)
const currentDomain = ref(store.currentDomain)
</script>
