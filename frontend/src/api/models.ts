import { apiFetch } from './client'

export async function listModels() {
  const result = await apiFetch<{ models: string[] }>('/api/models')
  return result.models
}
