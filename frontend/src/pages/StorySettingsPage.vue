<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { getStorySettings, updateStorySettings, type StorySettingsPayload } from '../api/stories'

const route = useRoute()
const router = useRouter()
const slug = computed(() => String(route.params.slug || ''))
const loading = ref(true)
const saving = ref(false)
const error = ref('')
const notice = ref('')

const form = reactive<StorySettingsPayload>({
  title: '',
  description: '',
  coverUrl: '',
  systemPrompt: '',
  scenario: '',
  stylePrompt: '',
  openingMessage: ''
})

onMounted(loadSettings)

async function loadSettings() {
  loading.value = true
  error.value = ''
  try {
    const story = await getStorySettings(slug.value)
    form.title = story.title || ''
    form.description = story.description || ''
    form.coverUrl = story.coverUrl || ''
    form.systemPrompt = story.systemPrompt || ''
    form.scenario = story.scenario || ''
    form.stylePrompt = story.stylePrompt || ''
    form.openingMessage = story.openingMessage || ''
  } catch (err) {
    error.value = err instanceof Error ? err.message : '故事参数加载失败'
  } finally {
    loading.value = false
  }
}

async function saveSettings() {
  if (!form.title.trim() || !form.systemPrompt.trim()) {
    error.value = '标题和故事规则不能为空'
    return
  }
  saving.value = true
  error.value = ''
  notice.value = ''
  try {
    const story = await updateStorySettings(slug.value, {
      title: form.title,
      description: form.description,
      coverUrl: form.coverUrl,
      systemPrompt: form.systemPrompt,
      scenario: form.scenario,
      stylePrompt: form.stylePrompt,
      openingMessage: form.openingMessage
    })
    form.title = story.title || ''
    form.description = story.description || ''
    form.coverUrl = story.coverUrl || ''
    form.systemPrompt = story.systemPrompt || ''
    form.scenario = story.scenario || ''
    form.stylePrompt = story.stylePrompt || ''
    form.openingMessage = story.openingMessage || ''
    notice.value = '故事参数已保存，新的对话会使用更新后的设定'
  } catch (err) {
    error.value = err instanceof Error ? err.message : '保存失败'
  } finally {
    saving.value = false
  }
}
</script>

<template>
  <main class="page settings-page">
    <header class="header">
      <div>
        <button class="ghost-button" @click="router.push('/stories')">返回故事集合</button>
        <h1>故事参数</h1>
        <p class="muted">修改故事标题、开场白、系统规则、场景和风格提示。</p>
      </div>
      <button class="primary-button" :disabled="loading || saving" @click="saveSettings">
        {{ saving ? '保存中...' : '保存参数' }}
      </button>
    </header>

    <p v-if="loading" class="muted">正在加载故事参数...</p>
    <p v-if="error" class="error">{{ error }}</p>
    <p v-if="notice" class="notice">{{ notice }}</p>

    <form v-if="!loading" class="settings-form panel" @submit.prevent="saveSettings">
      <label class="field">
        <span>标题 *</span>
        <input v-model="form.title" placeholder="故事标题" />
      </label>

      <label class="field">
        <span>描述</span>
        <textarea v-model="form.description" rows="3" placeholder="故事集合里展示的简介" />
      </label>

      <label class="field">
        <span>封面地址</span>
        <input v-model="form.coverUrl" placeholder="https://..." />
      </label>

      <label class="field">
        <span>故事规则 *</span>
        <textarea v-model="form.systemPrompt" rows="5" placeholder="给故事主持人的核心规则" />
      </label>

      <label class="field">
        <span>场景</span>
        <textarea v-model="form.scenario" rows="5" placeholder="用户扮演谁，故事发生在哪里，当前局势是什么" />
      </label>

      <label class="field">
        <span>风格</span>
        <textarea v-model="form.stylePrompt" rows="4" placeholder="叙事视角、语气、节奏、回复习惯" />
      </label>

      <label class="field">
        <span>开场白</span>
        <textarea v-model="form.openingMessage" rows="5" placeholder="新开对话时显示的第一条助手消息" />
      </label>

      <div class="button-row">
        <button type="button" :disabled="saving" @click="router.push(`/stories/${slug}`)">进入故事</button>
        <button class="primary-button" type="submit" :disabled="saving">
          {{ saving ? '保存中...' : '保存参数' }}
        </button>
      </div>
    </form>
  </main>
</template>

<style scoped>
.settings-page { max-width: 880px; }
.header {
  display: flex;
  align-items: end;
  justify-content: space-between;
  gap: 16px;
  margin-bottom: 24px;
}
h1 { margin: 14px 0 8px; font-size: clamp(28px, 5vw, 44px); }
.settings-form {
  display: grid;
  gap: 16px;
  padding: 20px;
}
.field {
  display: grid;
  gap: 8px;
}
.field span {
  color: var(--label);
  font-size: 14px;
}
input,
textarea {
  width: 100%;
  border: 1px solid var(--border);
  border-radius: 14px;
  background: var(--control);
  color: var(--text-strong);
  padding: 12px 14px;
  outline: none;
}
textarea { resize: vertical; line-height: 1.55; }
input:focus,
textarea:focus { border-color: var(--accent-border); }
button {
  border-radius: 999px;
  padding: 10px 14px;
  background: var(--control-hover);
  color: var(--text);
}
button:disabled { cursor: not-allowed; opacity: 0.55; }
.primary-button {
  background: var(--accent);
  color: var(--accent-contrast);
  font-weight: 700;
}
.ghost-button { background: var(--control); }
.button-row {
  display: flex;
  justify-content: flex-end;
  gap: 10px;
}
.error { color: var(--danger-text); margin-bottom: 12px; }
.notice { color: var(--success-text); margin-bottom: 12px; }
@media (max-width: 640px) {
  .header { align-items: stretch; flex-direction: column; }
  .button-row { flex-direction: column; }
}
</style>
