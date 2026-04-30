<template>
  <div class="relative w-full h-full">
    <canvas ref="canvas"></canvas>
  </div>
</template>

<script setup>
import { ref, onMounted, onUnmounted, watch, watchEffect } from 'vue'
import { useThemeStore } from '../stores/theme.js'
import {
  Chart,
  LineElement,
  PointElement,
  LinearScale,
  CategoryScale,
  Tooltip,
  Filler,
  LineController,
} from 'chart.js'

Chart.register(LineController, LineElement, PointElement, LinearScale, CategoryScale, Tooltip, Filler)

const props = defineProps({
  label:  { type: String, default: '' },
  data:   { type: Array,  default: () => [] },
  labels: { type: Array,  default: () => [] },
  color:  { type: String, default: '#f97316' },
})

const canvas = ref(null)
const theme = useThemeStore()
let chart = null

function readVar(name, fallback) {
  if (typeof getComputedStyle === 'undefined') return fallback
  const v = getComputedStyle(document.documentElement).getPropertyValue(name).trim()
  return v || fallback
}

function chartColors() {
  return {
    grid:  readVar('--chart-grid',  '#e5e5e5'),
    text:  readVar('--chart-text',  '#525252'),
    axis:  readVar('--chart-axis',  '#737373'),
    tipBg: theme.theme === 'dark' ? '#1c1c1f' : '#171717',
    tipFg: '#fafafa',
    tipMu: '#a3a3a3',
  }
}

function buildGradient(ctx, color) {
  const h = ctx.canvas.height || 200
  const g = ctx.createLinearGradient(0, 0, 0, h)
  g.addColorStop(0, color + '55')
  g.addColorStop(1, color + '00')
  return g
}

function applyTheme() {
  if (!chart) return
  const c = chartColors()
  chart.options.scales.y.grid.color  = c.grid
  chart.options.scales.y.ticks.color = c.text
  chart.options.scales.y.border = { display: false }
  chart.options.plugins.tooltip.backgroundColor = c.tipBg
  chart.options.plugins.tooltip.titleColor = c.tipFg
  chart.options.plugins.tooltip.bodyColor = c.tipMu
  chart.options.plugins.tooltip.borderColor = 'transparent'
  chart.update('none')
}

onMounted(() => {
  const ctx = canvas.value.getContext('2d')
  const c = chartColors()
  chart = new Chart(canvas.value, {
    type: 'line',
    data: {
      labels: [...props.labels],
      datasets: [{
        label:           props.label,
        data:            [...props.data],
        borderColor:     props.color,
        backgroundColor: buildGradient(ctx, props.color),
        fill:            true,
        tension:         0.32,
        pointRadius:     0,
        pointHoverRadius: 4,
        borderWidth:     2,
      }],
    },
    options: {
      animation:           false,
      responsive:          true,
      maintainAspectRatio: false,
      interaction: { intersect: false, mode: 'index' },
      scales: {
        x: { display: false },
        y: {
          beginAtZero: true,
          grid: { color: c.grid, drawBorder: false },
          border: { display: false },
          ticks: { color: c.text, font: { size: 10, family: '"Geist Mono", ui-monospace, monospace' }, maxTicksLimit: 4 },
        },
      },
      plugins: {
        legend: { display: false },
        tooltip: {
          backgroundColor: c.tipBg,
          titleColor: c.tipFg,
          bodyColor: c.tipMu,
          borderColor: 'transparent',
          padding: 10,
          cornerRadius: 8,
          displayColors: false,
          titleFont:  { size: 11, weight: '500', family: 'Geist, sans-serif' },
          bodyFont:   { size: 12, weight: '600', family: '"Geist Mono", monospace' },
        },
      },
    },
  })
})

watch(
  [() => props.data, () => props.labels],
  ([newData, newLabels]) => {
    if (!chart) return
    // Skip Chart.js layout + raster when the tab is in the background.
    // Chart.js still touches the canvas even with animation: false, which
    // adds up across 4-5 charts and slows hidden-tab CPU usage.
    if (typeof document !== 'undefined' && document.hidden) return
    chart.data.datasets[0].data   = [...(newData  ?? [])]
    chart.data.labels             = [...(newLabels ?? [])]
    chart.update('none')
  },
  { deep: true },
)

watchEffect(() => {
  // re-apply theme colors on dark/light toggle
  void theme.theme
  applyTheme()
})

onUnmounted(() => {
  chart?.destroy()
  chart = null
})
</script>
