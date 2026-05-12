import { getSessionId } from './client'
import type { Message } from './stories'

export interface TokenUsage {
  promptTokens: number
  completionTokens: number
  totalTokens: number
}

export async function streamChat(
  storySlug: string,
  message: string,
  model: string,
  thinkingEffort: string,
  messages: Message[],
  userProfile: string,
  onDelta: (content: string) => void,
  onUsage?: (usage: TokenUsage) => void
) {
  const response = await fetch('/api/chat/stream', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      Authorization: `Bearer ${getSessionId()}`
    },
    body: JSON.stringify({ storySlug, message, model, thinkingEffort, messages, userProfile })
  })
  if (!response.ok || !response.body) throw new Error(`Chat request failed: ${response.status}`)

  const reader = response.body.getReader()
  const decoder = new TextDecoder()
  let buffer = ''

  while (true) {
    const { value, done } = await reader.read()
    if (done) break
    buffer += decoder.decode(value, { stream: true })
    const events = buffer.split('\n\n')
    buffer = events.pop() || ''
    for (const event of events) handleEvent(event, onDelta, onUsage)
  }
  if (buffer) handleEvent(buffer, onDelta, onUsage)
}

function handleEvent(raw: string, onDelta: (content: string) => void, onUsage?: (usage: TokenUsage) => void) {
  const lines = raw.split('\n')
  const event = lines.find((line) => line.startsWith('event:'))?.slice(6).trim()
  const dataLine = lines.find((line) => line.startsWith('data:'))
  if (!event || !dataLine) return
  const data = JSON.parse(dataLine.slice(5).trim())
  if (event === 'delta') onDelta(data.content || '')
  if (event === 'usage') onUsage?.({
    promptTokens: Number(data.promptTokens || 0),
    completionTokens: Number(data.completionTokens || 0),
    totalTokens: Number(data.totalTokens || 0)
  })
  if (event === 'error') throw new Error(data.message || 'Chat stream failed')
}
