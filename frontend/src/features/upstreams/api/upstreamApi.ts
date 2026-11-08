import { apiClient } from '@/shared/api/apiClient'

export const upstreamSectionNames = [
  'gemini-api-key',
  'codex-api-key',
  'xai-api-key',
  'claude-api-key',
  'vertex-api-key',
  'openai-compatibility',
] as const

export type UpstreamSection = (typeof upstreamSectionNames)[number]
export type UpstreamItem = Record<string, unknown>

export function isUpstreamDisabled(section: UpstreamSection, item: UpstreamItem): boolean {
  return section === 'openai-compatibility'
    ? item.disabled === true
    : Array.isArray(item['excluded-models']) && item['excluded-models'].includes('*')
}

export function withUpstreamEnabled(section: UpstreamSection, item: UpstreamItem, enabled: boolean): UpstreamItem {
  const next = { ...item }
  if (section === 'openai-compatibility') next.disabled = !enabled
  else {
    const excluded = Array.isArray(item['excluded-models']) ? item['excluded-models'].filter(model => typeof model === 'string' && model !== '*') : []
    if (!enabled) excluded.unshift('*')
    if (excluded.length) next['excluded-models'] = excluded
    else delete next['excluded-models']
  }
  return next
}

export interface DiscoveredUpstreamModel {
  name: string
  alias?: string
  display_name?: string
}

export interface DiscoverUpstreamModelsRequest {
  section: UpstreamSection
  base_url: string
  api_key?: string
  auth_index?: string
  headers?: Record<string, string>
}

interface UpstreamSectionsResponse {
  sections: Partial<Record<UpstreamSection, UpstreamItem[]>>
  api_version?: number
  revision?: string
}

export async function listUpstreamSections() {
  const response = await apiClient.get<UpstreamSectionsResponse>('/upstreams')
  const sections = Object.fromEntries(
    upstreamSectionNames.map((name) => [name, Array.isArray(response.sections[name]) ? response.sections[name] : []]),
  ) as Record<UpstreamSection, UpstreamItem[]>
  return { sections, apiVersion: response.api_version ?? 0, revision: response.revision ?? '' }
}

export function replaceUpstreamSection(section: UpstreamSection, items: UpstreamItem[], apiVersion = 0, revision = '') {
  return apiClient.put<void>(`/upstreams/${section}`, apiVersion === 8 ? { items, api_version: apiVersion, revision } : items)
}

export async function discoverUpstreamModels(payload: DiscoverUpstreamModelsRequest): Promise<DiscoveredUpstreamModel[]> {
  const response = await apiClient.post<{ models: DiscoveredUpstreamModel[] }>('/upstreams/models', payload)
  return Array.isArray(response.models) ? response.models : []
}
