<script setup lang="ts">
import { ref } from 'vue'

const emit = defineEmits<{ send: [message: string]; blocked: [] }>()
const props = defineProps<{ disabled?: boolean; blocked?: boolean }>()
const message = ref('')

function send() {
  if (props.blocked) {
    emit('blocked')
    return
  }
  const value = message.value.trim()
  if (!value) return
  emit('send', value)
  message.value = ''
}

function onKeydown(event: KeyboardEvent) {
  if (props.blocked) {
    event.preventDefault()
    emit('blocked')
    return
  }
  if (event.key === 'Enter' && !event.shiftKey) {
    event.preventDefault()
    send()
  }
}

function handleBlockedInteraction() {
  if (props.blocked) emit('blocked')
}
</script>

<template>
  <div class="chat-input panel" :class="{ blocked }" @click="handleBlockedInteraction">
    <textarea
      v-model="message"
      :readonly="blocked"
      :disabled="disabled && !blocked"
      :placeholder="blocked ? '请先填写身份姓名...' : '输入你的行动或台词...'"
      @focus="handleBlockedInteraction"
      @keydown="onKeydown"
    />
    <button :disabled="(disabled && !blocked) || (!blocked && !message.trim())" @click.stop="send">发送</button>
  </div>
</template>

<style scoped>
.chat-input {
  display: grid;
  grid-template-columns: 1fr auto;
  gap: 8px;
  padding: 8px;
  border-radius: 16px;
}
textarea {
  min-height: 42px;
  max-height: 96px;
  resize: none;
  overflow-y: auto;
  border: 0;
  outline: 0;
  color: var(--text);
  background: transparent;
  line-height: 1.45;
  scrollbar-width: thin;
  scrollbar-color: var(--scrollbar) transparent;
}
textarea::-webkit-scrollbar { width: 8px; }
textarea::-webkit-scrollbar-track { background: transparent; }
textarea::-webkit-scrollbar-thumb {
  border: 2px solid transparent;
  border-radius: 999px;
  background: var(--scrollbar);
  background-clip: content-box;
}
button {
  align-self: end;
  padding: 10px 14px;
  border-radius: 999px;
  color: var(--accent-contrast);
  background: var(--accent);
  font-weight: 700;
}
button:disabled { opacity: 0.5; cursor: not-allowed; }
.chat-input.blocked {
  border-color: var(--accent-border);
}
.chat-input.blocked textarea,
.chat-input.blocked button {
  cursor: pointer;
}
</style>
