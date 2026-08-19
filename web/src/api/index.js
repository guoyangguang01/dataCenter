import axios from "axios"
const api = axios.create({ baseURL: "/api/v1", timeout: 10000 })
export const deviceApi = {
  list: (p) => api.get("/devices", { params: p }),
  get: (id) => api.get("/devices/" + id),
  create: (d) => api.post("/devices", d),
  update: (id, d) => api.put("/devices/" + id, d),
  delete: (id) => api.delete("/devices/" + id),
}
export default api
