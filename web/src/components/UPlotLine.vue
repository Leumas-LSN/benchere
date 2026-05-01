<template>
  <div ref="root" class="w-full h-full"></div>
</template>

<script setup>
import { ref, onMounted, onBeforeUnmount, watch } from 'vue'
import uPlot from 'uplot'
import 'uplot/dist/uPlot.min.css'

const props = defineProps({
  label: { type: String, default: '' },
  data:   { type: Array, default: () => [] }, // y values
  labels: { type: Array, default: () => [] }, // x as time strings (we synthesize numeric x by index)
  color:  { type: String, default: '#f97316' },
})

const root = ref(null)
let plot = null

function readVar(name, fallback) {
  if (typeof getComputedStyle === 'undefined') return fallback
  const v = getComputedStyle(document.documentElement).getPropertyValue(name).trim()
  return v || fallback
}

function build() {
  if (!root.value) return
  const grid = readVar('--chart-grid', '#e5e5e5')
  const text = readVar('--chart-text', '#525252')
  const opts = {
    width: root.value.clientWidth,
    height: root.value.clientHeight,
    legend: { show: false },
    cursor: { drag: { x: false, y: false }, points: { size: 6 } },
    scales: { x: { time: false }, y: { auto: true } },
    axes: [
      { stroke: text, grid: { stroke: grid }, ticks: { show: false } },
      { stroke: text, grid: { stroke: grid }, size: 32, font: '10px Geist Mono' },
    ],
    series: [
      {},
      {
        label: props.label,
        stroke: props.color,
        width: 2,
        fill: props.color + '22',
        points: { show: false },
      },
    ],
  }
  const data = [
    props.data.map((_, i) => i),
    props.data.slice(),
  ]
  plot = new uPlot(opts, data, root.value)
}

function update() {
  if (!plot) return
  if (typeof document !== 'undefined' && document.hidden) return
  const x = props.data.map((_, i) => i)
  plot.setData([x, props.data.slice()])
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

watch(() => props.data, update, { deep: false, flush: 'post' })
</script>
