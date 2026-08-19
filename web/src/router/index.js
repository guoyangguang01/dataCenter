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
  ]}
]
export default createRouter({ history: createWebHistory(), routes })
