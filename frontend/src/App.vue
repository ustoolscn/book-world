<script setup lang="ts">
import { onMounted, onUnmounted, ref, watch } from 'vue'

type ThemeMode = 'system' | 'dark' | 'light'

const THEME_KEY = 'book-world-theme'
const themeMode = ref<ThemeMode>(readThemeMode())
const mediaQuery = window.matchMedia('(prefers-color-scheme: dark)')

function readThemeMode(): ThemeMode {
  const saved = localStorage.getItem(THEME_KEY)
  return saved === 'dark' || saved === 'light' || saved === 'system' ? saved : 'system'
}

function resolvedTheme() {
  return themeMode.value === 'system' ? (mediaQuery.matches ? 'dark' : 'light') : themeMode.value
}

function applyTheme() {
  document.documentElement.dataset.theme = resolvedTheme()
  document.documentElement.dataset.themeMode = themeMode.value
  document.documentElement.style.colorScheme = resolvedTheme()
}

function handleSystemThemeChanged() {
  if (themeMode.value === 'system') applyTheme()
}

watch(themeMode, (value) => {
  localStorage.setItem(THEME_KEY, value)
  applyTheme()
})

onMounted(() => {
  applyTheme()
  mediaQuery.addEventListener('change', handleSystemThemeChanged)
})

onUnmounted(() => {
  mediaQuery.removeEventListener('change', handleSystemThemeChanged)
})
</script>

<template>
  <router-view />
  <label class="theme-switcher" aria-label="主题">
    <span>主题</span>
    <select v-model="themeMode">
      <option value="system">跟随系统</option>
      <option value="dark">暗色</option>
      <option value="light">亮色</option>
    </select>
  </label>
</template>

<style scoped>
.theme-switcher {
  position: fixed;
  right: 16px;
  bottom: 86px;
  z-index: 80;
  display: grid;
  grid-template-columns: auto auto;
  align-items: center;
  gap: 8px;
  padding: 8px 10px;
  border: 1px solid var(--border);
  border-radius: 999px;
  background: color-mix(in srgb, var(--panel) 92%, transparent);
  color: var(--muted);
  box-shadow: var(--shadow-md);
  backdrop-filter: blur(14px);
  font-size: 12px;
}

.theme-switcher select {
  width: auto;
  min-width: 94px;
  border: 0;
  border-radius: 999px;
  padding: 6px 8px;
  background: var(--control);
  color: var(--text);
  outline: none;
}

@media (max-width: 640px) {
  .theme-switcher {
    right: 10px;
    bottom: 78px;
  }
}
</style>
