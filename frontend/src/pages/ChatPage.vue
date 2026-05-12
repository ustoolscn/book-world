<script setup lang="ts">
import { computed, nextTick, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import ChatInput from '../components/ChatInput.vue'
import ChatMessage from '../components/ChatMessage.vue'
import { streamChat, type TokenUsage } from '../api/chat'
import { listModels } from '../api/models'
import {
  deleteStorySession,
  getStory,
  listStorySessions,
  loadStorySession,
  saveStorySession,
  type Message,
  type Story,
  type StorySessionRecord
} from '../api/stories'

interface UserProfile {
  name: string
  personality: string
  background: string
  preferences: string
}

interface LocalDraft {
  localId: string
  storySlug: string
  title: string
  messages: Message[]
  profile: UserProfile
  updatedAt: string
  usage?: TokenUsage
  modelUsage?: Record<string, TokenUsage>
  modelStats?: Record<string, ModelPerformanceStats>
  savedSessionId?: string
}

interface ModelPerformanceStats {
  firstTokenMs: number[]
  completionMs: number[]
  speeds: number[]
}

type PopupName = 'settings' | 'records' | 'metrics' | ''

const PROFILE_PREFIX = '__BOOK_WORLD_USER_PROFILE__:'
const emptyTokenUsage = { promptTokens: 0, completionTokens: 0, totalTokens: 0 }
const identityPromptText = '请先打开“身份设置”，至少填写姓名。设置完成后，就可以在下方输入行动或台词继续故事。'

const randomNames = ['林澈', '沈知微', '顾南星', '谢无咎', '许听雨', '陆青岚', '江明烛', '温照夜']
const randomPersonalities = [
  '谨慎敏锐，嘴硬心软，不轻易相信陌生人',
  '开朗直接，行动力强，喜欢用玩笑掩饰紧张',
  '冷静理性，观察细致，遇到危险会先寻找线索',
  '温和善良，但在关键时刻非常固执',
  '好奇心旺盛，容易被神秘事物吸引'
]
const randomBackgrounds = [
  '来自边境小镇的见习药师，正在寻找失踪的亲人。',
  '曾经是贵族家的书记员，因为一封秘密信件卷入事件。',
  '流浪多年的旅人，随身带着一本记不清来源的旧笔记。',
  '刚来到这片土地的外乡人，对本地传说几乎一无所知。',
  '受人委托调查异常事件，但委托人隐瞒了真正目的。'
]
const randomPreferences = [
  '偏好慢节奏探索，多给环境描写，不要替我决定行动。',
  '喜欢悬疑感和角色互动，可以多抛出选择和线索。',
  '希望剧情有压迫感，但保留清晰的行动空间。',
  '偏好细腻情绪和人物关系推进，少一些机械说明。',
  '遇到危险时请描述后果和风险，让我自己选择。'
]

const route = useRoute()
const router = useRouter()
const slug = computed(() => String(route.params.slug))
const story = ref<Story | null>(null)
const draft = ref<LocalDraft | null>(null)
const messages = ref<Message[]>([])
const localRecords = ref<LocalDraft[]>([])
const savedSessions = ref<StorySessionRecord[]>([])
const streaming = ref(false)
const saving = ref(false)
const error = ref('')
const notice = ref('')
const modelOptions = ref<string[]>([])
const modelLoading = ref(false)
const selectedModel = ref(localStorage.getItem('book-world-model') || '')
const selectedThinkingEffort = ref(localStorage.getItem('book-world-thinking-effort') || '')
const thinkingEffortOptions = [
  { value: '', label: '默认' },
  { value: 'low', label: '低' },
  { value: 'medium', label: '中' },
  { value: 'high', label: '高' }
]
const messagesEl = ref<HTMLElement | null>(null)
const followScroll = ref(true)
const showNewMessagePrompt = ref(false)
const activePopup = ref<PopupName>('')
const streamMetrics = ref({
  status: '未开始',
  firstTokenMs: 0,
  completionMs: 0,
  outputChars: 0,
  speed: 0,
  currentUsage: emptyUsage(),
  totalUsage: loadTotalUsage(slug.value)
})

const localKey = computed(() => `book-world-local-chat:${slug.value}`)
const localRecordsKey = computed(() => `book-world-local-records:${slug.value}`)
const hasMessages = computed(() => messages.value.some((msg) => msg.role === 'user' || msg.role === 'assistant'))
const profile = computed(() => draft.value?.profile || emptyProfile())
const hasProfile = computed(() => Boolean(profile.value.name.trim()))
const userName = computed(() => profile.value.name.trim() || '你')
const storyName = computed(() => story.value?.title || '故事')
const profilePrompt = computed(() => buildProfilePrompt(profile.value))
const pendingAssistantMessageId = computed(() => {
  if (!streaming.value) return ''
  const pending = [...messages.value].reverse().find((msg) => msg.role === 'assistant' && !msg.content.trim())
  return pending?.id || ''
})
const modelStatsRows = computed(() => {
  const usage = normalizeModelUsage(draft.value?.modelUsage)
  const stats = normalizeModelStats(draft.value?.modelStats)
  const models = Array.from(new Set([...Object.keys(usage), ...Object.keys(stats)]))
  return models
    .map((model) => ({
      model,
      tokens: usage[model] || emptyUsage(),
      avgFirstTokenMs: trimmedAverage(stats[model]?.firstTokenMs || []),
      avgCompletionMs: trimmedAverage(stats[model]?.completionMs || []),
      avgSpeed: trimmedAverage(stats[model]?.speeds || [])
    }))
    .sort((left, right) => right.tokens.totalTokens - left.tokens.totalTokens || left.model.localeCompare(right.model))
})

onMounted(async () => {
  try {
    const [storyResult, sessions] = await Promise.all([
      getStory(slug.value),
      listStorySessions(slug.value)
    ])
    story.value = storyResult
    savedSessions.value = sessions || []
    restoreLocalRecords()
    restoreLocalDraft()
    await loadModels()
    scrollToBottom(false)
  } catch (err) {
    error.value = err instanceof Error ? err.message : '加载对话失败'
  }
})

watch(
  messages,
  () => {
    handleMessagesChanged()
  },
  { deep: true, flush: 'post' }
)

function togglePopup(name: PopupName) {
  activePopup.value = activePopup.value === name ? '' : name
}

function closePopup() {
  activePopup.value = ''
}

function requestIdentitySetup() {
  if (hasProfile.value) return
  error.value = ''
  notice.value = ''
  ensureIdentityPromptMessage()
  activePopup.value = 'settings'
}

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

function restoreLocalRecords() {
  const raw = localStorage.getItem(localRecordsKey.value)
  if (!raw) {
    localRecords.value = []
    return
  }
  try {
    const parsed = JSON.parse(raw) as LocalDraft[]
    localRecords.value = Array.isArray(parsed)
      ? parsed
          .filter((record) => record.storySlug === slug.value && Array.isArray(record.messages))
          .map((record) => ({
            ...record,
            profile: normalizeProfile(record.profile),
            usage: normalizeUsage(record.usage),
            modelUsage: normalizeModelUsage(record.modelUsage),
            modelStats: normalizeModelStats(record.modelStats)
          }))
      : []
  } catch {
    localStorage.removeItem(localRecordsKey.value)
    localRecords.value = []
  }
}

function persistLocalRecords() {
  localStorage.setItem(localRecordsKey.value, JSON.stringify(localRecords.value))
}

function restoreLocalDraft() {
  const raw = localStorage.getItem(localKey.value)
  if (raw) {
    try {
      const parsed = JSON.parse(raw) as LocalDraft
      if (parsed.storySlug === slug.value && Array.isArray(parsed.messages)) {
        draft.value = {
          ...parsed,
          profile: normalizeProfile(parsed.profile),
          usage: normalizeUsage(parsed.usage),
          modelUsage: normalizeModelUsage(parsed.modelUsage),
          modelStats: normalizeModelStats(parsed.modelStats)
        }
        messages.value = parsed.messages
        streamMetrics.value = { ...streamMetrics.value, totalUsage: draft.value.usage || emptyUsage() }
        return
      }
    } catch {
      localStorage.removeItem(localKey.value)
    }
  }
  startNewRecord()
}

function startNewRecord() {
  const initialMessages = story.value?.openingMessage
    ? [localMessage('assistant', story.value.openingMessage)]
    : []
  draft.value = {
    localId: crypto.randomUUID(),
    storySlug: slug.value,
    title: story.value?.title || '新存档',
    messages: initialMessages,
    profile: emptyProfile(),
    updatedAt: new Date().toISOString(),
    usage: emptyUsage(),
    modelUsage: {},
    modelStats: {}
  }
  messages.value = initialMessages
  streamMetrics.value = { ...streamMetrics.value, currentUsage: emptyUsage(), totalUsage: emptyUsage() }
  ensureIdentityPromptMessage()
  notice.value = ''
  persistDraft()
}

async function loadSaved(record: StorySessionRecord) {
  if (streaming.value) return
  error.value = ''
  notice.value = ''
  try {
    const session = await loadStorySession(slug.value, record.chatSessionId)
    const savedProfile = parseProfileSummary(session.summary)
    draft.value = {
      localId: crypto.randomUUID(),
      storySlug: slug.value,
      title: session.title || record.title || story.value?.title || '已保存存档',
      messages: session.messages || [],
      profile: savedProfile,
      updatedAt: new Date().toISOString(),
      usage: emptyUsage(),
      modelUsage: {},
      modelStats: {},
      savedSessionId: session.chatSessionId
    }
    messages.value = session.messages || []
    streamMetrics.value = { ...streamMetrics.value, currentUsage: emptyUsage(), totalUsage: emptyUsage() }
    notice.value = savedProfile.name ? '已载入存档和绑定身份，可继续本地对话' : '已载入存档，请补充这个对话的故事身份'
    closePopup()
    persistDraft()
  } catch (err) {
    error.value = err instanceof Error ? err.message : '载入存档失败'
  }
}

async function removeSaved(record: StorySessionRecord) {
  if (streaming.value) return
  error.value = ''
  notice.value = ''
  try {
    await deleteStorySession(slug.value, record.chatSessionId)
    savedSessions.value = savedSessions.value.filter((item) => item.chatSessionId !== record.chatSessionId)
    if (draft.value?.savedSessionId === record.chatSessionId) {
      draft.value.savedSessionId = undefined
      persistDraft()
    }
    notice.value = '已删除存档'
  } catch (err) {
    error.value = err instanceof Error ? err.message : '删除存档失败'
  }
}

async function saveCurrentRecord() {
  if (!draft.value || saving.value || streaming.value || !hasMessages.value || !hasProfile.value) return
  error.value = ''
  notice.value = ''
  const title = buildTitle()
  draft.value.title = title
  persistDraft()
  const record = cloneDraft(draft.value)
  const existingIndex = localRecords.value.findIndex((item) => item.localId === record.localId)
  if (existingIndex >= 0) {
    localRecords.value[existingIndex] = record
  } else {
    localRecords.value.unshift(record)
  }
  localRecords.value.sort((a, b) => new Date(b.updatedAt).getTime() - new Date(a.updatedAt).getTime())
  persistLocalRecords()
  notice.value = '已保留到本地存档，可在存档弹窗上传到云'
}

function loadLocalRecord(record: LocalDraft) {
  if (streaming.value) return
  error.value = ''
  notice.value = ''
  draft.value = cloneDraft(record)
  messages.value = record.messages || []
  streamMetrics.value = { ...streamMetrics.value, currentUsage: emptyUsage(), totalUsage: draft.value.usage || emptyUsage() }
  closePopup()
  persistDraft()
  notice.value = '已载入本地存档'
}

function deleteLocalRecord(record: LocalDraft) {
  if (streaming.value) return
  localRecords.value = localRecords.value.filter((item) => item.localId !== record.localId)
  persistLocalRecords()
  notice.value = '已删除本地存档'
}

async function uploadLocalRecord(record: LocalDraft) {
  if (saving.value || streaming.value || !record.profile.name.trim() || !record.messages.length) return
  error.value = ''
  notice.value = ''
  saving.value = true
  try {
    const saved = await saveStorySession(slug.value, {
      title: record.title || buildTitleForMessages(record.messages),
      summary: encodeProfileSummary(record.profile),
      messages: record.messages
    })
    localRecords.value = localRecords.value.map((item) =>
      item.localId === record.localId ? { ...item, savedSessionId: saved.chatSessionId, title: saved.title } : item
    )
    persistLocalRecords()
    if (draft.value?.localId === record.localId) {
      draft.value.savedSessionId = saved.chatSessionId
      draft.value.title = saved.title
      persistDraft()
    }
    savedSessions.value = (await listStorySessions(slug.value)) || []
    notice.value = '已上传到云端数据库'
  } catch (err) {
    error.value = err instanceof Error ? err.message : '上传到云失败'
  } finally {
    saving.value = false
  }
}

async function send(content: string) {
  if (streaming.value || !selectedModel.value) return
  if (!hasProfile.value) {
    error.value = '请先为这个对话填写故事身份，至少需要姓名'
    activePopup.value = 'settings'
    return
  }
  error.value = ''
  notice.value = ''
  streaming.value = true
  const history = messages.value.slice()
  messages.value.push(localMessage('user', content))
  messages.value.push(localMessage('assistant', ''))
  const assistantIndex = messages.value.length - 1
  persistDraft()
  scrollToBottom(false)
  try {
    localStorage.setItem('book-world-model', selectedModel.value)
    localStorage.setItem('book-world-thinking-effort', selectedThinkingEffort.value)
    const requestModel = selectedModel.value
    const startedAt = performance.now()
    let firstTokenAt = 0
    let outputChars = 0
    let responseUsage = emptyUsage()
    const totalUsageBefore = draft.value?.usage || emptyUsage()
    streamMetrics.value = {
      status: '生成中',
      firstTokenMs: 0,
      completionMs: 0,
      outputChars: 0,
      speed: 0,
      currentUsage: emptyUsage(),
      totalUsage: totalUsageBefore
    }
    await streamChat(
      slug.value,
      content,
      requestModel,
      selectedThinkingEffort.value,
      history,
      profilePrompt.value,
      (delta) => {
        const current = messages.value[assistantIndex]
        if (!current) return
        const now = performance.now()
        if (!firstTokenAt && delta) {
          firstTokenAt = now
        }
        outputChars += delta.length
        const elapsedSeconds = Math.max((now - startedAt) / 1000, 0.001)
        streamMetrics.value = {
          status: '生成中',
          firstTokenMs: firstTokenAt ? Math.round(firstTokenAt - startedAt) : 0,
          completionMs: Math.round(now - startedAt),
          outputChars,
          speed: Number((outputChars / elapsedSeconds).toFixed(1)),
          currentUsage: responseUsage,
          totalUsage: draft.value?.usage || totalUsageBefore
        }
        messages.value[assistantIndex] = {
          ...current,
          content: current.content + delta,
          tokenEstimate: current.content.length + delta.length
        }
        handleStreamingContentChanged()
      },
      (usage) => {
        responseUsage = normalizeUsage(usage)
        const newTotalUsage = addUsage(totalUsageBefore, responseUsage)
        if (draft.value) {
          draft.value.usage = newTotalUsage
          draft.value.modelUsage = addModelUsage(draft.value.modelUsage, requestModel, responseUsage)
        }
        streamMetrics.value = {
          ...streamMetrics.value,
          currentUsage: responseUsage,
          totalUsage: newTotalUsage
        }
        persistDraft()
      }
    )
    const endedAt = performance.now()
    const elapsedSeconds = Math.max((endedAt - startedAt) / 1000, 0.001)
    const finalFirstTokenMs = firstTokenAt ? Math.round(firstTokenAt - startedAt) : 0
    const finalCompletionMs = Math.round(endedAt - startedAt)
    const finalSpeed = Number((outputChars / elapsedSeconds).toFixed(1))
    if (draft.value) {
      draft.value.modelStats = addModelStats(draft.value.modelStats, requestModel, {
        firstTokenMs: finalFirstTokenMs,
        completionMs: finalCompletionMs,
        speed: finalSpeed
      })
    }
    streamMetrics.value = {
      status: '已完成',
      firstTokenMs: finalFirstTokenMs,
      completionMs: finalCompletionMs,
      outputChars,
      speed: finalSpeed,
      currentUsage: responseUsage,
      totalUsage: draft.value?.usage || totalUsageBefore
    }
  } catch (err) {
    error.value = err instanceof Error ? err.message : '发送失败'
    streamMetrics.value = { ...streamMetrics.value, status: '失败' }
    messages.value = messages.value.filter((_, index) => index !== assistantIndex)
  } finally {
    streaming.value = false
    persistDraft()
  }
}

function updateProfile(field: keyof UserProfile, value: string) {
  if (!draft.value) return
  draft.value.profile = { ...draft.value.profile, [field]: value }
  persistDraft()
}

function randomizeProfile() {
  if (!draft.value || streaming.value) return
  draft.value.profile = {
    name: pick(randomNames),
    personality: pick(randomPersonalities),
    background: pick(randomBackgrounds),
    preferences: pick(randomPreferences)
  }
  notice.value = '已随机生成这个对话的故事身份'
  persistDraft()
}

function ensureIdentityPromptMessage() {
  messages.value = messages.value.filter((msg) => !(msg.role === 'system' && msg.content === identityPromptText))
  notice.value = identityPromptText
}

function persistDraft() {
  if (!draft.value) return
  draft.value.messages = messages.value
  draft.value.profile = normalizeProfile(draft.value.profile)
  draft.value.usage = normalizeUsage(draft.value.usage)
  draft.value.modelUsage = normalizeModelUsage(draft.value.modelUsage)
  draft.value.modelStats = normalizeModelStats(draft.value.modelStats)
  draft.value.updatedAt = new Date().toISOString()
  localStorage.setItem(localKey.value, JSON.stringify(draft.value))
}

function buildTitle() {
  return buildTitleForMessages(messages.value)
}

function buildTitleForMessages(source: Message[]) {
  const firstUser = source.find((msg) => msg.role === 'user' && msg.content.trim())
  if (!firstUser) return story.value?.title || '故事存档'
  const text = firstUser.content.trim()
  return text.length > 24 ? `${text.slice(0, 24)}...` : text
}

function cloneDraft(value: LocalDraft): LocalDraft {
  return {
    ...value,
    messages: value.messages.map((message) => ({ ...message })),
    profile: normalizeProfile(value.profile),
    usage: normalizeUsage(value.usage),
    modelUsage: normalizeModelUsage(value.modelUsage),
    modelStats: normalizeModelStats(value.modelStats)
  }
}

function formatTime(value: string) {
  if (!value) return ''
  return new Date(value).toLocaleString()
}

function localMessage(role: 'user' | 'assistant' | 'system', content: string): Message {
  return { id: crypto.randomUUID(), role, content, tokenEstimate: content.length, createdAt: new Date().toISOString() }
}

function emptyProfile(): UserProfile {
  return { name: '', personality: '', background: '', preferences: '' }
}

function emptyUsage(): TokenUsage {
  return { ...emptyTokenUsage }
}

function loadTotalUsage(_storySlug: string): TokenUsage {
  return emptyUsage()
}

function normalizeUsage(value?: Partial<TokenUsage> | null): TokenUsage {
  return {
    promptTokens: Number(value?.promptTokens || 0),
    completionTokens: Number(value?.completionTokens || 0),
    totalTokens: Number(value?.totalTokens || 0)
  }
}

function addUsage(left: TokenUsage, right: TokenUsage): TokenUsage {
  return {
    promptTokens: left.promptTokens + right.promptTokens,
    completionTokens: left.completionTokens + right.completionTokens,
    totalTokens: left.totalTokens + right.totalTokens
  }
}

function normalizeModelUsage(value?: Record<string, Partial<TokenUsage>> | null): Record<string, TokenUsage> {
  const normalized: Record<string, TokenUsage> = {}
  for (const [model, usage] of Object.entries(value || {})) {
    const name = model.trim()
    if (!name) continue
    normalized[name] = normalizeUsage(usage)
  }
  return normalized
}

function normalizeModelStats(value?: Record<string, Partial<ModelPerformanceStats>> | null): Record<string, ModelPerformanceStats> {
  const normalized: Record<string, ModelPerformanceStats> = {}
  for (const [model, stats] of Object.entries(value || {})) {
    const name = model.trim()
    if (!name) continue
    normalized[name] = {
      firstTokenMs: normalizeNumberList(stats?.firstTokenMs),
      completionMs: normalizeNumberList(stats?.completionMs),
      speeds: normalizeNumberList(stats?.speeds)
    }
  }
  return normalized
}

function normalizeNumberList(values?: number[] | null) {
  return Array.isArray(values)
    ? values.map((value) => Number(value)).filter((value) => Number.isFinite(value) && value > 0)
    : []
}

function addModelUsage(value: Record<string, TokenUsage> | undefined, model: string, usage: TokenUsage) {
  const name = model.trim() || '默认模型'
  const normalized = normalizeModelUsage(value)
  normalized[name] = addUsage(normalized[name] || emptyUsage(), usage)
  return normalized
}

function addModelStats(
  value: Record<string, ModelPerformanceStats> | undefined,
  model: string,
  sample: { firstTokenMs: number; completionMs: number; speed: number }
) {
  const name = model.trim() || '默认模型'
  const normalized = normalizeModelStats(value)
  const current = normalized[name] || { firstTokenMs: [], completionMs: [], speeds: [] }
  normalized[name] = {
    firstTokenMs: [...current.firstTokenMs, sample.firstTokenMs].filter((value) => value > 0),
    completionMs: [...current.completionMs, sample.completionMs].filter((value) => value > 0),
    speeds: [...current.speeds, sample.speed].filter((value) => value > 0)
  }
  return normalized
}

function trimmedAverage(values: number[]) {
  const normalized = normalizeNumberList(values)
  if (normalized.length === 0) return 0
  const sorted = [...normalized].sort((left, right) => left - right)
  const sample = sorted.length > 2 ? sorted.slice(1, -1) : sorted
  return sample.reduce((sum, value) => sum + value, 0) / sample.length
}

function formatUsage(value: TokenUsage) {
  return `${value.promptTokens}/${value.completionTokens}/${value.totalTokens}`
}

function formatMs(value: number) {
  if (!value) return '-'
  if (value >= 1000) return `${(value / 1000).toFixed(2)}s`
  return `${Math.round(value)}ms`
}

function formatSpeed(value: number) {
  return value ? `${value.toFixed(1)}字/秒` : '-'
}

function normalizeProfile(value?: Partial<UserProfile> | null): UserProfile {
  return { ...emptyProfile(), ...(value || {}) }
}

function pick(values: string[]) {
  return values[Math.floor(Math.random() * values.length)]
}

function buildProfilePrompt(value: UserProfile) {
  const lines = [
    `姓名：${value.name.trim()}`,
    value.personality.trim() ? `性格：${value.personality.trim()}` : '',
    value.background.trim() ? `背景：${value.background.trim()}` : '',
    value.preferences.trim() ? `偏好/补充：${value.preferences.trim()}` : ''
  ].filter(Boolean)
  return lines.join('\n')
}

function encodeProfileSummary(value: UserProfile) {
  return PROFILE_PREFIX + JSON.stringify(normalizeProfile(value))
}

function parseProfileSummary(summary: string) {
  if (!summary?.startsWith(PROFILE_PREFIX)) return emptyProfile()
  try {
    return normalizeProfile(JSON.parse(summary.slice(PROFILE_PREFIX.length)))
  } catch {
    return emptyProfile()
  }
}

function handleMessagesChanged() {
  const shouldFollow = followScroll.value || isMessagesAtBottom()
  if (shouldFollow) {
    followScroll.value = true
    showNewMessagePrompt.value = false
    scrollToBottom(true)
    return
  }
  showNewMessagePrompt.value = true
}

function handleStreamingContentChanged() {
  if (followScroll.value || isMessagesAtBottom()) {
    followScroll.value = true
    showNewMessagePrompt.value = false
    scrollToBottom(true)
    return
  }
  showNewMessagePrompt.value = true
}

function pauseAutoFollow() {
  if (streaming.value) {
    followScroll.value = false
  }
}

function onMessagesScroll() {
  followScroll.value = isMessagesAtBottom()
  if (followScroll.value) {
    showNewMessagePrompt.value = false
  }
}

function isMessagesAtBottom() {
  const el = messagesEl.value
  if (!el) return true
  return el.scrollHeight - el.scrollTop - el.clientHeight <= 48
}

function jumpToBottom() {
  followScroll.value = true
  showNewMessagePrompt.value = false
  scrollToBottom(false)
}

async function scrollToBottom(smooth: boolean) {
  await nextTick()
  requestAnimationFrame(() => {
    const el = messagesEl.value
    if (!el) return
    el.scrollTo({ top: el.scrollHeight, behavior: smooth ? 'smooth' : 'auto' })
  })
}
</script>

<template>
  <main class="chat-page">
    <header class="compact-header panel">
      <button class="ghost-button back-button" @click="router.push('/stories')">返回</button>
      <div class="title-block">
        <h1>{{ storyName }}</h1>
      </div>
      <nav class="header-actions" aria-label="对话工具">
        <button :class="{ active: activePopup === 'settings' }" @click="togglePopup('settings')">身份设置</button>
        <button :class="{ active: activePopup === 'records' }" @click="togglePopup('records')">存档</button>
        <button :class="{ active: activePopup === 'metrics' }" @click="togglePopup('metrics')">模型统计</button>
      </nav>
    </header>

    <section class="model-bar panel" aria-label="模型设置">
      <label class="compact-field">
        <span>{{ modelLoading ? '加载模型...' : '模型' }}</span>
        <select v-model="selectedModel" :disabled="streaming || modelLoading || modelOptions.length === 0">
          <option v-for="model in modelOptions" :key="model" :value="model">{{ model }}</option>
        </select>
      </label>
      <label class="compact-field effort-field">
        <span>思考程度</span>
        <select v-model="selectedThinkingEffort" :disabled="streaming">
          <option v-for="option in thinkingEffortOptions" :key="option.value" :value="option.value">{{ option.label }}</option>
        </select>
      </label>
    </section>

    <section ref="messagesEl" class="messages panel" @scroll="onMessagesScroll" @wheel.passive="pauseAutoFollow" @touchmove.passive="pauseAutoFollow">
      <ChatMessage
        v-for="message in messages"
        :key="message.id"
        :role="message.role"
        :content="message.content"
        :user-name="userName"
        :assistant-name="storyName"
        :pending="message.id === pendingAssistantMessageId"
      />
      <button v-if="showNewMessagePrompt" class="new-message-prompt" @click="jumpToBottom">有新消息，点击跳到底部</button>
    </section>

    <div class="tip-bar" :class="{ error: Boolean(error), notice: Boolean(notice) }">
      <span>{{ error || notice || ' ' }}</span>
    </div>

    <footer class="composer">
      <ChatInput :disabled="streaming" :blocked="!hasProfile" @blocked="requestIdentitySetup" @send="send" />
    </footer>

    <div v-if="activePopup" class="popup-layer" @click.self="closePopup">
      <section class="popup-panel panel" role="dialog" aria-modal="true">
        <header class="popup-header">
          <h2 v-if="activePopup === 'settings'">身份设置</h2>
          <h2 v-else-if="activePopup === 'records'">存档</h2>
          <h2 v-else>模型统计</h2>
          <button class="ghost-button" @click="closePopup">关闭</button>
        </header>

        <div v-if="activePopup === 'settings'" class="popup-body settings-body">
          <section class="popup-section">
            <div class="section-title">
              <h3>本对话身份</h3>
              <button :disabled="streaming" @click="randomizeProfile">随机</button>
            </div>
            <label class="field">
              <span>姓名 *</span>
              <input :value="profile.name" :disabled="streaming" placeholder="例如：林澈" @input="updateProfile('name', ($event.target as HTMLInputElement).value)" />
            </label>
            <label class="field">
              <span>性格</span>
              <input :value="profile.personality" :disabled="streaming" placeholder="谨慎、好奇、嘴硬心软..." @input="updateProfile('personality', ($event.target as HTMLInputElement).value)" />
            </label>
            <label class="field">
              <span>背景</span>
              <textarea :value="profile.background" :disabled="streaming" placeholder="这个身份在本故事中的背景" @input="updateProfile('background', ($event.target as HTMLTextAreaElement).value)" />
            </label>
            <label class="field">
              <span>偏好 / 补充</span>
              <textarea :value="profile.preferences" :disabled="streaming" placeholder="叙事偏好、禁忌、想要的风格" @input="updateProfile('preferences', ($event.target as HTMLTextAreaElement).value)" />
            </label>
            <p v-if="!hasProfile" class="identity-warning">至少填写姓名后才能发送或保留。</p>
          </section>
        </div>

        <div v-else-if="activePopup === 'records'" class="popup-body records-body">
          <section class="button-row archive-actions">
            <button :disabled="streaming" @click="startNewRecord">新开存档</button>
            <button class="primary-button" :disabled="streaming || saving || !hasMessages || !hasProfile" @click="saveCurrentRecord">
              {{ saving ? '保留中...' : '保留到本地' }}
            </button>
          </section>

          <section class="popup-section">
            <h3>本地存档</h3>
            <p v-if="localRecords.length === 0" class="muted empty">暂无本地存档</p>
            <article v-for="record in localRecords" :key="record.localId" class="record-card">
              <button class="record-main" @click="loadLocalRecord(record)">
                <strong>{{ record.title || '未命名存档' }}</strong>
                <span>{{ record.messages.length }} 条消息 · {{ formatTime(record.updatedAt) }}</span>
                <span>{{ record.savedSessionId ? '已上传云端' : '仅本地' }}</span>
              </button>
              <div class="record-actions">
                <button :disabled="saving || streaming || !record.profile.name || record.messages.length === 0" @click="uploadLocalRecord(record)">
                  {{ record.savedSessionId ? '再上传' : '上传到云' }}
                </button>
                <button class="delete" :disabled="streaming" @click="deleteLocalRecord(record)">删除</button>
              </div>
            </article>
          </section>

          <section class="popup-section">
            <h3>云端存档</h3>
            <p v-if="savedSessions.length === 0" class="muted empty">暂无云端存档</p>
            <article v-for="record in savedSessions" :key="record.chatSessionId" class="record-card">
              <button class="record-main" @click="loadSaved(record)">
                <strong>{{ record.title || '未命名存档' }}</strong>
                <span>{{ record.messageCount }} 条消息 · {{ formatTime(record.updatedAt) }}</span>
              </button>
              <button class="delete" :disabled="streaming" @click="removeSaved(record)">删除</button>
            </article>
          </section>
        </div>

        <div v-else class="popup-body metrics-body">
          <section class="popup-section model-usage-section">
            <h3>按模型统计</h3>
            <p v-if="modelStatsRows.length === 0" class="muted empty">暂无模型统计</p>
            <article v-for="row in modelStatsRows" :key="row.model" class="model-usage-card">
              <header class="model-usage-header">
                <strong>{{ row.model }}</strong>
                <span>Token {{ formatUsage(row.tokens) }}</span>
              </header>
              <div class="model-metric-grid">
                <div>
                  <span>平均首字</span>
                  <strong>{{ formatMs(row.avgFirstTokenMs) }}</strong>
                </div>
                <div>
                  <span>平均完成</span>
                  <strong>{{ formatMs(row.avgCompletionMs) }}</strong>
                </div>
                <div>
                  <span>平均速度</span>
                  <strong>{{ formatSpeed(row.avgSpeed) }}</strong>
                </div>
              </div>
            </article>
          </section>
          <p class="metrics-hint">平均值会在样本超过 2 条时去掉最快和最慢后计算；Token 格式：输入 / 输出 / 总计。</p>
        </div>
      </section>
    </div>
  </main>
</template>

<style scoped>
.chat-page {
  height: 100dvh;
  min-height: 0;
  display: grid;
  grid-template-rows: auto auto 1fr auto auto;
  gap: 8px;
  padding: 8px;
  overflow: hidden;
}
.compact-header {
  min-width: 0;
  display: grid;
  grid-template-columns: auto minmax(0, 1fr) auto;
  align-items: center;
  gap: 8px;
  padding: 8px;
  border-radius: 16px;
}
.title-block { min-width: 0; }
h1 {
  margin: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-size: 16px;
  line-height: 1.2;
}
p { margin: 0; }
.identity-warning { color: var(--accent-strong); }
.header-actions {
  display: flex;
  gap: 6px;
}
button {
  border-radius: 999px;
  padding: 8px 10px;
  background: var(--control-hover);
  color: var(--text);
  white-space: nowrap;
}
button.active,
.primary-button {
  background: var(--accent);
  color: var(--accent-contrast);
  font-weight: 700;
}
button:disabled,
input:disabled,
textarea:disabled,
select:disabled {
  cursor: not-allowed;
  opacity: 0.5;
}
.ghost-button { background: var(--control); }
.back-button { padding-inline: 9px; }
.model-bar {
  min-width: 0;
  display: grid;
  grid-template-columns: minmax(220px, 1fr) minmax(120px, 180px);
  gap: 8px;
  align-items: end;
  padding: 8px;
  border-radius: 16px;
}
.compact-field {
  min-width: 0;
  display: grid;
  gap: 4px;
  color: var(--muted-strong);
  font-size: 12px;
}
.compact-field select {
  height: 36px;
  padding-block: 7px;
}
.messages {
  min-height: 0;
  display: flex;
  flex-direction: column;
  gap: 10px;
  overflow-y: auto;
  overscroll-behavior: contain;
  padding: 10px;
  border-radius: 16px;
  scroll-behavior: smooth;
  position: relative;
}
.new-message-prompt {
  position: sticky;
  bottom: 6px;
  z-index: 10;
  align-self: center;
  margin-top: -2px;
  border: 1px solid color-mix(in srgb, var(--success-text) 40%, transparent);
  background: color-mix(in srgb, var(--success-soft) 76%, var(--panel-elevated));
  color: var(--success-text);
  box-shadow: 0 10px 24px color-mix(in srgb, var(--bg-deep) 45%, transparent);
}
.composer { min-height: 0; }
.tip-bar {
  min-height: 32px;
  display: flex;
  align-items: center;
  border-radius: 12px;
  padding: 7px 10px;
  color: var(--muted);
  font-size: 13px;
  background: var(--control);
}
.tip-bar span {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.tip-bar.error {
  color: var(--danger-text);
  background: var(--danger-soft);
}
.tip-bar.notice {
  color: var(--success-text);
  background: var(--success-soft);
}
.popup-layer {
  position: fixed;
  inset: 0;
  z-index: 20;
  display: flex;
  justify-content: flex-end;
  padding: 8px;
  background: var(--overlay);
}
.popup-panel {
  width: min(390px, 100%);
  min-height: 0;
  max-height: 100%;
  display: grid;
  grid-template-rows: auto 1fr;
  overflow: hidden;
  border-radius: 18px;
  background: var(--panel-elevated);
}
.popup-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 10px;
  padding: 10px 12px;
  border-bottom: 1px solid var(--control-hover);
}
.popup-header h2,
.popup-section h3,
.section-title h3 {
  margin: 0;
  font-size: 15px;
}
.popup-body {
  min-height: 0;
  overflow-y: auto;
  overscroll-behavior: contain;
  padding: 12px;
}
.settings-body,
.records-body {
  display: grid;
  align-content: start;
  gap: 14px;
}
.control-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 10px;
}
.popup-section {
  display: grid;
  gap: 10px;
}
.section-title,
.button-row {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 8px;
}
.archive-actions {
  padding-bottom: 12px;
  border-bottom: 1px solid var(--control-hover);
}
.field {
  display: grid;
  gap: 5px;
  color: var(--muted-strong);
  font-size: 12px;
}
select,
input,
textarea {
  width: 100%;
  min-width: 0;
  padding: 9px 10px;
  border: 1px solid var(--border-strong);
  border-radius: 12px;
  outline: 0;
  color: var(--text);
  background: var(--control-strong);
}
textarea {
  min-height: 66px;
  max-height: 140px;
  resize: vertical;
}
.record-card {
  display: grid;
  grid-template-columns: minmax(0, 1fr) auto;
  gap: 8px;
  padding: 8px;
  border: 1px solid var(--control-hover);
  border-radius: 14px;
  background: var(--control);
}
.record-main {
  min-width: 0;
  display: grid;
  gap: 4px;
  text-align: left;
  border-radius: 12px;
}
.record-main strong,
.record-main span {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.record-main span {
  color: var(--muted-strong);
  font-size: 12px;
}
.record-actions {
  display: grid;
  gap: 6px;
}
.delete { color: var(--danger-text); }
.empty {
  color: var(--muted);
  font-size: 13px;
}
.metrics-hint {
  color: var(--muted);
  font-size: 12px;
}
.model-usage-card {
  display: grid;
  gap: 12px;
  padding: 14px;
  border: 1px solid var(--control-hover);
  border-radius: 14px;
  background: var(--control);
}
.model-usage-header {
  min-width: 0;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding-bottom: 10px;
  border-bottom: 1px solid var(--control-hover);
}
.model-usage-header strong {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  color: var(--text);
  font-size: 15px;
}
.model-usage-header span {
  color: var(--muted-strong);
  font-size: 12px;
  white-space: nowrap;
}
.model-metric-grid {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 8px;
}
.model-metric-grid div {
  display: grid;
  gap: 5px;
  min-width: 0;
  padding: 10px;
  border-radius: 12px;
  background: var(--control);
}
.model-metric-grid span {
  color: var(--muted);
  font-size: 12px;
}
.model-metric-grid strong {
  color: var(--text);
  font-size: 17px;
  line-height: 1.25;
}
.metrics-hint { margin-top: 12px; }
.messages,
.popup-body,
textarea {
  scrollbar-width: thin;
  scrollbar-color: var(--scrollbar) transparent;
}
.messages::-webkit-scrollbar,
.popup-body::-webkit-scrollbar,
textarea::-webkit-scrollbar { width: 8px; }
.messages::-webkit-scrollbar-track,
.popup-body::-webkit-scrollbar-track,
textarea::-webkit-scrollbar-track { background: transparent; }
.messages::-webkit-scrollbar-thumb,
.popup-body::-webkit-scrollbar-thumb,
textarea::-webkit-scrollbar-thumb {
  border: 2px solid transparent;
  border-radius: 999px;
  background: var(--scrollbar);
  background-clip: content-box;
}
@media (max-width: 520px) {
  .chat-page {
    gap: 6px;
    padding: 6px;
  }
  .compact-header { grid-template-columns: auto minmax(0, 1fr); }
  .model-bar { grid-template-columns: 1fr; }
  .header-actions {
    grid-column: 1 / -1;
    justify-content: space-between;
  }
  .header-actions button { flex: 1; }
  .popup-layer {
    align-items: flex-end;
    padding: 6px;
  }
  .popup-panel {
    width: 100%;
    max-height: min(86dvh, 620px);
  }
  .control-grid,
  .model-usage-card { grid-template-columns: 1fr; }
  .model-usage-header {
    align-items: flex-start;
    flex-direction: column;
  }
  .model-metric-grid {
    grid-template-columns: 1fr;
  }
}
</style>
