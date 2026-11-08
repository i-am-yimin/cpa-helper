<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
defineProps<{ subtle?: boolean }>()
const outline = ref<SVGSVGElement>()
const width = ref(100),
  height = ref(36)
let observer: ResizeObserver | undefined
const path = computed(() => {
  const w = width.value,
    h = height.value
  return `M 7 2.2 Q ${w * 0.45} .6 ${w - 7} 2.6 Q ${w - 1.5} 2 ${w - 2} 8 Q ${w - 0.4} ${h * 0.52} ${w - 2.5} ${h - 6} Q ${w - 3} ${h - 1.4} ${w - 8} ${h - 2} Q ${w * 0.52} ${h - 0.3} 7 ${h - 2.5} Q 1.2 ${h - 3} 2 ${h - 8} Q .6 ${h * 0.45} 2.5 7 Q 2.8 1.8 7 2.2 Z`
})
onMounted(() => {
  const host = outline.value?.parentElement
  if (!host) return
  const resize = () => {
    width.value = host.clientWidth
    height.value = host.clientHeight
  }
  resize()
  observer = new ResizeObserver(resize)
  observer.observe(host)
})
onBeforeUnmount(() => observer?.disconnect())
</script>

<template>
  <svg
    ref="outline"
    class="ink-outline size-full"
    data-ink="outline"
    :class="{ 'ink-subtle': subtle }"
    :viewBox="`0 0 ${width} ${height}`"
    preserveAspectRatio="none"
    aria-hidden="true"
    focusable="false"
  >
    <path :d="path" fill="none" vector-effect="non-scaling-stroke" />
  </svg>
</template>

<style scoped>
svg {
  position: absolute;
  inset: 0;
  width: 100%;
  height: 100%;
  pointer-events: none;
  overflow: visible;
  stroke: var(--foreground);
  stroke-width: 1.15px;
}
.ink-subtle {
  stroke: var(--input);
  stroke-width: 1px;
}
</style>
