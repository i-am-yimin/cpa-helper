<script setup lang="ts">
import {
  computed,
  defineAsyncComponent,
  onBeforeUnmount,
  onMounted,
  reactive,
  ref,
  watch,
} from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { toast } from 'vue-sonner'
import {
  ArrowDownWideNarrow,
  ChevronDown,
  ChevronRight,
  KeyRound,
  Plus,
  RefreshCw,
  Search,
  ShieldCheck,
  SlidersHorizontal,
} from '@lucide/vue'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Switch } from '@/components/ui/switch'
import { Badge } from '@/components/ui/badge'
import { Spinner } from '@/components/ui/spinner'
import { Tabs, TabsList, TabsTrigger } from '@/components/ui/tabs'
import {
  Table,
  TableBody,
  TableCell,
  TableEmpty,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import AntigravityQuotaPanel from '@/features/codex-keeper/components/AntigravityQuotaPanel.vue'
import CodexQuotaPolicyEditor from '@/features/codex-keeper/components/CodexQuotaPolicyEditor.vue'
import {
  disableCodexKeeperAccount,
  enableCodexKeeperAccount,
  getAntigravityQuota,
  getGroupSchedulerStatus,
  listCodexKeeperAccounts,
  updateAntigravityGroup,
  updateCodexKeeperPriority,
  resumeCodexQuotaAccount,
} from '@/features/codex-keeper/api/codexKeeperApi'
import {
  listUpstreamSections,
  isUpstreamDisabled,
  withUpstreamEnabled,
  replaceUpstreamSection,
  upstreamSectionNames,
  type UpstreamItem,
  type UpstreamSection,
} from '../api/upstreamApi'
import type {
  AntigravityQuotaSnapshot,
  CodexKeeperAccount,
  CodexKeeperPriorityRule,
} from '@/shared/types/api'
import { useI18n } from '@/shared/i18n'
import { formatCompact, formatDateTime, formatUsd } from '@/shared/utils/format'

const UpstreamEditor = defineAsyncComponent(
  () => import('./UpstreamManagementView.vue'),
)
const Credentials = defineAsyncComponent(
  () => import('@/features/codex-keeper/views/CodexKeeperStatusView.vue'),
)
const InspectionSettings = defineAsyncComponent(
  () =>
    import('@/features/codex-keeper/components/CodexKeeperSettingsPanel.vue'),
)
const { t, errorText } = useI18n()
const route = useRoute(),
  router = useRouter()
const tab = computed(() =>
  ['configuration', 'credentials', 'rules'].includes(String(route.query.tab))
    ? String(route.query.tab)
    : 'all',
)
function changeTab(value: unknown) {
  if (hasDrafts.value) { toast.info(t('请先保存或放弃当前更改', 'Save or discard the current changes first')); return }
  void router.replace({ query: { ...route.query, tab: String(value) } })
}
const loading = ref(false),
  loadErrors = ref<string[]>([]),
  search = ref(''),
  provider = ref('all'),
  source = ref('all'),
  descending = ref(true)
const accounts = ref<CodexKeeperAccount[]>([]),
  rules = ref<CodexKeeperPriorityRule[]>([])
const sections = ref<Partial<Record<UpstreamSection, UpstreamItem[]>>>({}),
  apiVersion = ref(0),
  revision = ref('')
const quotas = ref<Record<string, AntigravityQuotaSnapshot>>({}),
  quotaLoading = reactive(new Set<string>())
const expanded = reactive(new Set<string>()),
  busy = ref(''),
  schedulerPhase = ref('unknown'),
  schedulerLastPick = ref('')
const schedulerReady = computed(() => schedulerPhase.value === 'ready')
const schedulerLabel = computed(() => ({
  ready: t('分组插件已加载', 'Group plugin loaded'),
  missing: t('未找到分组插件', 'Group plugin not found'),
  disabled: t('分组插件未启用', 'Group plugin disabled'),
  unknown: t('插件状态读取失败', 'Plugin status unavailable'),
})[schedulerPhase.value] || t('插件状态未知', 'Plugin status unknown'))
const policyDraftKey = ref(''), policyReset = ref(0)
const drafts = reactive<Record<string, string>>({}),
  modelDrafts = reactive<Record<string, string>>({})
const activeDraftKey = computed(() => Object.keys(drafts)[0] || Object.keys(modelDrafts)[0] || policyDraftKey.value)
const hasDrafts = computed(() => !!activeDraftKey.value)
function canEdit(key: string) { return !busy.value && (!hasDrafts.value || activeDraftKey.value === key) }
function discardDrafts() { for (const key of Object.keys(drafts)) delete drafts[key]; for (const key of Object.keys(modelDrafts)) delete modelDrafts[key]; policyDraftKey.value = ''; policyReset.value++; void load() }
function policyDirty(key: string, value: boolean) { policyDraftKey.value = value ? key : '' }
function upstreamIdentity(item: UpstreamItem, index: number) {
  const authIndex = item['auth-index'] || item.auth_index
  if (authIndex) return `#${String(authIndex)}`
  const key = item['api-key']
  return typeof key === 'string' && key.length > 8 ? `····${key.slice(-4)}` : `API #${index + 1}`
}
let disposed = false
let refreshTimer: ReturnType<typeof setInterval> | undefined
type Row = {
  id: string
  name: string
  provider: string
  kind: 'oauth' | 'api'
  priority: number
  disabled: boolean
  account?: CodexKeeperAccount
  item?: UpstreamItem
  section?: UpstreamSection
  index?: number
}
const providerLabels: Record<string, string> = {
  codex: 'Codex',
  antigravity: 'Antigravity',
  gemini: 'Gemini',
  claude: 'Claude',
  xai: 'xAI',
  vertex: 'Vertex',
  'openai-compatibility': 'OpenAI Compatible',
}
const allRows = computed<Row[]>(() => [
  ...accounts.value.map((account) => ({
    id: `oauth:${account.name}`,
    name: account.email || account.label || account.name,
    provider: account.provider,
    kind: 'oauth' as const,
    priority: account.priority ?? 0,
    disabled: account.disabled,
    account,
  })),
  ...upstreamSectionNames.flatMap((section) =>
    (sections.value[section] || []).map((item, index) => ({
      id: `${section}:${index}`,
      name: String(
        item.name ||
          item['base-url'] ||
          `${providerLabels[section.replace('-api-key', '')] || section} ${index + 1}`,
      ),
      provider: section.replace('-api-key', ''),
      kind: 'api' as const,
      priority: Number(item.priority ?? 0),
      disabled: isUpstreamDisabled(section, item),
      item,
      section,
      index,
    })),
  ),
])
const providers = computed(() => [
  ...new Set(allRows.value.map((row) => row.provider)),
])
const rows = computed(() =>
  allRows.value
    .filter(
      (row) =>
        (provider.value === 'all' || row.provider === provider.value) &&
        (source.value === 'all' || row.kind === source.value) &&
        `${row.name} ${row.provider} ${row.account?.name ?? ''}`
          .toLowerCase()
          .includes(search.value.trim().toLowerCase()),
    )
    .sort(
      (a, b) =>
        (descending.value
          ? b.priority - a.priority
          : a.priority - b.priority) || a.name.localeCompare(b.name),
    ),
)
const enabledCount = computed(
  () => allRows.value.filter((row) => !row.disabled).length,
)
const agGroupCount = computed(() =>
  Object.values(quotas.value).reduce((n, quota) => n + quota.groups.length, 0),
)

async function load() {
  if (loading.value || disposed || hasDrafts.value) return
  loading.value = true
  loadErrors.value = []
  const results = await Promise.allSettled([
    listCodexKeeperAccounts(),
    listUpstreamSections(),
    getGroupSchedulerStatus(),
  ])
  if (disposed || hasDrafts.value) { loading.value = false; return }
  const [credentials, upstreams, scheduler] = results
  if (credentials.status === 'fulfilled') {
    accounts.value = credentials.value.items
    rules.value = credentials.value.priority_rules
    if (credentials.value.degraded)
      loadErrors.value.push(
        t(
          '凭证列表正在使用本地缓存。',
          'Credentials are using the local cache.',
        ),
      )
  } else
    loadErrors.value.push(
      errorText(
        credentials.reason,
        '凭证加载失败，请重试',
        'Failed to load credentials. Retry.',
      ),
    )
  if (upstreams.status === 'fulfilled') {
    sections.value = upstreams.value.sections
    apiVersion.value = upstreams.value.apiVersion
    revision.value = upstreams.value.revision
  } else
    loadErrors.value.push(
      errorText(
        upstreams.reason,
        'API Key 上游加载失败，请重试',
        'Failed to load API Key upstreams. Retry.',
      ),
    )
  schedulerPhase.value = scheduler.status === 'fulfilled' ? scheduler.value.phase || (scheduler.value.loaded ? 'ready' : 'unknown') : 'unknown'
  schedulerLastPick.value = scheduler.status === 'fulfilled' ? scheduler.value.last_pick_at : ''
  loading.value = false
  // Bound upstream quota requests; never expose tokens to the browser.
  const pending = accounts.value.filter(
    (account) => account.provider === 'antigravity' && !account.disabled,
  )
  for (let i = 0; i < pending.length && !disposed; i += 4)
    await Promise.all(
      pending.slice(i, i + 4).map((account) => loadQuota(account)),
    )
}
async function loadQuota(account: CodexKeeperAccount, force = false) {
  if (quotaLoading.has(account.name) || account.disabled) return
  quotaLoading.add(account.name)
  try {
    const quota = await getAntigravityQuota(account.name, force)
    if (!disposed) quotas.value[account.name] = quota
  } catch (error) {
    toast.error(
      errorText(error, '额度查询失败，请重试', 'Quota query failed. Retry.'),
    )
  } finally {
    quotaLoading.delete(account.name)
  }
}
function priorityHint(row: Row) {
  if (row.provider === 'antigravity')
    return t(
      `各模型组独立调度；未匹配模型回退优先级 ${row.priority}`,
      `Groups schedule independently; unmatched models use priority ${row.priority}`,
    )
  if (row.provider !== 'codex')
    return t('数值越大越优先', 'Higher numbers run first')
  const policy = row.account?.quota_policy
  if (policy?.paused) return t('额度触发暂停 · 额度恢复后自动恢复原优先级', 'Paused by quota · Previous priority is restored when quota recovers')
  if (policy?.manual_priority != null) return t(`手动优先级 ${policy.manual_priority}`, `Manual priority ${policy.manual_priority}`)
  const plan = row.account?.plan_type || row.account?.account_type
  const rule = rules.value.find((rule) => rule.account_type === plan)
  return rule
    ? t(
        `套餐默认 ${rule.priority} · 手动设置后固定`,
        `Plan default ${rule.priority} · Manual values are retained`,
      )
    : t('手动设置后固定 · -1 为额度暂停', 'Manual values are retained · -1 means quota pause')
}
function quotaSummary(row: Row) {
  if (row.account?.quota_policy?.paused) return { label: row.account.quota_policy.sync_state === 'paused' ? t('额度触发暂停', 'Paused by quota') : t('暂停待同步', 'Pause pending sync'), remaining: row.account.primary_used_percent == null ? null : Math.max(0, Math.min(100, 100 - row.account.primary_used_percent)) }
  if (row.disabled) return { label: t('已禁用', 'Disabled'), remaining: null }
  if (row.provider === 'antigravity') {
    const quota = quotas.value[row.account!.name]
    const buckets =
      quota?.groups.flatMap((group) =>
        group.buckets.map((bucket) => ({ group, bucket })),
      ) || []
    buckets.sort(
      (a, b) => a.bucket.remaining_percent - b.bucket.remaining_percent,
    )
    const tightest = buckets[0]
    return {
      label: tightest
        ? `${tightest.group.label} · ${tightest.bucket.window}${quota?.stale ? t(' · 已过期', ' · Stale') : ''}`
        : t('等待额度', 'Awaiting quota'),
      remaining: tightest?.bucket.remaining_percent ?? null,
    }
  }
  const used = row.account?.primary_used_percent
  return {
    label:
      row.provider === 'codex'
        ? t('主窗口剩余', 'Primary window remaining')
        : t('上游未提供额度', 'No quota reported'),
    remaining: used == null ? null : Math.max(0, 100 - used),
  }
}
async function mutate(row: Row, action: 'priority' | 'toggle') {
  if (busy.value) return
  busy.value = row.id
  try {
    const rawPriority = String(drafts[row.id] ?? row.priority).trim()
    const priority = rawPriority ? Number(rawPriority) : Number.NaN
    if (
      action === 'priority' &&
      (!Number.isInteger(priority) || priority < -1000000 || priority > 1000000)
    )
      throw new Error(
        t('请输入有效整数优先级', 'Enter a valid integer priority'),
      )
    if (row.kind === 'oauth') {
      if (action === 'priority')
        await (row.account?.quota_policy?.paused ? resumeCodexQuotaAccount : updateCodexKeeperPriority)(row.account!.name, priority)
      else
        await (
          row.disabled ? enableCodexKeeperAccount : disableCodexKeeperAccount
        )(row.account!.name)
    } else {
      const items = (sections.value[row.section!] || []).map((item, i) =>
        i === row.index
          ? action === 'priority' ? { ...item, priority } : withUpstreamEnabled(row.section!, item, row.disabled)
          : item,
      )
      await replaceUpstreamSection(
        row.section!,
        items,
        apiVersion.value,
        revision.value,
      )
    }
    delete drafts[row.id]
    toast.success(t('已保存', 'Saved'))
    await load()
  } catch (error) {
    toast.error(
      error instanceof Error
        ? error.message
        : errorText(error, '保存失败，请重试', 'Save failed. Retry.'),
    )
  } finally {
    busy.value = ''
  }
}
function groupKey(name: string, groupId: string) {
  return JSON.stringify([name, groupId])
}
async function saveGroup(
  account: CodexKeeperAccount,
  group: AntigravityQuotaSnapshot['groups'][number],
) {
  const key = groupKey(account.name, group.id)
  const priority = Number(drafts[key] ?? group.priority)
  const patterns = (modelDrafts[key] ?? group.model_patterns.join(', '))
    .split(/[,\n]/)
    .map((s) => s.trim())
    .filter(Boolean)
  if (!Number.isInteger(priority) || priority === -1 || !patterns.length) {
    toast.error(
      t(
        '请输入整数优先级和模型映射；-1 留给自动降级',
        'Enter integer priority and model patterns; -1 is reserved for quota fallback',
      ),
    )
    return
  }
  busy.value = key
  try {
    quotas.value[account.name] = await updateAntigravityGroup({
      auth_name: account.name,
      group_id: group.id,
      priority,
      model_patterns: patterns,
    })
    delete drafts[key]
    delete modelDrafts[key]
    toast.success(t('分组规则已保存', 'Group policy saved'))
  } catch (error) {
    toast.error(
      errorText(error, '分组规则保存失败', 'Failed to save group policy'),
    )
  } finally {
    busy.value = ''
  }
}
function toggleDetails(id: string) {
  if (hasDrafts.value) { toast.info(t('请先保存或放弃当前更改', 'Save or discard the current changes first')); return }
  if (expanded.has(id)) expanded.delete(id)
  else expanded.add(id)
}
onMounted(() => {
  void load()
  refreshTimer = setInterval(() => {
    if (tab.value === 'all' && !busy.value) void load()
  }, 60000)
})
onBeforeUnmount(() => {
  disposed = true
  clearInterval(refreshTimer)
})
watch(tab, (value, previous) => { if (value === 'all' && previous !== 'all') void load() })
</script>

<template>
  <section class="page upstream-hub">
    <div class="page-toolbar hub-heading">
      <div>
        <h1 data-page-title class="page-title">
          {{ t('上游与凭证', 'Upstreams & credentials') }}
        </h1>
        <p class="hub-description">
          {{
            t(
              '在一处管理接入、额度与调度顺序。',
              'Manage connections, quotas and routing order in one place.',
            )
          }}
        </p>
      </div>
      <div class="hub-actions">
        <Button variant="outline" :disabled="loading || hasDrafts" @click="load">
          <RefreshCw :class="{ 'animate-spin': loading }" />{{
            t('刷新总览', 'Refresh overview')
          }}
        </Button><Button variant="outline" @click="changeTab('credentials')">
          <KeyRound />{{ t('OAuth / 导入', 'OAuth / import') }}
        </Button><Button @click="changeTab('configuration')">
          <Plus />{{ t('配置 API 上游', 'Configure API upstreams') }}
        </Button>
      </div>
    </div>
    <div v-if="hasDrafts" class="hub-draft-notice" role="status"><span>{{ t('当前有未保存更改，自动刷新已暂停。', 'Unsaved changes; automatic refresh is paused.') }}</span><Button variant="outline" size="sm" :disabled="!!busy" @click="discardDrafts">{{ t('放弃更改', 'Discard changes') }}</Button></div>
    <Tabs :model-value="tab" @update:model-value="changeTab">
      <TabsList class="hub-tabs">
        <TabsTrigger value="all">
          {{
            t('全部上游', 'All upstreams')
          }}
        </TabsTrigger><TabsTrigger value="configuration">
          {{
            t('API 接入配置', 'API connections')
          }}
        </TabsTrigger><TabsTrigger value="credentials">
          {{
            t('OAuth 凭证', 'OAuth credentials')
          }}
        </TabsTrigger><TabsTrigger value="rules">
          <SlidersHorizontal class="size-4" />{{
            t('巡检与默认规则', 'Inspection & defaults')
          }}
        </TabsTrigger>
      </TabsList>
    </Tabs>
    <template v-if="tab === 'all'">
      <div class="hub-overview" aria-live="polite">
        <span><strong>{{ allRows.length }}</strong>{{ t('个上游', 'upstreams') }}</span><span><strong>{{ enabledCount }}</strong>{{ t('个已启用', 'enabled') }}</span><span><strong>{{ agGroupCount }}</strong>{{ t('个 Antigravity 分组', 'Antigravity groups') }}</span><span class="scheduler-state"><ShieldCheck class="size-4" />{{
          schedulerLabel
        }}</span>
      </div>
      <div v-if="loadErrors.length" class="hub-error" role="alert">
        <p v-for="error in loadErrors" :key="error">{{ error }}</p>
        <Button variant="outline" size="sm" @click="load">
          {{
            t('重试', 'Retry')
          }}
        </Button>
      </div>
      <div class="hub-list paper-panel">
        <div class="hub-filters">
          <div class="hub-search">
            <Search class="size-4" /><Input
              v-model="search"
              :aria-label="t('搜索上游', 'Search upstreams')"
              :placeholder="
                t(
                  '搜索账号、服务商或地址…',
                  'Search accounts, providers or URLs…',
                )
              "
            />
          </div>
          <select v-model="provider" :aria-label="t('服务商', 'Provider')">
            <option value="all">{{ t('全部服务商', 'All providers') }}</option>
            <option v-for="p in providers" :key="p" :value="p">
              {{ providerLabels[p] || p }}
            </option>
          </select><select
            v-model="source"
            :aria-label="t('接入方式', 'Connection type')"
          >
            <option value="all">
              {{ t('全部接入方式', 'All connections') }}
            </option>
            <option value="oauth">OAuth</option>
            <option value="api">API Key</option>
          </select><Button variant="ghost" @click="descending = !descending">
            <ArrowDownWideNarrow class="size-4" />{{
              descending
                ? t('基础优先级从高到低', 'Base priority: high to low')
                : t('基础优先级从低到高', 'Base priority: low to high')
            }}
          </Button>
        </div>
        <Table class="hub-table">
          <TableHeader>
            <TableRow>
              <TableHead class="w-10">
                <span class="sr-only">{{
                  t('展开', 'Expand')
                }}</span>
              </TableHead><TableHead>
                {{
                  t('上游 / 凭证', 'Upstream / credential')
                }}
              </TableHead><TableHead>{{ t('服务商', 'Provider') }}</TableHead><TableHead>{{ t('额度概览', 'Quota overview') }}</TableHead><TableHead>{{ t('优先级', 'Priority') }}</TableHead><TableHead>{{ t('启用', 'Enabled') }}</TableHead>
            </TableRow>
          </TableHeader><TableBody>
            <TableEmpty v-if="loading && !allRows.length" :colspan="6">
              <Spinner />{{
                t('正在加载上游…', 'Loading upstreams…')
              }}
            </TableEmpty>
            <TableEmpty v-else-if="!rows.length" :colspan="6">
              {{
                search || provider !== 'all' || source !== 'all'
                  ? t(
                    '没有匹配的上游，请调整筛选。',
                    'No matching upstreams. Adjust the filters.',
                  )
                  : t(
                    '尚无上游，添加 API 接入或导入 OAuth 凭证开始使用。',
                    'No upstreams yet. Add an API connection or import an OAuth credential.',
                  )
              }}
            </TableEmpty>
            <template v-for="row in rows" :key="row.id">
              <TableRow :data-upstream-row="row.id">
                <TableCell>
                  <Button
                    variant="ghost"
                    size="icon-sm"
                    :aria-expanded="expanded.has(row.id)"
                    :aria-label="
                      t(`查看 ${row.name} 详情`, `View ${row.name} details`)
                    "
                    @click="toggleDetails(row.id)"
                  >
                    <ChevronDown v-if="expanded.has(row.id)" /><ChevronRight
                      v-else
                    />
                  </Button>
                </TableCell><TableCell>
                  <strong class="hub-row-name" :title="row.name">{{ row.name }}</strong><span class="hub-row-sub">{{
                    row.account?.name || `${upstreamIdentity(row.item!, row.index!)} · ${row.item?.['base-url'] || 'API Key'}`
                  }}</span>
                </TableCell><TableCell>
                  <span class="hub-provider">{{
                    providerLabels[row.provider] || row.provider
                  }}</span><span class="hub-row-sub">{{
                    row.kind === 'oauth' ? 'OAuth' : 'API Key'
                  }}</span>
                </TableCell><TableCell>
                  <div class="hub-quota">
                    <strong v-if="quotaSummary(row).remaining !== null">{{ quotaSummary(row).remaining }}%</strong><span v-else>—</span><span class="hub-row-sub">{{
                      quotaSummary(row).label
                    }}</span>
                    <div
                      v-if="quotaSummary(row).remaining !== null"
                      class="hub-quota-track"
                    >
                      <i
                        :class="{
                          'is-low': (quotaSummary(row).remaining ?? 100) < 30,
                        }"
                        :style="{ width: `${quotaSummary(row).remaining}%` }"
                      />
                    </div>
                  </div>
                </TableCell><TableCell>
                  <div v-if="row.provider === 'antigravity'" class="group-priority-summary">
                    <button v-for="group in quotas[row.account!.name]?.groups || []" :key="group.id" type="button" @click="toggleDetails(row.id)">
                      <span>{{ group.label }}</span><strong :class="{ 'text-destructive': group.degraded }">{{ group.effective_priority ?? row.priority }}</strong>
                    </button>
                  </div>
                  <div v-else class="hub-priority">
                    <span class="hub-mobile-label">{{ t('优先级', 'Priority') }}</span>
                    <Input
                      type="number"
                      :aria-label="
                        t(`${row.name} 优先级`, `${row.name} priority`)
                      "
                      :model-value="drafts[row.id] ?? row.priority"
                      :disabled="!canEdit(row.id)"
                      @update:model-value="drafts[row.id] = String($event)"
                      @keydown.enter="mutate(row, 'priority')"
                    /><Button
                      v-if="
                        drafts[row.id] !== undefined &&
                          Number(drafts[row.id]) !== row.priority
                      "
                      variant="outline"
                      size="sm"
                      :disabled="!!busy"
                      @click="mutate(row, 'priority')"
                    >
                      {{ row.account?.quota_policy?.paused ? t('保存并恢复路由', 'Save & resume routing') : t('保存', 'Save') }}
                    </Button>
                  </div>
                  <span class="hub-row-sub priority-explanation">{{
                    priorityHint(row)
                  }}</span><Button
                    v-if="row.provider === 'antigravity' || row.provider === 'codex'"
                    variant="link"
                    size="sm"
                    @click="toggleDetails(row.id)"
                  >
                    {{ row.provider === 'codex' ? t('设置额度暂停规则', 'Set quota pause rules') : t('设置分组优先级', 'Set group priorities') }}
                  </Button>
                </TableCell><TableCell>
                  <Switch
                    :model-value="!row.disabled"
                    :disabled="!!busy || hasDrafts"
                    :aria-label="t(`启用 ${row.name}`, `Enable ${row.name}`)"
                    @update:model-value="mutate(row, 'toggle')"
                  />
                </TableCell>
              </TableRow>
              <TableRow v-if="expanded.has(row.id)" class="hub-expanded">
                <TableCell :colspan="6">
                  <div class="hub-detail">
                    <template
                      v-if="row.provider === 'antigravity' && row.account"
                    >
                      <div class="group-rules-header">
                        <div>
                          <h2>
                            {{ t('模型分组调度', 'Model group scheduling') }}
                          </h2>
                          <p>
                            {{
                              t(
                                '只在命中模型组时应用；额度耗尽仅降低当前分组，不影响其他分组。',
                                'Applies to matching models. Exhaustion lowers this group only.',
                              )
                            }}
                          </p>
                        </div>
                        <Badge variant="outline">
                          {{
                            schedulerLabel
                          }}
                        </Badge>
                      </div>
                      <div
                        v-for="group in quotas[row.account.name]?.groups || []"
                        :key="group.id"
                        class="group-policy"
                      >
                        <div>
                          <strong>{{ group.label }}</strong>
                          <p>
                            {{ schedulerReady ? t('当前生效', 'Effective') : t('已配置优先级', 'Configured priority') }}
                            <b>{{ group.effective_priority ?? 0 }}</b><span
                              v-if="group.degraded"
                              class="text-destructive"
                            >
                              ·
                              {{
                                t(
                                  '额度耗尽，自动降级',
                                  'Quota exhausted; lowered',
                                )
                              }}</span>
                          </p>
                        </div>
                        <label>{{ t('分组优先级', 'Group priority')
                        }}<Input
                          type="number"
                          :aria-label="`${group.label} ${t('分组优先级', 'priority')}`"
                          :disabled="!canEdit(groupKey(row.account.name, group.id))"
                          :model-value="
                            drafts[groupKey(row.account.name, group.id)] ??
                              group.priority ??
                              0
                          "
                          @update:model-value="
                            drafts[groupKey(row.account!.name, group.id)] =
                              String($event)
                          "
                        /></label><label class="group-models">{{
                          t(
                            '模型映射（逗号分隔）',
                            'Model patterns (comma separated)',
                          )
                        }}<Input
                          :aria-label="`${group.label} ${t('模型映射', 'model patterns')}`"
                          :disabled="!canEdit(groupKey(row.account.name, group.id))"
                          :model-value="
                            modelDrafts[
                              groupKey(row.account.name, group.id)
                            ] ?? (group.model_patterns || []).join(', ')
                          "
                          @update:model-value="
                            modelDrafts[
                              groupKey(row.account!.name, group.id)
                            ] = String($event)
                          "
                        /></label><Button
                          variant="outline"
                          :disabled="!!busy"
                          @click="saveGroup(row.account, group)"
                        >
                          {{ t('保存分组', 'Save group') }}
                        </Button>
                      </div>
                      <AntigravityQuotaPanel
                        :quota="quotas[row.account.name]"
                        :loading="quotaLoading.has(row.account.name)"
                        :disabled="row.disabled"
                        can-refresh
                        @refresh="loadQuota(row.account!, true)"
                      />
                      <p class="hub-footnote">{{ t('最近分组调度', 'Last group routing') }}：{{ schedulerLastPick && !schedulerLastPick.startsWith('0001') ? formatDateTime(schedulerLastPick) : t('暂无请求记录，空闲不影响插件就绪状态', 'No requests recorded. Idle does not mean unloaded.') }}</p>
                      <p class="hub-footnote">
                        {{
                          t(
                            '分组窗口用量依据上方模型映射归因；相同请求可同时属于短窗口和周窗口，请勿将两个窗口相加。额度与费用来自不同数据源，不做线性总额度预测。',
                            'Group usage follows model mappings. Short and weekly windows overlap; do not add them. Quota and cost have different sources, so no linear budget projection is made.',
                          )
                        }}
                      </p>
                    </template>
                    <template
                      v-else-if="row.account && row.provider === 'codex'"
                    >
                      <CodexQuotaPolicyEditor :key="`${row.id}:${policyReset}`" :name="row.account.name" :initial="row.account.quota_policy" :disabled="!canEdit(`policy:${row.id}`)" @dirty="policyDirty(`policy:${row.id}`, $event)" @saved="load" />
                      <h2>{{ t('额度窗口用量', 'Quota window usage') }}</h2>
                      <div class="codex-window-list">
                        <div
                          v-for="(usage, i) in [
                            row.account.primary_window_usage,
                            row.account.secondary_window_usage,
                          ]"
                          :key="i"
                          class="codex-window"
                        >
                          <strong>{{
                            i === 0
                              ? t('主窗口', 'Primary window')
                              : t('次窗口', 'Secondary window')
                          }}</strong><template v-if="usage">
                            <span>{{ usage.records }}
                              {{ t('请求', 'requests') }}</span><span>{{
                              formatCompact(usage.total_tokens)
                            }}
                              Tokens</span><span>{{
                              formatUsd(usage.estimated_cost_usd)
                            }}</span><span>{{ t('重置', 'Resets') }}
                              {{ formatDateTime(usage.reset_at) }}</span><span v-if="usage.stale">{{
                              t('数据需刷新', 'Refresh needed')
                            }}</span>
                          </template><span v-else>{{
                            t(
                              '窗口用量尚不可用，请在 OAuth 凭证中刷新巡检。',
                              'Window usage unavailable. Refresh inspection in OAuth credentials.',
                            )
                          }}</span>
                        </div>
                      </div>
                    </template>
                    <template v-else>
                      <h2>{{ t('接入配置', 'Connection settings') }}</h2>
                      <p>{{ row.item?.['base-url'] || row.account?.name }}</p>
                      <Button
                        variant="outline"
                        @click="
                          changeTab(
                            row.kind === 'api'
                              ? 'configuration'
                              : 'credentials',
                          )
                        "
                      >
                        {{
                          t('编辑详细配置', 'Edit connection details')
                        }}
                      </Button>
                    </template>
                  </div>
                </TableCell>
              </TableRow>
            </template>
          </TableBody>
        </Table>
        <div class="hub-list-footer">
          <span>{{
            t(
              `显示 ${rows.length} / ${allRows.length} 个上游`,
              `Showing ${rows.length} of ${allRows.length} upstreams`,
            )
          }}</span><span>{{
            t(
              '数值越大越优先 · 同级策略由 CPA / 分组插件配置',
              'Higher priority first · Tie strategy is configured in CPA / the group plugin',
            )
          }}</span>
        </div>
      </div>
      <p class="hub-footnote">
        {{
          t(
            'Codex 命中低额度规则后暂停；巡检确认额度恢复后自动恢复原优先级与路由，人工禁用保持不变。Antigravity 的 Gemini、Claude / GPT 等分组分别按自身优先级与额度调度。',
            'Codex pauses at low quota. Inspection restores the previous priority and routing after quota recovers; manual disables are retained. Antigravity groups route using their independent priorities and quotas.',
          )
        }}
      </p>
    </template>
    <UpstreamEditor v-else-if="tab === 'configuration'" embedded />
    <Credentials v-else-if="tab === 'credentials'" embedded />
    <template v-else><CodexQuotaPolicyEditor :key="policyReset" :disabled="!canEdit('policy:global')" @dirty="policyDirty('policy:global', $event)" /><InspectionSettings /></template>
  </section>
</template>

<style scoped>
.hub-draft-notice { display: flex; align-items: center; justify-content: space-between; flex-wrap: wrap; gap: 10px; padding: 10px 14px; background: var(--accent); color: var(--accent-foreground); font-size: 13px; }
.upstream-hub { container-type: inline-size; }
.upstream-hub > * { min-width: 0; max-width: 100%; }
.hub-list { min-width: 0; overflow: hidden; }
.hub-description {
  margin: 8px 0 0;
  color: var(--muted-foreground);
  font-size: 14px;
}
.hub-actions,
.hub-filters,
.hub-priority,
.hub-overview,
.group-rules-header {
  display: flex;
  align-items: center;
  gap: 12px;
  flex-wrap: wrap;
}
.hub-overview {
  gap: 14px 30px;
  font-size: 13px;
  padding: 4px 2px;
}
.hub-overview > span {
  display: inline-flex;
  align-items: center;
  gap: 8px;
}
.hub-overview strong {
  font-size: 23px;
  font-variant-numeric: tabular-nums;
}
.scheduler-state {
  margin-inline-start: auto;
  color: var(--muted-foreground);
}
.hub-filters {
  padding: 18px 20px;
  border-bottom: 1px solid var(--border);
}
.hub-search {
  position: relative;
  min-width: 180px;
  flex: 1;
  max-width: 400px;
}
.hub-search > svg {
  position: absolute;
  left: 10px;
  top: 11px;
  z-index: 1;
  color: var(--muted-foreground);
}
.hub-search :deep(input) {
  padding-left: 34px;
}
select {
  min-height: 36px;
  padding: 6px 26px 6px 10px;
  border: 1px solid var(--input);
  border-radius: 10px;
  background: var(--card);
  color: var(--foreground);
  font-size: 13px;
  max-width: 100%;
}
.hub-list :deep(.hub-table) {
  min-width: 940px;
}
.hub-list :deep(.hub-table td) {
  padding-block: 20px;
  vertical-align: middle;
}
.hub-row-name {
  display: block;
  max-width: 250px;
  overflow: hidden;
  text-overflow: ellipsis;
  font-size: 14px;
}
.hub-row-sub {
  white-space: normal;
  display: block;
  margin-top: 5px;
  max-width: 260px;
  overflow-wrap: anywhere;
  color: var(--muted-foreground);
  font-size: 12px;
}
.hub-provider {
  font-weight: 600;
}
.hub-priority :deep(input) {
  width: 78px;
}
.hub-mobile-label { display: none; font-size: 12px; color: var(--muted-foreground); }
.group-priority-summary { display: grid; gap: 8px; }
.group-priority-summary button { display: flex; justify-content: space-between; align-items: center; gap: 12px; padding: 0; background: transparent; border: 0; max-width: 220px; text-align: start; font-size: 12px; white-space: normal; }
.group-priority-summary strong { font-size: 16px; font-variant-numeric: tabular-nums; }
.priority-explanation {
  max-width: 210px;
}
.hub-quota {
  max-width: 230px;
}
.hub-quota > strong {
  font-size: 19px;
  font-variant-numeric: tabular-nums;
}
.hub-quota-track {
  height: 5px;
  margin-top: 8px;
  background: var(--muted);
  border-radius: 3px;
}
.hub-quota-track i {
  display: block;
  height: 100%;
  background: var(--cpa-success);
  border-radius: inherit;
}
.hub-quota-track .is-low {
  background: var(--destructive);
}
.hub-expanded {
  background: var(--muted);
}
.hub-detail {
  padding: 4px 20px 12px;
  max-width: 1050px;
  white-space: normal;
}
.hub-detail h2 {
  font-size: 16px;
  font-weight: 700;
  margin: 0 0 6px;
}
.group-rules-header {
  justify-content: space-between;
  margin-bottom: 16px;
}
.group-rules-header p,
.group-policy p {
  color: var(--muted-foreground);
  font-size: 12px;
  margin: 4px 0;
}
.group-policy {
  display: grid;
  grid-template-columns: minmax(160px, 1fr) 110px minmax(180px, 1.5fr) auto;
  align-items: end;
  gap: 18px;
  padding: 14px 0;
  border-bottom: 1px solid var(--border);
}
.group-policy label {
  display: grid;
  gap: 6px;
  font-size: 12px;
}
.hub-footnote {
  color: var(--muted-foreground);
  font-size: 12px;
  line-height: 1.7;
  margin: 0;
}
.hub-detail > [data-antigravity-quota] {
  margin-block: 20px;
}
.hub-detail :deep([data-antigravity-quota] section) {
  padding-block: 12px;
  border-bottom: 1px solid var(--border);
}
.hub-list-footer {
  display: flex;
  justify-content: space-between;
  flex-wrap: wrap;
  gap: 10px;
  padding: 14px 20px;
  border-top: 1px solid var(--border);
  font-size: 12px;
  color: var(--muted-foreground);
}
.hub-error {
  padding: 16px;
  border: 1px solid var(--destructive);
  background: var(--card);
  color: var(--destructive);
}
.hub-error p {
  margin: 0 0 8px;
}
.codex-window {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 18px;
  padding-block: 14px;
  border-bottom: 1px solid var(--border);
  font-variant-numeric: tabular-nums;
  font-size: 13px;
}
@container (max-width: 959px) {
  .hub-mobile-label { display: inline; }
  .hub-heading > div, .hub-actions { width: 100%; min-width: 0; }
  .hub-tabs { width: 100%; }
  .hub-list :deep(.hub-table) { min-width: 0; width: 100%; }
  .hub-list :deep(.hub-table thead) { display: none; }
  .hub-list :deep(.hub-table tbody) { display: block; }
  .hub-list :deep(.hub-table tr[data-upstream-row]) { display: grid; grid-template-columns: 32px minmax(0, 1fr) 48px; padding: 12px 4px; }
  .hub-list :deep(.hub-table td) { min-width: 0; padding: 5px 8px; white-space: normal; }
  .hub-list :deep(.hub-table tr[data-upstream-row] td:nth-child(1)) { grid-column: 1; grid-row: 1; }
  .hub-list :deep(.hub-table tr[data-upstream-row] td:nth-child(2)) { grid-column: 2; grid-row: 1; }
  .hub-list :deep(.hub-table tr[data-upstream-row] td:nth-child(3)) { grid-column: 2; grid-row: 2; }
  .hub-list :deep(.hub-table tr[data-upstream-row] td:nth-child(4)) { grid-column: 2; grid-row: 3; }
  .hub-list :deep(.hub-table tr[data-upstream-row] td:nth-child(5)) { grid-column: 2; grid-row: 4; }
  .hub-list :deep(.hub-table tr[data-upstream-row] td:nth-child(6)) { grid-column: 3; grid-row: 1; }
  .hub-list :deep(.hub-table tr.hub-expanded), .hub-list :deep(.hub-table tr.hub-expanded > td) { display: block; width: 100%; }
  .hub-row-name, .hub-row-sub, .priority-explanation, .hub-quota { max-width: 100%; }
  .hub-detail { padding: 12px 4px; }
  .group-policy { grid-template-columns: minmax(0, 1fr); gap: 10px; align-items: stretch; }
  .group-models { min-width: 0; }
  .hub-actions {
    gap: 8px;
  }
  .hub-actions :deep(button) {
    flex: 1;
  }
  .scheduler-state {
    margin-inline-start: 0;
  }
  .hub-tabs {
    max-width: 100%;
    overflow-x: auto;
    justify-content: flex-start;
  }
  .hub-filters {
    padding: 12px;
  }
  .hub-search {
    flex-basis: 100%;
    max-width: none;
  }
  .hub-filters select {
    flex: 1;
    min-width: 0;
  }
  .hub-filters > :deep(button) {
    flex-basis: 100%;
    justify-content: flex-start;
  }
  .hub-overview {
    gap: 12px 20px;
  }
}
</style>
