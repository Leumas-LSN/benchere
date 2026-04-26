<template>
  <div class="flex items-center gap-3 py-2.5 first:pt-0">
    <div class="w-9 h-9 rounded-lg bg-soft flex items-center justify-center fg-muted shrink-0">
      <Icon name="server" :size="16" />
    </div>
    <div class="flex-1 min-w-0">
      <p class="text-sm font-medium fg-primary truncate">{{ name }}</p>
      <div class="mt-1 grid grid-cols-2 gap-2.5">
        <div>
          <div class="flex items-center justify-between text-[11px] fg-muted mb-0.5">
            <span class="uppercase tracking-wide">CPU</span>
            <span class="num" :class="cpuTextColor">{{ cpu.toFixed(1) }}%</span>
          </div>
          <div class="h-1 rounded-full bg-soft overflow-hidden">
            <div
              class="h-full rounded-full transition-[width] duration-500 ease-out"
              :class="cpuBarColor"
              :style="{ width: Math.min(cpu, 100) + '%' }"
            ></div>
          </div>
        </div>
        <div>
          <div class="flex items-center justify-between text-[11px] fg-muted mb-0.5">
            <span class="uppercase tracking-wide">RAM</span>
            <span class="num text-sky-600 dark:text-sky-400">{{ ram.toFixed(1) }}%</span>
          </div>
          <div class="h-1 rounded-full bg-soft overflow-hidden">
            <div
              class="h-full rounded-full bg-sky-500 transition-[width] duration-500 ease-out"
              :style="{ width: Math.min(ram, 100) + '%' }"
            ></div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { computed } from 'vue'
import Icon from './Icon.vue'

const props = defineProps({
  name: { type: String, required: true },
  cpu:  { type: Number, default: 0 },
  ram:  { type: Number, default: 0 },
})

const cpuBarColor = computed(() =>
  props.cpu > 90 ? 'bg-red-500' :
  props.cpu > 70 ? 'bg-amber-500' :
                   'bg-brand-500'
)
const cpuTextColor = computed(() =>
  props.cpu > 90 ? 'text-red-600 dark:text-red-400' :
  props.cpu > 70 ? 'text-amber-600 dark:text-amber-400' :
                   'text-brand-600 dark:text-brand-400'
)
</script>
