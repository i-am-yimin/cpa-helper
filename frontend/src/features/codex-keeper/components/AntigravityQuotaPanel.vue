<script setup lang="ts">
import { Button } from '@/components/ui/button'
import { Spinner } from '@/components/ui/spinner'
import { RefreshCw } from '@lucide/vue'
import { useI18n } from '@/shared/i18n'
import type { AntigravityQuotaSnapshot } from '@/shared/types/api'
import { formatDateTime } from '@/shared/utils/format'

defineProps<{
  quota?: AntigravityQuotaSnapshot
  loading: boolean
  disabled: boolean
  canRefresh: boolean
}>()
defineEmits<{ refresh: [] }>()
const { t, credentialServerText } = useI18n()

function windowLabel(window: string, fallback: string): string {
  switch (window.toLowerCase()) {
    case '5h':
    case 'five_hour': return t('5小时', '5 hours')
    case 'daily':
    case 'day': return t('每日', 'Daily')
    case 'weekly':
    case 'week': return t('每周', 'Weekly')
    default: return fallback
  }
}
</script>

<template>
  <div class="space-y-3 py-1" data-antigravity-quota :aria-busy="loading">
    <div class="flex items-center justify-between gap-2 text-xs text-muted-foreground">
      <span v-if="disabled">{{ t('凭证已禁用', 'Credential disabled') }}</span>
      <span v-else-if="quota?.checked_at" :title="formatDateTime(quota.checked_at)">
        {{ quota.stale ? t('上次查询结果 · 已过期', 'Previous result · Stale') : t('分组额度', 'Group quotas') }}
      </span>
      <span v-else>{{ loading ? t('正在查询额度…', 'Loading quota…') : t('尚未查询额度', 'Quota not loaded') }}</span>
      <Button v-if="canRefresh" variant="ghost" size="icon-sm" :disabled="disabled || loading" :aria-label="t('刷新 Antigravity 额度', 'Refresh Antigravity quota')" @click="$emit('refresh')">
        <Spinner v-if="loading" />
        <RefreshCw v-else />
      </Button>
      <Spinner v-else-if="loading" class="size-3" />
    </div>
    <p v-if="quota?.error && !disabled" role="status" class="text-xs text-destructive">{{ credentialServerText(quota.error) }}</p>
    <template v-if="!disabled">
      <section v-for="(group, index) in quota?.groups ?? []" :key="`${index}-${group.label}`" class="space-y-1.5" :class="{ 'opacity-60': quota?.stale }">
        <h4 class="text-xs font-medium break-words">{{ group.label }}</h4>
        <div v-for="bucket in group.buckets" :key="bucket.id" class="space-y-1" :title="bucket.label">
          <div class="flex justify-between gap-2 text-xs">
            <span>{{ windowLabel(bucket.window, bucket.label) }}</span>
            <strong class="tabular-nums">{{ bucket.remaining_percent }}%</strong>
          </div>
          <div role="progressbar" :aria-label="`${group.label} · ${windowLabel(bucket.window, bucket.label)}`" :aria-valuenow="bucket.remaining_percent" :aria-valuemin="0" :aria-valuemax="100" class="h-1.5 overflow-hidden rounded-full bg-muted">
            <div class="h-full rounded-full" :class="bucket.remaining_percent < 30 ? 'bg-destructive' : bucket.remaining_percent < 70 ? 'bg-amber-500' : 'bg-emerald-500'" :style="{ width: `${bucket.remaining_percent}%` }" />
          </div>
          <p class="text-[11px] text-muted-foreground tabular-nums">
            {{ t('重置', 'Resets') }}: {{ bucket.reset_at ? formatDateTime(bucket.reset_at) : t('上游未提供', 'Not provided') }}
          </p>
        </div>
      </section>
    </template>
  </div>
</template>
