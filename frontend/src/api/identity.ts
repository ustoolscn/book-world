import { apiFetch, setSessionId } from './client'

const BASE_URL_KEY = 'book-world-baseurl'
const API_KEY_KEY = 'book-world-apikey'

export async function enterIdentity(baseUrl: string, apiKey: string) {
  const result = await apiFetch<{ sessionId: string }>('/api/identity/enter', {
    method: 'POST',
    body: JSON.stringify({ baseUrl, apiKey })
  })
  setSessionId(result.sessionId)
  setProviderConfig(baseUrl, apiKey)
  return result
}

export function getProviderConfig() {
  return {
    baseUrl: localStorage.getItem(BASE_URL_KEY) || '',
    apiKey: localStorage.getItem(API_KEY_KEY) || ''
  }
}

export function setProviderConfig(baseUrl: string, apiKey: string) {
  localStorage.setItem(BASE_URL_KEY, baseUrl)
  localStorage.setItem(API_KEY_KEY, apiKey)
}
