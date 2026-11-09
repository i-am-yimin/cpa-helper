<script setup lang="ts">
import { logoUrl } from '@/shared/utils/assets'

defineProps<{
  sectionLabel: string
}>()
</script>

<template>
  <main class="auth-screen">
    <section class="auth-content" :aria-label="sectionLabel">
      <div class="brand-mark"><img :src="logoUrl" alt=""></div>
      <slot />
    </section>
  </main>
</template>

<style scoped>
.auth-screen {
  display: grid;
  align-items: safe center;
  justify-items: center;
  height: 100vh;
  height: 100dvh;
  min-height: 0;
  overflow: auto;
  padding: 48px 24px;
  background: var(--cpa-auth-light-band);
}
.auth-content {
  display: grid;
  width: min(460px, 100%);
  min-width: 0;
  justify-items: center;
  align-content: center;
  gap: 26px;
  padding: 40px 16px 24px;
  border-radius: 30px;
  border: 1px solid var(--cpa-glass-border);
  background: var(--cpa-glass-floating);
  -webkit-backdrop-filter: var(--cpa-glass-floating-filter);
  backdrop-filter: var(--cpa-glass-floating-filter);
  box-shadow: var(--cpa-glass-floating-shadow);
  animation: auth-arrive 420ms var(--cpa-ease-out);
}
.brand-mark { display: grid; width: 68px; height: 68px; place-items: center; overflow: hidden; border-radius: 20px; background: var(--card); }
.brand-mark img { display: block; width: 100%; height: 100%; object-fit: cover; }
.auth-content :deep([data-slot="card"]) { gap: 28px; padding-top: 0; border: 0; border-radius: 0; background: transparent; -webkit-backdrop-filter: none; backdrop-filter: none; box-shadow: none; }
.auth-content :deep([data-slot="card-header"]) { gap: 10px; }
.auth-content :deep(h1) { margin: 0; font-size: 30px; font-weight: 700; letter-spacing: -.035em; line-height: 1.25; }
.auth-content :deep([data-slot="card-description"]) { color: var(--muted-foreground); font-size: 14px; line-height: 1.65; }
.auth-content :deep([data-slot="field-label"]) { font-size: 13px; }
.auth-content :deep([data-slot="input"]), .auth-content :deep([data-slot="input-group"]) { min-height: 46px; background: var(--cpa-glass-slot); border-color: color-mix(in srgb, var(--input) 54%, transparent); box-shadow: inset 0 1px 3px rgb(0 0 0 / 5%), 0 1px 0 var(--cpa-glass-top); }
.auth-content :deep([data-slot="input"]:focus-visible), .auth-content :deep([data-slot="input-group"]:focus-within) { border-color: var(--ring); }
.auth-content :deep(button[type="submit"]) { min-height: 46px; margin-top: 4px; font-size: 15px; }
@keyframes auth-arrive { from { opacity: .5; transform: translateY(12px); } to { opacity: 1; transform: translateY(0); } }
@media (max-width: 520px) {
  .auth-screen { place-items: start center; padding: max(40px, env(safe-area-inset-top)) 16px 24px; }
  .auth-content { gap: 24px; padding: 32px 4px 20px; border-radius: 26px; }
  .auth-content :deep(h1) { font-size: 26px; }
  .brand-mark { width: 60px; height: 60px; border-radius: 18px; }
}
</style>
