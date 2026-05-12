<script setup lang="ts">
import { computed, onMounted, reactive, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import StoryCard from '../components/StoryCard.vue'
import { enterIdentity, getProviderConfig } from '../api/identity'
import { listStoriesPage, toggleStoryLike, type Story } from '../api/stories'

const PAGE_SIZE = 24
const route = useRoute()
const router = useRouter()
const stories = ref<Story[]>([])
const totalStories = ref(0)
const error = ref('')
const loading = ref(true)
const settingsOpen = ref(false)
const settingsSaving = ref(false)
const settingsError = ref('')
const showApiKey = ref(false)
const searchInput = ref(String(route.query.search || ''))
const settingsForm = reactive(getProviderConfig())
const sortMode = computed(() => String(route.query.sort || 'time'))
const currentPage = computed(() => {
  const value = Number(route.query.page || 1)
  return Number.isFinite(value) && value > 0 ? Math.floor(value) : 1
})
const totalPages = computed(() => Math.max(1, Math.ceil(totalStories.value / PAGE_SIZE)))
const pageNumbers = computed(() => {
  const total = totalPages.value
  const current = currentPage.value
  const start = Math.max(1, current - 2)
  const end = Math.min(total, start + 4)
  return Array.from({ length: end - start + 1 }, (_, index) => start + index)
})

onMounted(async () => {
  await loadPage(currentPage.value)
})

watch(currentPage, (page) => {
  loadPage(page)
})

watch(() => [route.query.sort, route.query.search], () => {
  searchInput.value = String(route.query.search || '')
  loadPage(currentPage.value)
})

async function loadPage(pageNumber: number) {
  loading.value = true
  error.value = ''
  try {
    const offset = (pageNumber - 1) * PAGE_SIZE
    const page = await listStoriesPage(PAGE_SIZE, offset, {
      sort: sortMode.value,
      search: String(route.query.search || '')
    })
    stories.value = page.items || []
    totalStories.value = page.total || 0
  } catch (err) {
    error.value = err instanceof Error ? err.message : '故事加载失败'
  } finally {
    loading.value = false
  }
}

function goToPage(pageNumber: number) {
  const nextPage = Math.min(Math.max(1, pageNumber), totalPages.value)
  router.push({ path: '/stories', query: buildListQuery(nextPage) })
}

function buildListQuery(pageNumber = currentPage.value) {
  const query: Record<string, string> = {}
  const search = String(route.query.search || '').trim()
  if (pageNumber > 1) query.page = String(pageNumber)
  if (sortMode.value !== 'time') query.sort = sortMode.value
  if (search) query.search = search
  return query
}

function changeSort(value: string) {
  const query = buildListQuery(1)
  if (value === 'time') delete query.sort
  else query.sort = value
  router.push({ path: '/stories', query })
}

function submitSearch() {
  const search = searchInput.value.trim()
  const query: Record<string, string> = {}
  if (sortMode.value !== 'time') query.sort = sortMode.value
  if (search) query.search = search
  router.push({ path: '/stories', query })
}

async function handleLike(story: Story) {
  const previous = { liked: story.liked, likeCount: story.likeCount || 0 }
  story.liked = !story.liked
  story.likeCount = Math.max(0, previous.likeCount + (story.liked ? 1 : -1))
  try {
    const result = await toggleStoryLike(story.slug)
    story.liked = result.liked
    story.likeCount = result.likeCount
  } catch (err) {
    story.liked = previous.liked
    story.likeCount = previous.likeCount
    error.value = err instanceof Error ? err.message : '点赞失败'
  }
}

function openSettings() {
  const saved = getProviderConfig()
  settingsForm.baseUrl = saved.baseUrl
  settingsForm.apiKey = saved.apiKey
  settingsError.value = ''
  showApiKey.value = false
  settingsOpen.value = true
}

async function saveSettings() {
  if (!settingsForm.baseUrl.trim() || !settingsForm.apiKey.trim()) {
    settingsError.value = 'baseurl 和 apikey 都不能为空'
    return
  }
  settingsSaving.value = true
  settingsError.value = ''
  try {
    await enterIdentity(settingsForm.baseUrl.trim(), settingsForm.apiKey.trim())
    settingsOpen.value = false
    await loadPage(currentPage.value)
  } catch (err) {
    settingsError.value = err instanceof Error ? err.message : '连接设置保存失败'
  } finally {
    settingsSaving.value = false
  }
}
</script>

<template>
  <main class="page">
    <header class="header">
      <div>
        <div class="title-row">
          <h1>故事集合</h1>
          <button class="settings-icon-button" title="连接设置" aria-label="连接设置" @click="openSettings">⚙</button>
        </div>
        <p class="muted">选择一个故事。进入故事后，每个新对话都会绑定独立的故事身份。</p>
      </div>
      <button class="admin-button" @click="router.push('/admin')">书写你的故事</button>
    </header>

    <section class="list-controls">
      <form class="search-form" @submit.prevent="submitSearch">
        <input v-model="searchInput" placeholder="搜索故事标题或简介" />
        <button type="submit">搜索</button>
      </form>
      <label class="sort-control">
        <select :value="sortMode" @change="changeSort(($event.target as HTMLSelectElement).value)">
          <option value="time">排序：按时间</option>
          <option value="likes">排序：按点赞</option>
        </select>
      </label>
    </section>

    <p v-if="loading" class="muted">正在加载故事...</p>
    <p v-else-if="error" class="error">{{ error }}</p>
    <section v-else class="grid">
      <StoryCard v-for="story in stories" :key="story.slug" :story="story" @like="handleLike" />
    </section>
    <div v-if="!loading && !error" class="pagination-row">
      <span class="muted">第 {{ currentPage }} / {{ totalPages }} 页，共 {{ totalStories }} 个故事</span>
      <div class="page-buttons">
        <button :disabled="currentPage <= 1" @click="goToPage(currentPage - 1)">上一页</button>
        <button
          v-for="page in pageNumbers"
          :key="page"
          :class="{ active: page === currentPage }"
          @click="goToPage(page)"
        >
          {{ page }}
        </button>
        <button :disabled="currentPage >= totalPages" @click="goToPage(currentPage + 1)">下一页</button>
      </div>
    </div>

    <div v-if="settingsOpen" class="settings-layer" @click.self="settingsOpen = false">
      <form class="settings-panel panel" @submit.prevent="saveSettings">
        <header>
          <h2>连接设置</h2>
          <button type="button" class="ghost-button" @click="settingsOpen = false">关闭</button>
        </header>
        <label>
          <span>baseurl</span>
          <input v-model="settingsForm.baseUrl" placeholder="https://api.example.com" />
        </label>
        <label>
          <span>apikey</span>
          <div class="secret-field">
            <input v-model="settingsForm.apiKey" :type="showApiKey ? 'text' : 'password'" placeholder="sk-..." />
            <button type="button" class="ghost-button" @click="showApiKey = !showApiKey">{{ showApiKey ? '隐藏' : '显示' }}</button>
          </div>
        </label>
        <p v-if="settingsError" class="settings-error">{{ settingsError }}</p>
        <button class="primary-button" type="submit" :disabled="settingsSaving">
          {{ settingsSaving ? '保存中...' : '保存连接' }}
        </button>
      </form>
    </div>
  </main>
</template>

<style scoped>
.page {
  padding-bottom: 104px;
}
.header {
  display: flex;
  align-items: end;
  justify-content: space-between;
  margin-bottom: 24px;
}
.title-row {
  display: flex;
  align-items: center;
  gap: 10px;
}
h1 { margin: 0 0 8px; font-size: clamp(28px, 5vw, 44px); }
.settings-icon-button {
  width: 36px;
  height: 36px;
  border-radius: 8px;
  padding: 0;
  background: var(--control-hover);
  color: var(--text);
  font-size: 18px;
  line-height: 1;
}
.list-controls {
  display: grid;
  grid-template-columns: minmax(220px, 1fr) auto;
  gap: 12px;
  align-items: end;
  margin-bottom: 20px;
}
.search-form {
  display: grid;
  grid-template-columns: minmax(0, 1fr) auto;
  gap: 8px;
  margin: 0;
}
.search-form input,
.sort-control select {
  width: 100%;
  min-height: 42px;
  border: 1px solid var(--border-strong);
  border-radius: 8px;
  background: var(--control);
  color: var(--text-strong);
  padding: 11px 12px;
  outline: none;
}
.search-form button,
.sort-control {
  border-radius: 8px;
}
.search-form button {
  min-height: 42px;
  padding: 10px 14px;
  background: var(--control-hover);
  color: var(--text);
}
.sort-control {
  display: block;
  min-width: 150px;
  margin: 0;
  color: var(--label);
  font-size: 14px;
}
.grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(300px, 1fr));
  gap: 22px;
}
.pagination-row {
  position: fixed;
  left: 50%;
  bottom: 16px;
  z-index: 20;
  transform: translateX(-50%);
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 14px;
  flex-wrap: wrap;
  width: min(920px, calc(100% - 32px));
  padding: 10px 12px;
  border: 1px solid var(--control-hover);
  border-radius: 12px;
  background: var(--panel-elevated);
  box-shadow: 0 18px 44px color-mix(in srgb, var(--bg-deep) 58%, transparent);
  backdrop-filter: blur(10px);
  margin-top: 0;
}
.page-buttons {
  display: flex;
  gap: 8px;
  flex-wrap: wrap;
  justify-content: center;
}
.page-buttons button {
  border-radius: 8px;
  padding: 10px 14px;
  background: var(--control-hover);
  color: var(--text);
}
.page-buttons button.active {
  background: var(--accent);
  color: var(--accent-contrast);
  font-weight: 700;
}
.page-buttons button:disabled {
  cursor: not-allowed;
  opacity: 0.55;
}
.error { color: var(--danger-text); }
.admin-button {
  border-radius: 8px;
  padding: 10px 14px;
  background: var(--accent);
  color: var(--accent-contrast);
  font-weight: 700;
}
.settings-layer {
  position: fixed;
  inset: 0;
  z-index: 40;
  display: grid;
  place-items: center;
  padding: 20px;
  background: var(--overlay);
}
.settings-panel {
  width: min(460px, 100%);
  display: grid;
  gap: 14px;
  padding: 18px;
  border-radius: 12px;
  background: var(--panel-elevated);
}
.settings-panel header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
}
h2 {
  margin: 0;
  font-size: 20px;
}
label {
  display: grid;
  gap: 7px;
  color: var(--label);
  font-size: 14px;
}
input {
  width: 100%;
  border: 1px solid var(--border-strong);
  border-radius: 8px;
  background: var(--control);
  color: var(--text-strong);
  padding: 11px 12px;
  outline: none;
}
.secret-field {
  display: grid;
  grid-template-columns: minmax(0, 1fr) auto;
  gap: 8px;
}
.ghost-button,
.primary-button {
  border-radius: 8px;
  padding: 8px 10px;
  background: var(--control-hover);
  color: var(--text);
}
.primary-button {
  padding: 11px 14px;
  background: var(--accent);
  color: var(--accent-contrast);
  font-weight: 700;
}
.primary-button:disabled {
  cursor: not-allowed;
  opacity: 0.55;
}
.settings-error {
  margin: 0;
  color: var(--danger-text);
}
@media (max-width: 640px) {
  .header { align-items: stretch; flex-direction: column; }
  .list-controls,
  .search-form { grid-template-columns: 1fr; }
}
</style>
