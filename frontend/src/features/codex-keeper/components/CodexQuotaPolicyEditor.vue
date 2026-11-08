<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { toast } from 'vue-sonner'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Checkbox } from '@/components/ui/checkbox'
import { Select, SelectTrigger, SelectValue, SelectContent, SelectItem } from '@/components/ui/select'
import { getCodexQuotaPolicy, saveCodexQuotaPolicy } from '../api/codexKeeperApi'
import type { CodexQuotaPolicy, CodexQuotaRule } from '@/shared/types/api'
import { useI18n } from '@/shared/i18n'
import { formatDateTime } from '@/shared/utils/format'

const props = defineProps<{ name?: string; initial?: CodexQuotaPolicy; disabled?: boolean }>()
const emit = defineEmits<{ dirty: [value: boolean]; saved: [] }>()
const { t, errorText } = useI18n()
const state = ref<CodexQuotaPolicy>()
const mode = ref<CodexQuotaRule['mode']>(props.name ? 'inherit' : 'off')
const fiveEnabled = ref(false), weekEnabled = ref(false)
const five = ref('10'), week = ref('10'), busy = ref(false), ready = ref(!!props.name), error = ref('')
const baseline = ref('')
const serialized = computed(() => JSON.stringify([mode.value, fiveEnabled.value, weekEnabled.value, five.value, week.value]))
const dirty = computed(() => !!baseline.value && serialized.value !== baseline.value)
watch(dirty, value => emit('dirty', value))
function apply(value?: CodexQuotaPolicy) {
  state.value = value
  mode.value = value?.mode || (props.name ? 'inherit' : 'off')
  fiveEnabled.value = value?.five_hour_remaining != null
  weekEnabled.value = value?.week_remaining != null
  five.value = String(value?.five_hour_remaining ?? 10)
  week.value = String(value?.week_remaining ?? 10)
  baseline.value = serialized.value
}
watch(() => props.initial, value => { if (!dirty.value) apply(value) }, { immediate: true })
async function load() {
  busy.value = true; error.value = ''
  try { apply(await getCodexQuotaPolicy(props.name)); ready.value = true }
  catch (e) { error.value = errorText(e, '规则读取失败，请重试', 'Could not load rules. Retry.') }
  finally { busy.value = false }
}
onMounted(() => { if (!props.name) void load() })
async function save() {
  error.value = ''
  const threshold = (enabled: boolean, value: string) => enabled ? (String(value).trim() ? Number(value) : Number.NaN) : null
  const rule: CodexQuotaRule = { mode: mode.value, five_hour_remaining: threshold(fiveEnabled.value, five.value), week_remaining: threshold(weekEnabled.value, week.value) }
  if (rule.mode === 'custom' && (![rule.five_hour_remaining, rule.week_remaining].some(v => v != null) || [rule.five_hour_remaining, rule.week_remaining].some(v => v != null && (!Number.isInteger(v) || v < 0 || v > 100)))) {
    error.value = t('至少启用一个窗口，剩余下限须为 0–100 的整数。', 'Enable a window and enter an integer from 0 to 100.'); return
  }
  busy.value = true
  try {
    apply(await saveCodexQuotaPolicy(rule, props.name))
    emit('dirty', false)
    toast.success(t('规则已保存；已有暂停需要手动恢复', 'Rules saved. Existing pauses require manual resume.'))
    emit('saved')
  } catch (e) { error.value = errorText(e, '保存失败，请重试', 'Save failed. Retry.') }
  finally { busy.value = false }
}
</script>

<template>
  <section class="quota-policy-editor" :aria-label="t('Codex 低额度暂停规则', 'Codex low quota pause rules')">
    <div class="quota-policy-heading">
      <div>
        <h2>{{ name ? t('此凭证的低额度暂停', 'Low quota pause for this credential') : t('Codex 低额度暂停', 'Codex low quota pause') }}</h2>
        <p>{{ t('每 30 秒检查，任一窗口命中即禁用路由并置为 -1。额度恢复后仍需手动恢复。', 'Checked every 30 seconds. Either threshold disables routing and sets -1. Recovery requires manual resume.') }}</p>
      </div>
      <Select v-if="ready" :model-value="mode" :disabled="busy || disabled" @update:model-value="mode = $event as CodexQuotaRule['mode']">
        <SelectTrigger class="quota-policy-mode" :aria-label="t('额度规则模式', 'Quota rule mode')"><SelectValue /></SelectTrigger>
        <SelectContent>
          <SelectItem v-if="name" value="inherit">{{ t('继承全局规则', 'Inherit global rules') }}</SelectItem>
          <SelectItem value="custom">{{ name ? t('单独设置', 'Custom rules') : t('启用规则', 'Enable rules') }}</SelectItem>
          <SelectItem value="off">{{ t('关闭规则', 'Disable rules') }}</SelectItem>
        </SelectContent>
      </Select>
    </div>
    <div v-if="ready && mode === 'custom'" class="quota-threshold-fields">
      <div class="quota-threshold-field">
        <label class="quota-threshold-label"><Checkbox :model-value="fiveEnabled" :disabled="busy || disabled" @update:model-value="fiveEnabled = $event === true" />{{ t('5 小时剩余额度 ≤', '5-hour remaining ≤') }}</label>
        <Input v-model="five" :disabled="!fiveEnabled || busy || disabled" type="number" min="0" max="100" step="1" :aria-label="t('5 小时剩余额度下限', '5-hour remaining threshold')" /><span>%</span>
      </div>
      <div class="quota-threshold-field">
        <label class="quota-threshold-label"><Checkbox :model-value="weekEnabled" :disabled="busy || disabled" @update:model-value="weekEnabled = $event === true" />{{ t('周剩余额度 ≤', 'Weekly remaining ≤') }}</label>
        <Input v-model="week" :disabled="!weekEnabled || busy || disabled" type="number" min="0" max="100" step="1" :aria-label="t('周剩余额度下限', 'Weekly remaining threshold')" /><span>%</span>
      </div>
    </div>
    <p v-if="ready && mode === 'inherit'" class="quota-policy-hint">
      {{ t('跟随全局设置；可在“巡检与默认规则”中配置。', 'Follows global settings in Inspection & defaults.') }}
      <span v-if="state?.effective_rule?.mode === 'custom'">{{ t('5 小时下限', '5-hour threshold') }} {{ state.effective_rule.five_hour_remaining == null ? '—' : `${state.effective_rule.five_hour_remaining}%` }} · {{ t('周下限', 'Weekly threshold') }} {{ state.effective_rule.week_remaining == null ? '—' : `${state.effective_rule.week_remaining}%` }}</span>
      <span v-else>{{ t('全局规则当前关闭。', 'Global rules are currently off.') }}</span>
    </p>
    <p v-if="state?.paused" class="quota-policy-paused" role="status">{{ t('额度触发暂停', 'Paused by quota') }} · {{ state.reason }}<br>{{ state.sync_state === 'paused' ? t('CPA 已确认停止路由', 'CPA confirmed routing is disabled') : t('暂停尚未确认，请检查同步状态', 'Pause not confirmed; check synchronization') }} · {{ formatDateTime(state.paused_at) }}</p>
    <p v-if="state?.sync_error" class="text-destructive" role="alert">{{ t('同步失败', 'Sync failed') }}：{{ state.sync_error }}</p>
    <p v-if="state?.manual_disabled" class="quota-policy-hint">{{ t('此凭证另有人工禁用。先用列表中的启用开关解除人工禁用，再指定优先级恢复额度暂停。', 'This credential is also manually disabled. Clear manual disable with the list toggle, then set a priority to resume the quota pause.') }}</p>
    <p v-if="error" class="text-destructive" role="alert">{{ error }}</p>
    <div class="quota-policy-footer">
      <span>{{ t('关闭阈值不会自动放行；请在列表中指定优先级并恢复。此规则会实际执行，不受巡检模拟模式影响。', 'Disabling thresholds does not resume routing. Set a priority and resume in the list. These rules apply even during inspection dry runs.') }}</span>
      <div class="quota-policy-actions"><Button v-if="dirty" variant="ghost" :disabled="busy" @click="apply(state)">{{ t('放弃更改', 'Discard changes') }}</Button><Button v-if="ready" :disabled="busy || disabled || !dirty" @click="save">{{ busy ? t('保存中…', 'Saving…') : t('保存暂停规则', 'Save pause rules') }}</Button><Button v-else variant="outline" :disabled="busy" @click="load">{{ busy ? t('加载中…', 'Loading…') : t('重试', 'Retry') }}</Button></div>
    </div>
  </section>
</template>

<style scoped>
.quota-policy-editor { display: grid; gap: 16px; padding-block: 20px; min-width: 0; }
.quota-policy-heading { display: flex; align-items: start; justify-content: space-between; gap: 20px; }
.quota-policy-heading h2 { font-size: 16px; font-weight: 600; letter-spacing: -0.02em; }
.quota-policy-heading p, .quota-policy-hint, .quota-policy-footer { color: var(--muted-foreground); font-size: 13px; line-height: 1.65; }
.quota-policy-heading p { margin-top: 6px; max-width: 66ch; }
.quota-policy-mode { width: 180px; flex-shrink: 0; }
.quota-threshold-fields { display: flex; flex-wrap: wrap; gap: 16px 32px; }
.quota-threshold-field { display: flex; align-items: center; gap: 8px; }
.quota-threshold-label { display: flex; align-items: center; gap: 8px; font-size: 13px; }
.quota-threshold-field :deep(input) { width: 76px; font-variant-numeric: tabular-nums; }
.quota-policy-hint { display: grid; gap: 4px; }
.quota-policy-paused { padding: 12px 16px; background: var(--cpa-warning-weak); color: var(--cpa-warning); border-radius: 12px; font-size: 13px; line-height: 1.7; }
.quota-policy-footer { display: flex; align-items: center; justify-content: space-between; gap: 16px; }
.quota-policy-footer > span { max-width: 68ch; }
.quota-policy-actions { display: flex; flex-shrink: 0; gap: 8px; }
@media (max-width: 767px) { .quota-policy-heading, .quota-policy-footer { flex-direction: column; gap: 12px; } .quota-policy-mode { width: 100%; } .quota-threshold-fields { gap: 12px; } }
</style>
