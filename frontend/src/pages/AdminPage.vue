<script setup lang="ts">
import { computed, onMounted, onUnmounted, reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import { listModels } from '../api/models'
import {
  createAdminCharacter,
  createAdminStory,
  createAdminWorldInfo,
  deleteAdminCharacter,
  deleteAdminStory,
  deleteAdminWorldInfo,
  generateAdminStoryDraft,
  listAdminCharacters,
  listAdminStoriesPage,
  listAdminWorldInfo,
  updateAdminCharacter,
  updateAdminStory,
  updateAdminWorldInfo,
  type Character,
  type DraftChatMessage,
  type Story,
  type StoryDraft,
  type StorySettingsPayload,
  type WorldInfo
} from '../api/stories'

type TabName = 'generate' | 'story' | 'characters' | 'world'

const ADMIN_PAGE_SIZE = 18
const router = useRouter()
const stories = ref<Story[]>([])
const totalStories = ref(0)
const storyPage = ref(1)
const storySearchInput = ref('')
const storySearch = ref('')
const characters = ref<Character[]>([])
const worldInfo = ref<WorldInfo[]>([])
const selectedSlug = ref('')
const loading = ref(true)
const detailLoading = ref(false)
const saving = ref(false)
const generating = ref(false)
const modelLoading = ref(false)
const error = ref('')
const notice = ref('')
const activeTab = ref<TabName>('story')
const characterDrawerOpen = ref(false)
const worldDrawerOpen = ref(false)

const modelOptions = ref<string[]>([])
const selectedModel = ref(localStorage.getItem('book-world-model') || '')
const selectedThinkingEffort = ref(localStorage.getItem('book-world-thinking-effort') || '')
const draftPrompt = ref('')
const draftMessages = ref<DraftChatMessage[]>([])
const generatedDraft = ref<StoryDraft | null>(null)
const generationElapsed = ref(0)
const generationProgressMessage = ref('')
const generationProgressPercent = ref(0)
let generationTimer: number | undefined

const thinkingEffortOptions = [
  { value: '', label: '默认' },
  { value: 'low', label: '低' },
  { value: 'medium', label: '中' },
  { value: 'high', label: '高' }
]

const emptyStory = (): StorySettingsPayload & { slug: string } => ({
  slug: '',
  title: '',
  description: '',
  coverUrl: '',
  systemPrompt: '',
  scenario: '',
  stylePrompt: '',
  openingMessage: ''
})

const storyForm = reactive(emptyStory())
const characterForm = reactive<Omit<Character, 'id' | 'storyId'> & { id: string }>({
  id: '',
  name: '',
  description: '',
  personality: '',
  exampleDialogue: '',
  priority: 100
})
const worldForm = reactive<Omit<WorldInfo, 'id' | 'storyId'> & { id: string; keywordsText: string }>({
  id: '',
  keywords: [],
  keywordsText: '',
  content: '',
  priority: 100,
  enabled: true
})

const selectedStory = computed(() => stories.value.find((story) => story.slug === selectedSlug.value))
const isNewStory = computed(() => !selectedStory.value)
const canUseConfigTabs = computed(() => Boolean(selectedSlug.value || generatedDraft.value))
const isDraftPreview = computed(() => !selectedSlug.value && Boolean(generatedDraft.value))
const draftCharacters = computed(() => generatedDraft.value?.characters || [])
const draftWorldInfo = computed(() => generatedDraft.value?.worldInfo || [])
const generationElapsedLabel = computed(() => `${generationElapsed.value}s`)
const modeLabel = computed(() => selectedSlug.value ? '编辑已有故事' : '新建故事')
const totalStoryPages = computed(() => Math.max(1, Math.ceil(totalStories.value / ADMIN_PAGE_SIZE)))
const storyPageNumbers = computed(() => {
  const total = totalStoryPages.value
  const current = storyPage.value
  const start = Math.max(1, Math.min(current - 2, total - 4))
  const end = Math.min(total, start + 4)
  return Array.from({ length: end - start + 1 }, (_, index) => start + index)
})

onMounted(async () => {
  await Promise.all([loadStories(), loadModels()])
})

onUnmounted(stopGenerationProgress)

async function loadModels() {
  modelLoading.value = true
  try {
    const models = await listModels()
    modelOptions.value = models
    if (!selectedModel.value || !models.includes(selectedModel.value)) {
      selectedModel.value = models[0] || ''
    }
  } catch (err) {
    error.value = err instanceof Error ? `模型列表加载失败：${err.message}` : '模型列表加载失败'
  } finally {
    modelLoading.value = false
  }
}

async function loadStories(slugToSelect = selectedSlug.value) {
  loading.value = true
  error.value = ''
  try {
    const page = await listAdminStoriesPage(ADMIN_PAGE_SIZE, (storyPage.value - 1) * ADMIN_PAGE_SIZE, {
      search: storySearch.value
    })
    stories.value = page.items || []
    totalStories.value = page.total || 0
    if (!stories.value.length && storyPage.value > 1 && totalStories.value > 0) {
      storyPage.value -= 1
      await loadStories('')
      return
    }
    selectedSlug.value = stories.value.some((story) => story.slug === slugToSelect)
      ? slugToSelect
      : stories.value[0]?.slug || ''
    syncStoryForm()
    await loadStoryDetails()
  } catch (err) {
    error.value = err instanceof Error ? err.message : '创作数据加载失败'
  } finally {
    loading.value = false
  }
}

async function submitStorySearch() {
  storySearch.value = storySearchInput.value.trim()
  storyPage.value = 1
  await loadStories('')
}

async function goToStoryPage(page: number) {
  storyPage.value = Math.min(Math.max(1, page), totalStoryPages.value)
  await loadStories('')
}

async function openTab(tab: TabName) {
  if ((tab === 'characters' || tab === 'world') && !canUseConfigTabs.value) return
  activeTab.value = tab
  if ((tab === 'characters' || tab === 'world') && selectedSlug.value && !detailLoading.value) {
    await loadStoryDetails()
  }
}

async function selectStory(slug: string) {
  selectedSlug.value = slug
  activeTab.value = 'story'
  generatedDraft.value = null
  characterDrawerOpen.value = false
  worldDrawerOpen.value = false
  syncStoryForm()
  await loadStoryDetails()
}

function startNewStory() {
  Object.assign(storyForm, emptyStory())
  selectedSlug.value = ''
  characters.value = []
  worldInfo.value = []
  activeTab.value = 'generate'
  characterDrawerOpen.value = false
  worldDrawerOpen.value = false
  notice.value = ''
  error.value = ''
  resetCharacterForm()
  resetWorldForm()
  resetDraftChat()
}

function syncStoryForm() {
  const story = selectedStory.value
  if (!story) {
    Object.assign(storyForm, emptyStory())
    return
  }
  Object.assign(storyForm, {
    slug: story.slug,
    title: story.title || '',
    description: story.description || '',
    coverUrl: story.coverUrl || '',
    systemPrompt: story.systemPrompt || '',
    scenario: story.scenario || '',
    stylePrompt: story.stylePrompt || '',
    openingMessage: story.openingMessage || ''
  })
}

async function loadStoryDetails() {
  if (!selectedSlug.value) return
  detailLoading.value = true
  error.value = ''
  try {
    const [nextCharacters, nextWorldInfo] = await Promise.all([
      listAdminCharacters(selectedSlug.value),
      listAdminWorldInfo(selectedSlug.value)
    ])
    characters.value = nextCharacters || []
    worldInfo.value = nextWorldInfo || []
    resetCharacterForm()
    resetWorldForm()
  } catch (err) {
    error.value = err instanceof Error ? err.message : '配置加载失败'
  } finally {
    detailLoading.value = false
  }
}

async function saveStory() {
  if (!storyForm.slug.trim() || !storyForm.title.trim() || !storyForm.systemPrompt.trim()) {
    error.value = 'Slug、标题和故事规则不能为空'
    return
  }
  saving.value = true
  error.value = ''
  notice.value = ''
  try {
    const payload = { ...storyForm, slug: storyForm.slug.trim().toLowerCase() }
    const shouldImportDraft = isNewStory.value && generatedDraft.value
    const previousSlug = selectedSlug.value
    const story = isNewStory.value ? await createAdminStory(payload) : await updateAdminStory(previousSlug, payload)
    if (shouldImportDraft) await importGeneratedContent(story.slug)
    notice.value = shouldImportDraft ? '故事已创建，AI 生成的角色和世界书也已导入' : isNewStory.value ? '故事已创建' : '故事已保存'
    storySearchInput.value = ''
    storySearch.value = ''
    storyPage.value = 1
    await loadStories(story.slug)
  } catch (err) {
    error.value = err instanceof Error ? err.message : '故事保存失败'
  } finally {
    saving.value = false
  }
}

async function generateDraft() {
  const message = draftPrompt.value.trim()
  if (!message) {
    error.value = '请先告诉 AI 你想要什么故事'
    return
  }
  generating.value = true
  error.value = ''
  notice.value = ''
  startGenerationProgress()
  try {
    const response = await generateAdminStoryDraft({
      messages: draftMessages.value,
      message,
      model: selectedModel.value,
      thinkingEffort: selectedThinkingEffort.value
    })
    if (selectedModel.value) localStorage.setItem('book-world-model', selectedModel.value)
    localStorage.setItem('book-world-thinking-effort', selectedThinkingEffort.value)
    draftMessages.value.push({ role: 'user', content: message })
    draftMessages.value.push({ role: 'assistant', content: response.reply })
    generatedDraft.value = response.draft
    draftPrompt.value = ''
    applyGeneratedDraft()
    activeTab.value = 'story'
    generationProgressMessage.value = '草稿已生成，正在写入预览'
    generationProgressPercent.value = 100
    notice.value = 'AI 草稿已应用到表单。角色和世界书可切换页签预览，保存故事时会一并导入。'
  } catch (err) {
    error.value = err instanceof Error ? err.message : 'AI 生成失败'
  } finally {
    generating.value = false
    stopGenerationProgress()
  }
}

function startGenerationProgress() {
  stopGenerationProgress()
  generationElapsed.value = 0
  generationProgressPercent.value = 8
  generationProgressMessage.value = '正在连接模型'
  generationTimer = window.setInterval(() => {
    generationElapsed.value += 1
    if (generationElapsed.value < 4) {
      generationProgressMessage.value = '正在连接模型'
      generationProgressPercent.value = Math.min(20, generationProgressPercent.value + 3)
    } else if (generationElapsed.value < 20) {
      generationProgressMessage.value = '正在生成故事设定'
      generationProgressPercent.value = Math.min(58, generationProgressPercent.value + 2)
    } else if (generationElapsed.value < 45) {
      generationProgressMessage.value = '正在生成角色和世界书'
      generationProgressPercent.value = Math.min(82, generationProgressPercent.value + 1)
    } else {
      generationProgressMessage.value = '模型仍在思考，复杂草稿会更久'
      generationProgressPercent.value = Math.min(94, generationProgressPercent.value + 0.35)
    }
  }, 1000)
}

function stopGenerationProgress() {
  if (!generationTimer) return
  window.clearInterval(generationTimer)
  generationTimer = undefined
}

function applyGeneratedDraft() {
  if (!generatedDraft.value) return
  Object.assign(storyForm, {
    slug: generatedDraft.value.story.slug || storyForm.slug,
    title: generatedDraft.value.story.title || storyForm.title,
    description: generatedDraft.value.story.description || '',
    coverUrl: generatedDraft.value.story.coverUrl || '',
    systemPrompt: generatedDraft.value.story.systemPrompt || '',
    scenario: generatedDraft.value.story.scenario || '',
    stylePrompt: generatedDraft.value.story.stylePrompt || '',
    openingMessage: generatedDraft.value.story.openingMessage || ''
  })
}

function resetDraftChat() {
  draftMessages.value = []
  generatedDraft.value = null
  draftPrompt.value = ''
  if (!selectedSlug.value) {
    activeTab.value = 'generate'
  }
}

async function importGeneratedContent(slug: string) {
  const draft = generatedDraft.value
  if (!draft) return
  await Promise.all([
    ...draft.characters
      .filter((character) => character.name?.trim())
      .map((character) => createAdminCharacter(slug, {
        name: character.name,
        description: character.description || '',
        personality: character.personality || '',
        exampleDialogue: character.exampleDialogue || '',
        priority: Number(character.priority || 100)
      })),
    ...draft.worldInfo
      .filter((entry) => entry.content?.trim() && entry.keywords?.length)
      .map((entry) => createAdminWorldInfo(slug, {
        keywords: entry.keywords,
        content: entry.content,
        priority: Number(entry.priority || 100),
        enabled: entry.enabled !== false
      }))
  ])
}

async function removeStory() {
  if (!selectedSlug.value || !confirm(`删除故事「${selectedStory.value?.title || selectedSlug.value}」？相关角色、世界书和云端记录都会被删除。`)) {
    return
  }
  saving.value = true
  error.value = ''
  notice.value = ''
  try {
    await deleteAdminStory(selectedSlug.value)
    notice.value = '故事已删除'
    await loadStories('')
  } catch (err) {
    error.value = err instanceof Error ? err.message : '故事删除失败'
  } finally {
    saving.value = false
  }
}

function editCharacter(character: Character) {
  Object.assign(characterForm, {
    id: character.id,
    name: character.name,
    description: character.description,
    personality: character.personality,
    exampleDialogue: character.exampleDialogue,
    priority: character.priority
  })
  activeTab.value = 'characters'
  characterDrawerOpen.value = true
}

function resetCharacterForm() {
  Object.assign(characterForm, {
    id: '',
    name: '',
    description: '',
    personality: '',
    exampleDialogue: '',
    priority: 100
  })
}

function createCharacter() {
  resetCharacterForm()
  activeTab.value = 'characters'
  characterDrawerOpen.value = true
}

async function saveCharacter() {
  if (!selectedSlug.value || !characterForm.name.trim()) {
    error.value = '请先选择已有故事并填写角色名'
    return
  }
  saving.value = true
  error.value = ''
  notice.value = ''
  try {
    const { id, ...payload } = characterForm
    if (id) await updateAdminCharacter(id, payload)
    else await createAdminCharacter(selectedSlug.value, payload)
    notice.value = id ? '角色已保存' : '角色已创建'
    characterDrawerOpen.value = false
    await loadStoryDetails()
  } catch (err) {
    error.value = err instanceof Error ? err.message : '角色保存失败'
  } finally {
    saving.value = false
  }
}

async function removeCharacter(character: Character) {
  if (!confirm(`删除角色「${character.name}」？`)) return
  saving.value = true
  error.value = ''
  notice.value = ''
  try {
    await deleteAdminCharacter(character.id)
    notice.value = '角色已删除'
    await loadStoryDetails()
  } catch (err) {
    error.value = err instanceof Error ? err.message : '角色删除失败'
  } finally {
    saving.value = false
  }
}

function editWorld(entry: WorldInfo) {
  Object.assign(worldForm, {
    id: entry.id,
    keywords: [...entry.keywords],
    keywordsText: entry.keywords.join(', '),
    content: entry.content,
    priority: entry.priority,
    enabled: entry.enabled
  })
  activeTab.value = 'world'
  worldDrawerOpen.value = true
}

function resetWorldForm() {
  Object.assign(worldForm, {
    id: '',
    keywords: [],
    keywordsText: '',
    content: '',
    priority: 100,
    enabled: true
  })
}

function createWorldInfo() {
  resetWorldForm()
  activeTab.value = 'world'
  worldDrawerOpen.value = true
}

async function saveWorldInfo() {
  if (!selectedSlug.value || !worldForm.keywordsText.trim() || !worldForm.content.trim()) {
    error.value = '请先选择已有故事，并填写关键词和内容'
    return
  }
  saving.value = true
  error.value = ''
  notice.value = ''
  try {
    const payload = {
      keywords: worldForm.keywordsText.split(/[,，\n]/).map((keyword) => keyword.trim()).filter(Boolean),
      content: worldForm.content,
      priority: worldForm.priority,
      enabled: worldForm.enabled
    }
    if (worldForm.id) await updateAdminWorldInfo(worldForm.id, payload)
    else await createAdminWorldInfo(selectedSlug.value, payload)
    notice.value = worldForm.id ? '世界书条目已保存' : '世界书条目已创建'
    worldDrawerOpen.value = false
    await loadStoryDetails()
  } catch (err) {
    error.value = err instanceof Error ? err.message : '世界书保存失败'
  } finally {
    saving.value = false
  }
}

async function removeWorldInfo(entry: WorldInfo) {
  if (!confirm(`删除世界书条目「${entry.keywords.join(', ')}」？`)) return
  saving.value = true
  error.value = ''
  notice.value = ''
  try {
    await deleteAdminWorldInfo(entry.id)
    notice.value = '世界书条目已删除'
    await loadStoryDetails()
  } catch (err) {
    error.value = err instanceof Error ? err.message : '世界书删除失败'
  } finally {
    saving.value = false
  }
}
</script>

<template>
  <main class="page admin-page">
    <header class="admin-header">
      <div>
        <button class="ghost-button" @click="router.push('/stories')">返回故事集合</button>
        <h1>书写你的故事</h1>
        <p class="muted">创作并管理你自己创建的故事、角色和世界书配置。</p>
      </div>
    </header>

    <p v-if="loading" class="muted">正在加载管理数据...</p>
    <p v-if="error" class="error">{{ error }}</p>
    <p v-if="notice" class="notice">{{ notice }}</p>

    <section v-if="!loading" class="admin-layout">
      <aside class="story-nav panel">
        <button class="story-nav-item new-item" :class="{ active: !selectedSlug }" @click="startNewStory">
          <strong>新建故事</strong>
          <span>AI 草稿或手动填写</span>
        </button>
        <form class="story-search" @submit.prevent="submitStorySearch">
          <input v-model="storySearchInput" placeholder="搜索标题、简介或 slug" />
          <button type="submit">搜索</button>
        </form>
        <button
          v-for="story in stories"
          :key="story.slug"
          class="story-nav-item"
          :class="{ active: story.slug === selectedSlug }"
          @click="selectStory(story.slug)"
        >
          <strong>{{ story.title }}</strong>
          <span>{{ story.slug }}</span>
        </button>
        <p v-if="!stories.length" class="empty-state">没有找到故事</p>
        <div class="story-pager">
          <span>第 {{ storyPage }} / {{ totalStoryPages }} 页，共 {{ totalStories }} 个</span>
          <div class="story-page-buttons">
            <button type="button" :disabled="storyPage <= 1" @click="goToStoryPage(storyPage - 1)">上一页</button>
            <button
              v-for="page in storyPageNumbers"
              :key="page"
              type="button"
              :class="{ active: page === storyPage }"
              @click="goToStoryPage(page)"
            >
              {{ page }}
            </button>
            <button type="button" :disabled="storyPage >= totalStoryPages" @click="goToStoryPage(storyPage + 1)">下一页</button>
          </div>
        </div>
      </aside>

      <section class="editor">
        <div class="mode-strip">
          <strong>{{ modeLabel }}</strong>
          <span v-if="isDraftPreview">正在预览 AI 草稿，保存故事后角色和世界书会正式导入。</span>
          <span v-else-if="selectedSlug">正在编辑数据库里的现有配置。</span>
          <span v-else>先生成或填写故事；生成后可预览角色和世界书。</span>
        </div>

        <nav class="tabs">
          <button type="button" :class="{ active: activeTab === 'generate' }" @click="openTab('generate')">AI 生成</button>
          <button type="button" :class="{ active: activeTab === 'story' }" @click="openTab('story')">故事</button>
          <button type="button" :disabled="!canUseConfigTabs" :class="{ active: activeTab === 'characters' }" @click="openTab('characters')">角色</button>
          <button type="button" :disabled="!canUseConfigTabs" :class="{ active: activeTab === 'world' }" @click="openTab('world')">世界书</button>
        </nav>

        <section v-if="activeTab === 'generate'" class="draft-panel panel">
          <div class="draft-header">
            <div>
              <h2>AI 生成故事</h2>
              <p class="muted">描述题材、氛围、主角身份或关键设定，AI 会生成可保存的故事、角色和世界书。</p>
            </div>
            <button type="button" :disabled="generating" @click="resetDraftChat">重置对话</button>
          </div>

          <div v-if="draftMessages.length" class="draft-chat">
            <article v-for="(message, index) in draftMessages" :key="index" :class="['draft-message', message.role]">
              <strong>{{ message.role === 'user' ? '你' : 'AI' }}</strong>
              <p>{{ message.role === 'assistant' ? '已生成一版结构化故事草稿。你可以继续要求调整，也可以直接保存。' : message.content }}</p>
            </article>
          </div>

          <label class="field">
            <span>生成需求</span>
            <textarea
              v-model="draftPrompt"
              rows="4"
              placeholder="例如：做一个民国怪谈故事，玩家是新来的报社记者，核心谜团和一座旧戏院有关，氛围克制阴冷。"
              @keydown.ctrl.enter.prevent="generateDraft"
            />
          </label>

          <section class="draft-controls">
            <label class="field">
              <span>{{ modelLoading ? '加载模型...' : '模型' }}</span>
              <select v-model="selectedModel" :disabled="generating || modelLoading || modelOptions.length === 0">
                <option v-if="modelOptions.length === 0" value="">使用后端默认模型</option>
                <option v-for="model in modelOptions" :key="model" :value="model">{{ model }}</option>
              </select>
            </label>
            <label class="field">
              <span>思考等级</span>
              <select v-model="selectedThinkingEffort" :disabled="generating">
                <option v-for="option in thinkingEffortOptions" :key="option.value" :value="option.value">{{ option.label }}</option>
              </select>
            </label>
          </section>

          <div v-if="generating" class="progress-box">
            <div class="progress-meta">
              <span>{{ generationProgressMessage }}</span>
              <span>{{ generationElapsedLabel }}</span>
            </div>
            <div class="progress-track">
              <div class="progress-fill" :style="{ width: `${generationProgressPercent}%` }" />
            </div>
          </div>

          <div v-if="generatedDraft" class="draft-summary">
            <span>草稿：{{ generatedDraft.story.title }} · {{ generatedDraft.characters.length }} 个角色 · {{ generatedDraft.worldInfo.length }} 条世界书</span>
            <button type="button" @click="applyGeneratedDraft">重新应用到表单</button>
          </div>

          <div class="button-row">
            <button class="primary-button" type="button" :disabled="generating" @click="generateDraft">
              {{ generating ? '生成中...' : draftMessages.length ? '继续生成/调整' : '生成故事草稿' }}
            </button>
          </div>
        </section>

        <form v-if="activeTab === 'story'" class="panel editor-form" @submit.prevent="saveStory">
          <div class="form-grid">
            <label class="field">
              <span>Slug *</span>
              <input v-model="storyForm.slug" placeholder="midnight-inn" />
            </label>
            <label class="field">
              <span>标题 *</span>
              <input v-model="storyForm.title" placeholder="故事标题" />
            </label>
          </div>

          <label class="field">
            <span>描述</span>
            <textarea v-model="storyForm.description" rows="3" placeholder="故事列表里展示的简介" />
          </label>

          <label class="field">
            <span>封面地址</span>
            <input v-model="storyForm.coverUrl" placeholder="https://..." />
          </label>

          <label class="field">
            <span>故事规则 *</span>
            <textarea v-model="storyForm.systemPrompt" rows="5" placeholder="给故事主持人的核心规则" />
          </label>

          <label class="field">
            <span>场景</span>
            <textarea v-model="storyForm.scenario" rows="5" placeholder="玩家身份、舞台、当前局势" />
          </label>

          <label class="field">
            <span>风格</span>
            <textarea v-model="storyForm.stylePrompt" rows="4" placeholder="叙事视角、语气、节奏、回复习惯" />
          </label>

          <label class="field">
            <span>开场白</span>
            <textarea v-model="storyForm.openingMessage" rows="5" placeholder="新对话显示的第一条助手消息" />
          </label>

          <div class="button-row">
            <button v-if="selectedSlug" type="button" class="danger-button" :disabled="saving" @click="removeStory">删除故事</button>
            <button v-if="selectedSlug" type="button" :disabled="saving" @click="router.push(`/stories/${selectedSlug}`)">进入故事</button>
            <button class="primary-button" type="submit" :disabled="saving">{{ saving ? '保存中...' : isNewStory ? '创建故事' : '保存故事' }}</button>
          </div>
        </form>

        <section v-if="activeTab === 'characters'" class="config-panel">
          <section v-if="isDraftPreview" class="panel preview-panel">
            <h2>AI 草稿角色预览</h2>
            <p class="muted">这些角色还没有写入数据库。点击“创建故事”后会自动导入。</p>
            <article v-for="(character, index) in draftCharacters" :key="`${character.name}-${index}`" class="list-item preview-item">
              <div>
                <strong>{{ character.name }}</strong>
                <span>优先级 {{ character.priority }}</span>
                <p>{{ character.description || '暂无描述' }}</p>
                <p v-if="character.personality"><b>性格：</b>{{ character.personality }}</p>
                <p v-if="character.exampleDialogue"><b>示例：</b>{{ character.exampleDialogue }}</p>
              </div>
            </article>
            <p v-if="draftCharacters.length === 0" class="muted">这份草稿没有角色。</p>
          </section>

          <template v-else>
            <div class="config-toolbar">
              <div>
                <h2>角色</h2>
                <p class="muted">管理会进入故事上下文的角色设定。</p>
              </div>
              <button type="button" class="primary-button" @click="createCharacter">新增角色</button>
            </div>
            <div class="item-list panel list-panel">
              <p v-if="detailLoading" class="muted">正在加载角色...</p>
              <article v-for="character in characters" :key="character.id" class="list-item">
                <div>
                  <strong>{{ character.name }}</strong>
                  <span>优先级 {{ character.priority }}</span>
                  <p>{{ character.description || '暂无描述' }}</p>
                </div>
                <div class="item-actions">
                  <button type="button" @click="editCharacter(character)">编辑</button>
                  <button type="button" class="danger-button" @click="removeCharacter(character)">删除</button>
                </div>
              </article>
              <p v-if="!detailLoading && characters.length === 0" class="muted">还没有角色配置。</p>
            </div>
          </template>
        </section>

        <section v-if="activeTab === 'world'" class="config-panel">
          <section v-if="isDraftPreview" class="panel preview-panel">
            <h2>AI 草稿世界书预览</h2>
            <p class="muted">这些世界书条目还没有写入数据库。点击“创建故事”后会自动导入。</p>
            <article v-for="(entry, index) in draftWorldInfo" :key="`${entry.keywords.join('-')}-${index}`" class="list-item preview-item" :class="{ disabled: !entry.enabled }">
              <div>
                <strong>{{ entry.keywords.join(', ') }}</strong>
                <span>优先级 {{ entry.priority }} · {{ entry.enabled ? '启用' : '停用' }}</span>
                <p>{{ entry.content }}</p>
              </div>
            </article>
            <p v-if="draftWorldInfo.length === 0" class="muted">这份草稿没有世界书条目。</p>
          </section>

          <template v-else>
            <div class="config-toolbar">
              <div>
                <h2>世界书</h2>
                <p class="muted">管理由关键词触发、加入上下文的设定条目。</p>
              </div>
              <button type="button" class="primary-button" @click="createWorldInfo">新增世界书</button>
            </div>
            <div class="item-list panel list-panel">
              <p v-if="detailLoading" class="muted">正在加载世界书...</p>
              <article v-for="entry in worldInfo" :key="entry.id" class="list-item" :class="{ disabled: !entry.enabled }">
                <div>
                  <strong>{{ entry.keywords.join(', ') }}</strong>
                  <span>优先级 {{ entry.priority }} · {{ entry.enabled ? '启用' : '停用' }}</span>
                  <p>{{ entry.content }}</p>
                </div>
                <div class="item-actions">
                  <button type="button" @click="editWorld(entry)">编辑</button>
                  <button type="button" class="danger-button" @click="removeWorldInfo(entry)">删除</button>
                </div>
              </article>
              <p v-if="!detailLoading && worldInfo.length === 0" class="muted">还没有世界书条目。</p>
            </div>
          </template>
        </section>
      </section>
    </section>

    <div v-if="characterDrawerOpen" class="drawer-layer" @click.self="characterDrawerOpen = false">
      <form class="drawer-panel panel editor-form" @submit.prevent="saveCharacter">
        <header class="drawer-header">
          <h2>{{ characterForm.id ? '编辑角色' : '新增角色' }}</h2>
          <button type="button" class="ghost-button" @click="characterDrawerOpen = false">关闭</button>
        </header>
        <label class="field">
          <span>名称 *</span>
          <input v-model="characterForm.name" placeholder="角色名" />
        </label>
        <label class="field">
          <span>优先级</span>
          <input v-model.number="characterForm.priority" type="number" />
        </label>
        <label class="field">
          <span>描述</span>
          <textarea v-model="characterForm.description" rows="4" placeholder="外观、身份、背景" />
        </label>
        <label class="field">
          <span>性格</span>
          <textarea v-model="characterForm.personality" rows="4" placeholder="说话方式、动机、隐瞒的信息" />
        </label>
        <label class="field">
          <span>示例对话</span>
          <textarea v-model="characterForm.exampleDialogue" rows="5" placeholder="玩家：...&#10;角色：..." />
        </label>
        <div class="button-row">
          <button type="button" :disabled="saving" @click="resetCharacterForm">清空</button>
          <button class="primary-button" type="submit" :disabled="saving">{{ saving ? '保存中...' : '保存角色' }}</button>
        </div>
      </form>
    </div>

    <div v-if="worldDrawerOpen" class="drawer-layer" @click.self="worldDrawerOpen = false">
      <form class="drawer-panel panel editor-form" @submit.prevent="saveWorldInfo">
        <header class="drawer-header">
          <h2>{{ worldForm.id ? '编辑世界书' : '新增世界书' }}</h2>
          <button type="button" class="ghost-button" @click="worldDrawerOpen = false">关闭</button>
        </header>
        <label class="field">
          <span>关键词 *</span>
          <textarea v-model="worldForm.keywordsText" rows="3" placeholder="用逗号或换行分隔，例如：黑猫, 地下室" />
        </label>
        <label class="field">
          <span>优先级</span>
          <input v-model.number="worldForm.priority" type="number" />
        </label>
        <label class="check-field">
          <input v-model="worldForm.enabled" type="checkbox" />
          <span>启用</span>
        </label>
        <label class="field">
          <span>内容 *</span>
          <textarea v-model="worldForm.content" rows="8" placeholder="关键词触发后加入上下文的设定内容" />
        </label>
        <div class="button-row">
          <button type="button" :disabled="saving" @click="resetWorldForm">清空</button>
          <button class="primary-button" type="submit" :disabled="saving">{{ saving ? '保存中...' : '保存世界书' }}</button>
        </div>
      </form>
    </div>
  </main>
</template>

<style scoped>
.admin-page { max-width: 1280px; }
.admin-header {
  display: flex;
  align-items: end;
  justify-content: space-between;
  gap: 16px;
  margin-bottom: 24px;
}
h1 { margin: 14px 0 8px; font-size: clamp(28px, 5vw, 44px); }
h2 { margin: 0; font-size: 20px; }
.admin-layout {
  display: grid;
  grid-template-columns: 280px 1fr;
  gap: 18px;
}
.story-nav {
  display: grid;
  align-content: start;
  gap: 8px;
  padding: 12px;
  max-height: calc(100vh - 160px);
  overflow: auto;
}
.story-nav-item {
  display: grid;
  gap: 4px;
  width: 100%;
  border-radius: 8px;
  padding: 12px;
  background: var(--control);
  color: var(--text);
  text-align: left;
}
.story-nav-item.active {
  background: var(--accent-soft);
  outline: 1px solid var(--accent-border);
}
.new-item {
  border: 1px dashed var(--accent-border);
}
.story-search {
  display: grid;
  grid-template-columns: minmax(0, 1fr) auto;
  gap: 8px;
  margin: 4px 0 6px;
}
.story-search input {
  min-width: 0;
  border: 1px solid var(--border-strong);
  border-radius: 8px;
  background: var(--control);
  color: var(--text-strong);
  padding: 10px 11px;
  outline: none;
}
.story-search button {
  border-radius: 8px;
  padding: 9px 11px;
  background: var(--control-hover);
  color: var(--text);
}
.empty-state {
  margin: 10px 0;
  color: var(--muted);
  font-size: 14px;
}
.story-pager {
  position: sticky;
  bottom: 0;
  display: grid;
  gap: 8px;
  margin-top: 4px;
  padding: 10px 0 0;
  background: var(--panel-elevated);
  color: var(--muted);
  font-size: 12px;
}
.story-page-buttons {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}
.story-page-buttons button {
  border-radius: 8px;
  padding: 7px 9px;
  background: var(--control-hover);
  color: var(--text);
}
.story-page-buttons button.active {
  background: var(--accent);
  color: var(--accent-contrast);
  font-weight: 700;
}
.story-page-buttons button:disabled {
  cursor: not-allowed;
  opacity: 0.55;
}
.story-nav-item span,
.list-item span {
  color: var(--muted);
  font-size: 13px;
}
.editor {
  display: grid;
  gap: 14px;
}
.mode-strip {
  display: flex;
  justify-content: space-between;
  gap: 12px;
  border: 1px solid var(--control-hover);
  border-radius: 8px;
  padding: 12px 14px;
  background: var(--control);
}
.mode-strip span {
  color: var(--muted);
}
.draft-panel,
.preview-panel {
  display: grid;
  gap: 14px;
  padding: 18px;
}
.draft-header {
  display: flex;
  align-items: start;
  justify-content: space-between;
  gap: 14px;
}
.draft-header p {
  margin: 6px 0 0;
}
.draft-chat {
  display: grid;
  gap: 8px;
  max-height: 220px;
  overflow: auto;
}
.draft-controls {
  display: grid;
  grid-template-columns: minmax(0, 1fr) 180px;
  gap: 12px;
}
.draft-message {
  display: grid;
  gap: 4px;
  border-radius: 8px;
  padding: 10px 12px;
  background: var(--control);
}
.draft-message.user {
  background: var(--accent-soft);
}
.draft-message p {
  margin: 0;
  color: var(--muted-strong);
  line-height: 1.5;
  white-space: pre-wrap;
}
.draft-summary,
.progress-box {
  border: 1px solid color-mix(in srgb, var(--success-text) 35%, transparent);
  border-radius: 8px;
  padding: 10px 12px;
  color: var(--success-text);
  background: var(--success-soft);
}
.draft-summary {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
}
.progress-meta {
  display: flex;
  justify-content: space-between;
  gap: 12px;
  margin-bottom: 8px;
}
.progress-track {
  height: 8px;
  overflow: hidden;
  border-radius: 999px;
  background: var(--border);
}
.progress-fill {
  height: 100%;
  border-radius: inherit;
  background: var(--accent);
  transition: width 0.3s ease;
}
.tabs {
  display: flex;
  gap: 8px;
}
.tabs button,
.draft-header button,
.draft-summary button,
.button-row button,
.item-actions button,
.ghost-button,
.primary-button,
.danger-button {
  border-radius: 8px;
  padding: 10px 14px;
  background: var(--control-hover);
  color: var(--text);
}
.tabs button.active {
  background: var(--accent);
  color: var(--accent-contrast);
  font-weight: 700;
}
button:disabled {
  cursor: not-allowed;
  opacity: 0.55;
}
.primary-button {
  background: var(--accent);
  color: var(--accent-contrast);
  font-weight: 700;
}
.danger-button {
  background: var(--danger);
  color: var(--text-strong);
}
.editor-form {
  display: grid;
  gap: 16px;
  padding: 18px;
}
.form-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 12px;
}
.field {
  display: grid;
  gap: 8px;
}
.field span,
.check-field span {
  color: var(--label);
  font-size: 14px;
}
input,
select,
textarea {
  width: 100%;
  border: 1px solid var(--border);
  border-radius: 8px;
  background: var(--control);
  color: var(--text-strong);
  padding: 12px 14px;
  outline: none;
}
textarea {
  line-height: 1.55;
  resize: vertical;
}
input:focus,
select:focus,
textarea:focus { border-color: var(--accent-border); }
.check-field {
  display: flex;
  align-items: center;
  gap: 10px;
}
.check-field input {
  width: 18px;
  height: 18px;
}
.button-row {
  display: flex;
  justify-content: flex-end;
  gap: 10px;
  flex-wrap: wrap;
}
.config-panel {
  display: grid;
  gap: 14px;
}
.config-toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 14px;
  padding: 14px 0 2px;
}
.config-toolbar h2,
.config-toolbar p {
  margin: 0;
}
.config-toolbar p {
  margin-top: 4px;
}
.list-panel {
  padding: 12px;
}
.split-editor {
  display: grid;
  grid-template-columns: minmax(320px, 0.9fr) minmax(320px, 1.1fr);
  gap: 16px;
}
.item-list {
  display: grid;
  align-content: start;
  gap: 10px;
}
.list-item {
  display: grid;
  grid-template-columns: 1fr auto;
  gap: 12px;
  padding: 14px;
  border: 1px solid var(--control-hover);
  border-radius: 8px;
  background: var(--panel);
}
.preview-item {
  grid-template-columns: 1fr;
}
.list-item.disabled {
  opacity: 0.62;
}
.list-item p {
  margin: 8px 0 0;
  color: var(--muted-strong);
  line-height: 1.5;
  white-space: pre-wrap;
}
.item-actions {
  display: flex;
  align-items: start;
  gap: 8px;
}
.drawer-layer {
  position: fixed;
  inset: 0;
  z-index: 50;
  display: flex;
  justify-content: flex-end;
  background: var(--overlay);
}
.drawer-panel {
  width: min(520px, 100%);
  height: 100%;
  overflow: auto;
  border-radius: 0;
  border-right: 0;
  padding: 20px;
  background: var(--panel-elevated);
  box-shadow: -18px 0 50px var(--overlay);
}
.drawer-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
}
.error { color: var(--danger-text); margin-bottom: 12px; }
.notice { color: var(--success-text); margin-bottom: 12px; }
@media (max-width: 900px) {
  .admin-header,
  .draft-header,
  .draft-summary,
  .mode-strip,
  .config-toolbar,
  .button-row {
    align-items: stretch;
    flex-direction: column;
  }
  .admin-layout,
  .split-editor,
  .draft-controls,
  .form-grid,
  .list-item {
    grid-template-columns: 1fr;
  }
  .story-nav {
    max-height: none;
  }
  .item-actions {
    justify-content: flex-end;
  }
  .drawer-panel {
    width: 100%;
  }
}
</style>
