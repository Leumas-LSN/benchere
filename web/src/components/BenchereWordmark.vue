<template>
  <span :class="['benchere-wordmark', sizeClass]" aria-label="Benchere" role="img">
    <span class="benchere-wordmark__bench" aria-hidden="true">bench</span>
    <span class="benchere-wordmark__bar" aria-hidden="true"></span>
    <span class="benchere-wordmark__ere"   aria-hidden="true">ere</span>
  </span>
</template>

<script setup>
import { computed } from 'vue'

const props = defineProps({
  size: { type: String, default: 'md', validator: v => ['sm', 'md', 'lg', 'xl'].includes(v) },
})

const sizeClass = computed(() => `benchere-wordmark--${props.size}`)
</script>

<style scoped>
.benchere-wordmark {
  display: inline-flex;
  align-items: baseline;
  font-family: 'Geist', ui-sans-serif, system-ui, sans-serif;
  letter-spacing: -0.04em;
  color: currentColor;
  line-height: 1;
  user-select: none;
  white-space: nowrap;
}

.benchere-wordmark__bench { font-weight: 800; }
.benchere-wordmark__ere   { font-weight: 600; opacity: 0.55; }

.benchere-wordmark__bar {
  display: inline-block;
  background: #f97316;
  transform: skew(-18deg);
  align-self: center;
  border-radius: 1px;
}

.benchere-wordmark--sm { font-size: 16px; }
.benchere-wordmark--sm .benchere-wordmark__bar { width: 2px; height: 13px; margin: 0 4px; }

.benchere-wordmark--md { font-size: 22px; }
.benchere-wordmark--md .benchere-wordmark__bar { width: 3px; height: 18px; margin: 0 5px; }

.benchere-wordmark--lg { font-size: 32px; }
.benchere-wordmark--lg .benchere-wordmark__bar { width: 4px; height: 26px; margin: 0 7px; }

.benchere-wordmark--xl { font-size: 56px; letter-spacing: -0.045em; }
.benchere-wordmark--xl .benchere-wordmark__bar { width: 6px; height: 46px; margin: 0 12px; }
</style>
