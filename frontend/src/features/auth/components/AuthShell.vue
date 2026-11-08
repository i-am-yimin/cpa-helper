<script setup lang="ts">
import { logoUrl } from '@/shared/utils/assets'

defineProps<{
  sectionLabel: string
}>()
</script>

<template>
  <main class="auth-screen">
    <section class="auth-brand-panel" aria-hidden="true">
      <div class="brand-stage">
        <span class="brand-word brand-word-cpa">CPA</span>
        <span class="brand-word brand-word-helper">HELPER</span>
      </div>
    </section>

    <section class="auth-content" :aria-label="sectionLabel">
      <div class="brand-mark">
        <img :src="logoUrl" alt="">
      </div>
      <slot />
    </section>
  </main>
</template>

<style scoped>
.auth-screen {
  display: grid;
  grid-template-columns: minmax(420px, 1fr) minmax(420px, 1fr);
  height: 100vh;
  height: 100dvh;
  min-height: 0;
  overflow: auto;
  background: var(--background);
}

.auth-brand-panel {
  position: relative;
  display: grid;
  min-height: 100%;
  align-items: center;
  overflow: hidden;
  padding: 72px 48px;
  background: var(--muted);
}

.auth-brand-panel::before {
  position: absolute;
  inset: 0;
  border: 1px solid var(--border);
  border-radius: 24px;
  inset: 24px;
  content: "";
}

.brand-stage {
  position: relative;
  z-index: 1;
  display: grid;
  width: min(620px, 100%);
  gap: 8px;
  align-content: center;
  justify-self: center;
  font-weight: 820;
  line-height: 0.9;
  text-transform: uppercase;
}

.brand-word {
  display: block;
  max-width: 100%;
  overflow-wrap: anywhere;
  letter-spacing: 0;
}

.brand-word-cpa {
  justify-self: start;
  color: var(--foreground);
  font-size: 148px;
}

.brand-word-helper {
  justify-self: end;
  color: var(--foreground);
  font-size: 116px;
}

.auth-content {
  display: grid;
  min-width: 0;
  align-content: center;
  justify-items: center;
  gap: 24px;
  padding: 48px;
  background: var(--card);
}

.brand-mark {
  display: grid;
  width: 68px;
  height: 68px;
  place-items: center;
  overflow: hidden;
  border: 1px solid var(--border);
  border-radius: calc(var(--radius) * 1.6);
  background: var(--card);
  box-shadow: var(--cpa-shadow-soft);
}

.brand-mark img {
  display: block;
  width: 100%;
  height: 100%;
  object-fit: cover;
}

@media (max-width: 1320px) {
  .brand-word-cpa {
    font-size: 128px;
  }

  .brand-word-helper {
    font-size: 102px;
  }
}

@media (max-width: 1180px) {
  .auth-screen {
    grid-template-columns: minmax(360px, 0.9fr) minmax(390px, 1.1fr);
  }

  .brand-word-cpa {
    font-size: 118px;
  }

  .brand-word-helper {
    font-size: 94px;
  }
}

@media (max-width: 900px) {
  .auth-screen {
    grid-template-columns: 1fr;
  }

  .auth-brand-panel {
    display: none;
  }

  .auth-content {
    min-height: 100%;
    align-content: start;
    gap: 16px;
    padding: max(28px, env(safe-area-inset-top)) 14px 20px;
  }

  .brand-mark {
    width: 56px;
    height: 56px;
    border-radius: calc(var(--radius) * 1.4);
  }
}

@media (max-width: 520px) {
  .auth-content {
    gap: 14px;
  }
}
</style>
