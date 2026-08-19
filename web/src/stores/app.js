import { defineStore } from "pinia"
import { ref } from "vue"
export const useAppStore = defineStore("app", () => {
  const currentDomain = ref("default")
  const setDomain = (d) => { currentDomain.value = d }
  return { currentDomain, setDomain }
})
