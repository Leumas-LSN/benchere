<template>
  <div
    class="w-full h-1.5 rounded-full overflow-hidden"
    :class="trackClass"
    role="progressbar"
    :aria-valuenow="clamped"
    aria-valuemin="0"
    aria-valuemax="100"
  >
    <div
      class="h-full rounded-full transition-[width] duration-500 ease-out"
      :class="barClass"
      :style="`width:${clamped}%`"
    ></div>
  </div>
</template>

<script setup>
import { computed } from 'vue'

const props = defineProps({
  value: { type: Number, required: true },     // 0..100 OR 0..1 (auto-detected)
  tone:  { type: String, default: 'brand' },   // brand | success | warn | danger | info | neutral
})

const clamped = computed(() => {
  let v = Number(props.value)
  if (Number.isNaN(v)) return 0
  if (v <= 1) v = v * 100
  return Math.max(0, Math.min(100, v))
})

const barClass = computed(() => {
  switch (props.tone) {
    case 'success':  return 'bg-emerald-500'
    case 'warn':     return 'bg-amber-500'
    case 'danger':   return 'bg-red-500'
    case 'info':     return 'bg-sky-500'
    case 'neutral':  return 'bg-ink-400 dark:bg-ink-500'
    case 'brand':
    default:         return 'bg-brand-500'
  }
})

const trackClass = computed(() => 'bg-ink-100 dark:bg-ink-800')
</script>
