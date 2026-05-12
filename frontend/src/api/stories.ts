import { apiFetch } from './client'

export interface Story {
  id: string
  slug: string
  title: string
  description: string
  coverUrl: string
  systemPrompt?: string
  scenario?: string
  stylePrompt?: string
  openingMessage?: string
  likeCount: number
  liked: boolean
}

export interface Message {
  id: string
  role: 'user' | 'assistant' | 'system'
  content: string
  tokenEstimate: number
  createdAt: string
}

export interface StorySession {
  chatSessionId: string
  title: string
  summary: string
  messages: Message[]
  createdAt: string
  updatedAt: string
}

export interface StorySessionRecord {
  chatSessionId: string
  title: string
  summary: string
  messageCount: number
  createdAt: string
  updatedAt: string
}

export interface StorySettingsPayload {
  slug?: string
  title: string
  description?: string
  coverUrl?: string
  systemPrompt: string
  scenario?: string
  stylePrompt?: string
  openingMessage?: string
}

export interface Character {
  id: string
  storyId: string
  name: string
  description: string
  personality: string
  exampleDialogue: string
  priority: number
}

export interface WorldInfo {
  id: string
  storyId: string
  keywords: string[]
  content: string
  priority: number
  enabled: boolean
}

export interface StoryDraft {
  story: StorySettingsPayload & { slug: string }
  characters: Array<Omit<Character, 'id' | 'storyId'>>
  worldInfo: Array<Omit<WorldInfo, 'id' | 'storyId'>>
}

export interface DraftChatMessage {
  role: 'user' | 'assistant'
  content: string
}

export interface StoriesPage {
  items: Story[]
  total: number
  limit: number
  offset: number
}

export function listStories() {
  return apiFetch<Story[]>('/api/stories')
}

export function listStoriesPage(limit: number, offset: number, options: { sort?: string; search?: string } = {}) {
  const params = new URLSearchParams({
    limit: String(limit),
    offset: String(offset)
  })
  if (options.sort) params.set('sort', options.sort)
  if (options.search) params.set('search', options.search)
  return apiFetch<StoriesPage>(`/api/stories?${params.toString()}`)
}

export function getStory(slug: string) {
  return apiFetch<Story>(`/api/stories/${slug}`)
}

export function getStorySettings(slug: string) {
  return apiFetch<Story>(`/api/stories/${slug}/settings`)
}

export function updateStorySettings(slug: string, payload: StorySettingsPayload) {
  return apiFetch<Story>(`/api/stories/${slug}/settings`, {
    method: 'PATCH',
    body: JSON.stringify(payload)
  })
}

export function createAdminStory(payload: StorySettingsPayload & { slug: string }) {
  return apiFetch<Story>('/api/admin/stories', {
    method: 'POST',
    body: JSON.stringify(payload)
  })
}

export function updateAdminStory(slug: string, payload: StorySettingsPayload & { slug: string }) {
  return apiFetch<Story>(`/api/admin/stories/${slug}`, {
    method: 'PATCH',
    body: JSON.stringify(payload)
  })
}

export function deleteAdminStory(slug: string) {
  return apiFetch<{ ok: boolean }>(`/api/admin/stories/${slug}`, { method: 'DELETE' })
}

export function toggleStoryLike(slug: string) {
  return apiFetch<{ liked: boolean; likeCount: number }>(`/api/stories/${slug}/like`, { method: 'POST' })
}

export function listAdminStories() {
  return apiFetch<Story[]>('/api/admin/stories')
}

export function listAdminStoriesPage(limit: number, offset: number, options: { search?: string } = {}) {
  const params = new URLSearchParams({
    limit: String(limit),
    offset: String(offset)
  })
  if (options.search) params.set('search', options.search)
  return apiFetch<StoriesPage>(`/api/admin/stories?${params.toString()}`)
}

export function generateAdminStoryDraft(payload: {
  messages: DraftChatMessage[]
  message: string
  model?: string
  thinkingEffort?: string
}) {
  const controller = new AbortController()
  const timeout = window.setTimeout(() => controller.abort(), 120000)
  return apiFetch<{ reply: string; draft: StoryDraft }>('/api/admin/story-drafts', {
    method: 'POST',
    signal: controller.signal,
    body: JSON.stringify(payload)
  }).catch((err) => {
    if (err instanceof DOMException && err.name === 'AbortError') {
      throw new Error('生成超时，请换一个模型或降低思考等级后重试')
    }
    throw err
  }).finally(() => {
    window.clearTimeout(timeout)
  })
}

export function listAdminCharacters(slug: string) {
  return apiFetch<Character[]>(`/api/admin/stories/${slug}/characters`)
}

export function createAdminCharacter(slug: string, payload: Omit<Character, 'id' | 'storyId'>) {
  return apiFetch<Character>(`/api/admin/stories/${slug}/characters`, {
    method: 'POST',
    body: JSON.stringify(payload)
  })
}

export function updateAdminCharacter(id: string, payload: Omit<Character, 'id' | 'storyId'>) {
  return apiFetch<Character>(`/api/admin/characters/${id}`, {
    method: 'PATCH',
    body: JSON.stringify(payload)
  })
}

export function deleteAdminCharacter(id: string) {
  return apiFetch<{ ok: boolean }>(`/api/admin/characters/${id}`, { method: 'DELETE' })
}

export function listAdminWorldInfo(slug: string) {
  return apiFetch<WorldInfo[]>(`/api/admin/stories/${slug}/world-info`)
}

export function createAdminWorldInfo(slug: string, payload: Omit<WorldInfo, 'id' | 'storyId'>) {
  return apiFetch<WorldInfo>(`/api/admin/stories/${slug}/world-info`, {
    method: 'POST',
    body: JSON.stringify(payload)
  })
}

export function updateAdminWorldInfo(id: string, payload: Omit<WorldInfo, 'id' | 'storyId'>) {
  return apiFetch<WorldInfo>(`/api/admin/world-info/${id}`, {
    method: 'PATCH',
    body: JSON.stringify(payload)
  })
}

export function deleteAdminWorldInfo(id: string) {
  return apiFetch<{ ok: boolean }>(`/api/admin/world-info/${id}`, { method: 'DELETE' })
}

export function listStorySessions(slug: string) {
  return apiFetch<StorySessionRecord[]>(`/api/stories/${slug}/sessions`)
}

export function loadStorySession(slug: string, sessionId: string) {
  return apiFetch<StorySession>(`/api/stories/${slug}/sessions/${sessionId}`)
}

export function saveStorySession(slug: string, payload: { title: string; summary?: string; messages: Message[] }) {
  return apiFetch<StorySessionRecord>(`/api/stories/${slug}/sessions`, {
    method: 'POST',
    body: JSON.stringify(payload)
  })
}

export function deleteStorySession(slug: string, sessionId: string) {
  return apiFetch<{ ok: boolean }>(`/api/stories/${slug}/sessions/${sessionId}`, { method: 'DELETE' })
}
