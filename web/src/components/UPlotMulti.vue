<template>
  <div ref="root" class="w-full h-full"></div>
</template>

<script setup>
import { ref, onMounted, onBeforeUnmount, watch } from 'vue'
import uPlot from 'uplot'
import 'uplot/dist/uPlot.min.css'

const props = defineProps({
  series: { type: Array, default: () => [] }, // [{label, color, data:[]}]
  log:    { type: Boolean, default: false },
})

const root = ref(null)
let plot = null

function readVar(name, fallback) {
  if (typeof getComputedStyle === 'undefined') return fallback
  const v = getComputedStyle(document.documentElement).getPropertyValue(name).trim()
  return v || fallback
}

function buildOpts() {
  const grid = readVar('--chart-grid', '#e5e5e5')
  const text = readVar('--chart-text', '#525252')
  const series = [{}]
  for (const s of props.series) {
    series.push({
      label: s.label,
      stroke: s.color,
      width: 2,
      points: { show: false },
    })
  }
  return {
    width: root.value.clientWidth,
    height: root.value.clientHeight,
    legend: {
      show: true, live: false,
      mount: (self, el) => { el.style.fontSize = '11px'; el.style.fontFamily = 'Geist Mono' },
    },
    cursor: { drag: { x: false, y: false } },
    scales: { x: { time: false }, y: { auto: true, distr: props.log ? 3 : 1 } },
    axes: [
      { stroke: text, grid: { stroke: grid }, ticks: { show: false } },
      { stroke: text, grid: { stroke: grid }, size: 36, font: '10px Geist Mono' },
    ],
    series,
  }
}

function buildData() {
  const len = props.series.length === 0 ? 0 : (props.series[0].data?.length || 0)
  const xs = []
  for (let i = 0; i < len; i++) xs.push(i)
  return [xs, ...props.series.map(s => s.data.slice())]
}

function build() {
  if (!root.value) return
  plot = new uPlot(buildOpts(), buildData(), root.value)
}

function update() {
  if (!plot) return
  if (typeof document !== 'undefined' && document.hidden) return
  plot.setData(buildData())
}

let ro = null
onMounted(() => {
  build()
  ro = new ResizeObserver(() => {
    if (plot && root.value) plot.setSize({ width: root.value.clientWidth, height: root.value.clientHeight })
  })
  ro.observe(root.value)
})
onBeforeUnmount(() => {
  ro?.disconnect()
  plot?.destroy()
  plot = null
})

watch(() => props.series.map(s => s.data.length).join(','), update)
</script>
