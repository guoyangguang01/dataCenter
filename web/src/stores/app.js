import { defineStore } from "pinia"
import { ref } from "vue"
import { domainApi } from "../api"

export const useAppStore = defineStore("app", () => {
  const currentDomain = ref("")
  const domains = ref([])

  const setDomain = (d) => { currentDomain.value = d }

  const loadDomains = async () => {
    try {
      const res = await domainApi.list()
      domains.value = res.data.data || []
    } catch (e) {
      console.error("Failed to load domains:", e)
    }
  }

  return { currentDomain, domains, setDomain, loadDomains }
})
