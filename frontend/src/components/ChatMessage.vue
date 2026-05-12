<script setup lang="ts">
import { computed } from 'vue'

const props = defineProps<{
  role: string
  content: string
  userName?: string
  assistantName?: string
  pending?: boolean
}>()

type InlinePart = {
  text: string
  strong?: boolean
  emphasis?: boolean
  code?: boolean
}

type TextBlock = {
  type: 'paragraph' | 'heading' | 'quote' | 'code'
  text: string
  parts?: InlinePart[]
}

type ListBlock = {
  type: 'list'
  ordered: boolean
  items: InlinePart[][]
}

type RuleBlock = {
  type: 'rule'
}

type MessageBlock = TextBlock | ListBlock | RuleBlock

const displayRole = computed(() => {
  if (props.role === 'user') return props.userName || '你'
  if (props.role === 'assistant') return props.assistantName || '故事'
  return '系统'
})

const blocks = computed<MessageBlock[]>(() => parseMessage(props.content))

function parseMessage(content: string): MessageBlock[] {
  const lines = normalizeParagraphs(content).split('\n')
  const result: MessageBlock[] = []
  let paragraph: string[] = []
  let code: string[] | null = null
  let list: ListBlock | null = null

  const flushParagraph = () => {
    if (!paragraph.length) return
    const text = paragraph.join('\n').trim()
    result.push({ type: 'paragraph', text, parts: parseInline(text) })
    paragraph = []
  }

  const flushList = () => {
    if (!list) return
    result.push(list)
    list = null
  }

  for (const line of lines) {
    if (line.trim().startsWith('```')) {
      flushParagraph()
      flushList()
      if (code) {
        result.push({ type: 'code', text: code.join('\n') })
        code = null
      } else {
        code = []
      }
      continue
    }

    if (code) {
      code.push(line)
      continue
    }

    const trimmed = line.trim()
    if (!trimmed) {
      flushParagraph()
      flushList()
      continue
    }

    if (/^[-*_]{3,}$/.test(trimmed)) {
      flushParagraph()
      flushList()
      result.push({ type: 'rule' })
      continue
    }

    const headingMatch = trimmed.match(/^#{1,3}\s+(.+)$/)
    if (headingMatch) {
      flushParagraph()
      flushList()
      const text = headingMatch[1].trim()
      result.push({ type: 'heading', text, parts: parseInline(text) })
      continue
    }

    const quoteMatch = trimmed.match(/^>\s?(.+)$/)
    if (quoteMatch) {
      flushParagraph()
      flushList()
      const text = quoteMatch[1].trim()
      result.push({ type: 'quote', text, parts: parseInline(text) })
      continue
    }

    const unorderedMatch = trimmed.match(/^[-*]\s+(.+)$/)
    const orderedMatch = trimmed.match(/^\d+[.)]\s+(.+)$/)
    if (unorderedMatch || orderedMatch) {
      flushParagraph()
      const ordered = Boolean(orderedMatch)
      if (!list || list.ordered !== ordered) {
        flushList()
        list = { type: 'list', ordered, items: [] }
      }
      list.items.push(parseInline((unorderedMatch?.[1] || orderedMatch?.[1] || '').trim()))
      continue
    }

    flushList()
    paragraph.push(line)
  }

  if (code) result.push({ type: 'code', text: code.join('\n') })
  flushParagraph()
  flushList()
  return result.length ? result : [{ type: 'paragraph', text: content, parts: parseInline(content) }]
}

function normalizeParagraphs(content: string) {
  return content
    .replace(/\r\n/g, '\n')
    .replace(/([^\n])\n(?=[「“\"（(《【\[])/g, '$1\n\n')
    .replace(/([。！？!?」”])\n(?=\S)/g, '$1\n\n')
}

function parseInline(text: string): InlinePart[] {
  const parts: InlinePart[] = []
  const pattern = /(\*\*[^*]+\*\*|__[^_]+__|\*[^*]+\*|_[^_]+_|`[^`]+`)/g
  let lastIndex = 0
  let match: RegExpExecArray | null

  while ((match = pattern.exec(text))) {
    if (match.index > lastIndex) parts.push({ text: text.slice(lastIndex, match.index) })

    const token = match[0]
    if (token.startsWith('**') || token.startsWith('__')) {
      parts.push({ text: token.slice(2, -2), strong: true })
    } else if (token.startsWith('`')) {
      parts.push({ text: token.slice(1, -1), code: true })
    } else {
      parts.push({ text: token.slice(1, -1), emphasis: true })
    }
    lastIndex = pattern.lastIndex
  }

  if (lastIndex < text.length) parts.push({ text: text.slice(lastIndex) })
  return parts.length ? parts : [{ text }]
}
</script>

<template>
  <div class="message" :class="role">
    <div class="role">{{ displayRole }}</div>
    <div class="content">
      <div v-if="pending && !content.trim()" class="pending-indicator" aria-live="polite">
        <span class="pending-dot"></span>
        <span class="pending-dot"></span>
        <span class="pending-dot"></span>
        <strong>正在生成回复</strong>
      </div>
      <template v-for="(block, index) in blocks" :key="index">
        <h3 v-if="block.type === 'heading'">
          <template v-for="(part, partIndex) in block.parts" :key="partIndex">
            <strong v-if="part.strong">{{ part.text }}</strong>
            <em v-else-if="part.emphasis">{{ part.text }}</em>
            <code v-else-if="part.code">{{ part.text }}</code>
            <span v-else>{{ part.text }}</span>
          </template>
        </h3>
        <blockquote v-else-if="block.type === 'quote'">
          <template v-for="(part, partIndex) in block.parts" :key="partIndex">
            <strong v-if="part.strong">{{ part.text }}</strong>
            <em v-else-if="part.emphasis">{{ part.text }}</em>
            <code v-else-if="part.code">{{ part.text }}</code>
            <span v-else>{{ part.text }}</span>
          </template>
        </blockquote>
        <pre v-else-if="block.type === 'code'"><code>{{ block.text }}</code></pre>
        <hr v-else-if="block.type === 'rule'" />
        <ol v-else-if="block.type === 'list' && block.ordered">
          <li v-for="(item, itemIndex) in block.items" :key="itemIndex">
            <template v-for="(part, partIndex) in item" :key="partIndex">
              <strong v-if="part.strong">{{ part.text }}</strong>
              <em v-else-if="part.emphasis">{{ part.text }}</em>
              <code v-else-if="part.code">{{ part.text }}</code>
              <span v-else>{{ part.text }}</span>
            </template>
          </li>
        </ol>
        <ul v-else-if="block.type === 'list'">
          <li v-for="(item, itemIndex) in block.items" :key="itemIndex">
            <template v-for="(part, partIndex) in item" :key="partIndex">
              <strong v-if="part.strong">{{ part.text }}</strong>
              <em v-else-if="part.emphasis">{{ part.text }}</em>
              <code v-else-if="part.code">{{ part.text }}</code>
              <span v-else>{{ part.text }}</span>
            </template>
          </li>
        </ul>
        <p v-else>
          <template v-for="(part, partIndex) in block.parts" :key="partIndex">
            <strong v-if="part.strong">{{ part.text }}</strong>
            <em v-else-if="part.emphasis">{{ part.text }}</em>
            <code v-else-if="part.code">{{ part.text }}</code>
            <span v-else>{{ part.text }}</span>
          </template>
        </p>
      </template>
    </div>
  </div>
</template>

<style scoped>
.message {
  display: grid;
  gap: 6px;
  width: fit-content;
  max-width: min(760px, 94%);
  padding: 12px 14px;
  border-radius: 15px;
  background: var(--control);
  line-height: 1.82;
}
.message.assistant,
.message.system {
  width: min(760px, 100%);
}
.message.user {
  margin-left: auto;
  background: var(--accent-soft);
}
.role {
  color: var(--accent-strong);
  font-size: 12px;
  font-weight: 700;
}
.content {
  display: grid;
  gap: 12px;
  color: var(--text);
  font-size: 15px;
  letter-spacing: 0.01em;
  overflow-wrap: anywhere;
}
.content :where(p, h3, blockquote, pre, ul, ol) {
  margin: 0;
}
.content p {
  white-space: pre-wrap;
}
.content strong {
  color: var(--accent-strong);
  font-weight: 800;
}
.content em {
  color: var(--muted-strong);
  font-style: italic;
}
.content h3 {
  padding-bottom: 4px;
  border-bottom: 1px solid var(--accent-soft);
  color: var(--accent-strong);
  font-size: 16px;
  line-height: 1.5;
}
.content blockquote {
  padding: 8px 11px;
  border-left: 3px solid var(--accent-border);
  border-radius: 9px;
  color: var(--muted-strong);
  background: var(--control);
  white-space: pre-wrap;
}
.content ul,
.content ol {
  display: grid;
  gap: 7px;
  padding-left: 22px;
}
.content li {
  padding-left: 3px;
}
.content :not(pre) > code {
  padding: 1px 5px;
  border-radius: 6px;
  color: var(--accent-strong);
  background: color-mix(in srgb, var(--bg-deep) 45%, transparent);
}
.content pre {
  max-width: 100%;
  overflow-x: auto;
  padding: 10px 11px;
  border-radius: 10px;
  color: var(--text);
  background: color-mix(in srgb, var(--bg-deep) 54%, transparent);
  line-height: 1.6;
  scrollbar-width: thin;
  scrollbar-color: var(--scrollbar) transparent;
}
.content pre::-webkit-scrollbar { height: 8px; }
.content pre::-webkit-scrollbar-track { background: transparent; }
.content pre::-webkit-scrollbar-thumb {
  border: 2px solid transparent;
  border-radius: 999px;
  background: var(--scrollbar);
  background-clip: content-box;
}
.content code {
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
  font-size: 13px;
}
.content hr {
  width: 100%;
  margin: 2px 0;
  border: 0;
  border-top: 1px solid var(--border-strong);
}
.pending-indicator {
  display: inline-flex;
  align-items: center;
  gap: 7px;
  color: var(--muted-strong);
}
.pending-indicator strong {
  color: var(--muted-strong);
  font-size: 14px;
  font-weight: 700;
}
.pending-dot {
  width: 7px;
  height: 7px;
  border-radius: 999px;
  background: var(--accent);
  animation: pendingPulse 1.1s ease-in-out infinite;
}
.pending-dot:nth-child(2) {
  animation-delay: 0.16s;
}
.pending-dot:nth-child(3) {
  animation-delay: 0.32s;
}
@keyframes pendingPulse {
  0%,
  80%,
  100% {
    opacity: 0.35;
    transform: translateY(0);
  }
  40% {
    opacity: 1;
    transform: translateY(-3px);
  }
}
@media (max-width: 520px) {
  .message {
    max-width: 96%;
    padding: 11px 12px;
  }
  .message.assistant,
  .message.system {
    width: 100%;
  }
  .content {
    gap: 11px;
    font-size: 14px;
    line-height: 1.78;
  }
}
</style>
