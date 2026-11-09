<script setup lang="ts">
import type { HTMLAttributes } from 'vue'
import { cn } from '@/lib/utils'

const props = defineProps<{
  class?: HTMLAttributes['class']
  as?: string
}>()
</script>

<template>
  <component
    :is="as || 'div'"
    data-slot="card-title"
    :class="cn('text-base leading-snug font-medium group-data-[size=sm]/card:text-sm cn-font-heading', props.class)"
  >
    <slot />
  </component>
</template>
