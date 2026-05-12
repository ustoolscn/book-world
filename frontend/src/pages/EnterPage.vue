<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { enterIdentity } from '../api/identity'

const route = useRoute()
const router = useRouter()
const error = ref('')
const loading = ref(true)

onMounted(async () => {
  const baseUrl = String(route.query.baseurl || route.query.baseUrl || '')
  const apiKey = String(route.query.apikey || route.query.apiKey || '')
  if (!baseUrl || !apiKey) {
    loading.value = false
    error.value = '请通过 /app?baseurl=...&apikey=... 进入，或点击右上角连接设置。baseurl 不需要包含 /v1。'
    return
  }
  try {
    await enterIdentity(baseUrl, apiKey)
    await router.replace('/stories')
  } catch (err) {
    loading.value = false
    error.value = err instanceof Error ? err.message : '进入失败'
  }
})
</script>

<template>
  <main class="page enter">
    <section class="panel box">
      <h1>Book World</h1>
      <p v-if="loading" class="muted">正在载入你的故事世界...</p>
      <p v-else class="error">{{ error }}</p>
    </section>
  </main>
</template>

<style scoped>
.enter {
  min-height: 100vh;
  display: grid;
  place-items: center;
}
.box { width: min(520px, 100%); padding: 32px; text-align: center; }
h1 { margin: 0 0 12px; }
.error { color: var(--danger-text); }
</style>
