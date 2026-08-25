import { createRouter, createWebHistory } from "vue-router"
const routes = [
  { path: "/", component: () => import("../views/Layout.vue"), children: [
    { path: "", redirect: "/dashboard" },
    { path: "dashboard", name: "Dashboard", component: () => import("../views/Dashboard.vue") },
    { path: "devices", name: "Devices", component: () => import("../views/Devices.vue") },
    { path: "rules", name: "Rules", component: () => import("../views/Rules.vue") },
    { path: "models", name: "Models", component: () => import("../views/Models.vue") },
    { path: "alerts", name: "Alerts", component: () => import("../views/Alerts.vue") },
    { path: "domains", name: "Domains", component: () => import("../views/Domains.vue") },
    { path: "gateways", name: "Gateways", component: () => import("../views/Gateways.vue") },
    { path: "monitoring", name: "Monitoring", component: () => import("../views/Monitoring.vue") },
    { path: "device-data", name: "DeviceData", component: () => import("../views/DeviceData.vue") },
    { path: "metric-data", name: "MetricData", component: () => import("../views/MetricData.vue") },
  ]}
]
export default createRouter({ history: createWebHistory(), routes })
