import axios from "axios"
const api = axios.create({ baseURL: "/api/v1", timeout: 10000 })

export const deviceApi = {
  list: (p) => api.get("/devices", { params: p }),
  get: (id) => api.get("/devices/" + id),
  create: (d) => api.post("/devices", d),
  update: (id, d) => api.put("/devices/" + id, d),
  delete: (id) => api.delete("/devices/" + id),
}

export const domainApi = {
  list: () => api.get("/domains"),
  get: (id) => api.get("/domains/" + id),
  create: (d) => api.post("/domains", d),
  delete: (id) => api.delete("/domains/" + id),
  listMembers: (id) => api.get("/domains/" + id + "/members"),
  addMember: (id, m) => api.post("/domains/" + id + "/members", m),
  removeMember: (id, userId) => api.delete("/domains/" + id + "/members/" + userId),
}

export const modelApi = {
  list: (domainId) => api.get("/models", { params: { domain_id: domainId } }),
  get: (id) => api.get("/models/" + id),
  create: (m) => api.post("/models", m),
  delete: (id) => api.delete("/models/" + id),
  bind: (data) => api.post("/models/bind", data),
  unbind: (deviceId) => api.delete("/models/unbind/" + deviceId),
  getDeviceModel: (deviceId) => api.get("/models/device/" + deviceId),
}

export const ruleApi = {
  list: (domainId) => api.get("/rules", { params: { domain_id: domainId } }),
  get: (id) => api.get("/rules/" + id),
  create: (r) => api.post("/rules", r),
  update: (id, r) => api.put("/rules/" + id, r),
  delete: (id) => api.delete("/rules/" + id),
  toggle: (id) => api.put("/rules/" + id + "/toggle"),
}

export const alertApi = {
  listWebhooks: (domainId) => api.get("/alerts/webhooks", { params: { domain_id: domainId } }),
  getWebhook: (id) => api.get("/alerts/webhooks/" + id),
  createWebhook: (w) => api.post("/alerts/webhooks", w),
  updateWebhook: (id, w) => api.put("/alerts/webhooks/" + id, w),
  deleteWebhook: (id) => api.delete("/alerts/webhooks/" + id),
  testWebhook: (id) => api.post("/alerts/webhooks/" + id + "/test"),
  listLogs: (webhookId) => api.get("/alerts/logs", { params: { webhook_id: webhookId } }),
}

export const gatewayApi = {
  list: () => api.get("/gateways"),
  get: (id) => api.get("/gateways/" + id),
  create: (g) => api.post("/gateways", g),
  update: (id, g) => api.put("/gateways/" + id, g),
  delete: (id) => api.delete("/gateways/" + id),
  start: (id) => api.post("/gateways/" + id + "/start"),
  stop: (id) => api.post("/gateways/" + id + "/stop"),
}

export default api
