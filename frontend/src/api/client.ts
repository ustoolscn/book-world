const TOKEN_KEY = 'book-world-session'

export function getSessionId() {
  return sessionStorage.getItem(TOKEN_KEY) || localStorage.getItem(TOKEN_KEY) || ''
}

export function setSessionId(token: string) {
  sessionStorage.setItem(TOKEN_KEY, token)
}

export async function apiFetch<T>(path: string, options: RequestInit = {}): Promise<T> {
  const headers = new Headers(options.headers)
  headers.set('Content-Type', 'application/json')
  const token = getSessionId()
  if (token) headers.set('Authorization', `Bearer ${token}`)

  const response = await fetch(path, { ...options, headers })
  if (!response.ok) {
    let message = `Request failed: ${response.status}`
    try {
      const data = await response.json()
      message = data.message || message
    } catch {}
    throw new Error(message)
  }
  return response.json()
}
