<template>
  <svg
    class="animate-spin"
    :width="size"
    :height="size"
    viewBox="0 0 24 24"
    fill="none"
    aria-hidden="true"
  >
    <circle cx="12" cy="12" r="10" stroke="currentColor" stroke-width="2.5" stroke-opacity="0.18" />
    <path
      d="M22 12a10 10 0 0 1-10 10"
      stroke="currentColor"
      stroke-width="2.5"
      stroke-linecap="round"
    />
  </svg>
</template>

<script setup>
defineProps({
  size: { type: [String, Number], default: 16 },
})
</script>
