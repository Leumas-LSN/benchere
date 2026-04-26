<template>
  <header class="mb-7 flex flex-col gap-4 md:flex-row md:items-end md:justify-between">
    <div>
      <p v-if="eyebrow" class="section-eyebrow mb-1.5">{{ eyebrow }}</p>
      <h1 class="text-2xl font-semibold tracking-tight fg-primary">
        <slot name="title">{{ title }}</slot>
      </h1>
      <p v-if="description || $slots.description" class="mt-1.5 text-sm fg-secondary max-w-2xl">
        <slot name="description">{{ description }}</slot>
      </p>
    </div>
    <div v-if="$slots.actions" class="flex items-center gap-2 flex-wrap">
      <slot name="actions" />
    </div>
  </header>
</template>

<script setup>
defineProps({
  title:       { type: String, default: '' },
  eyebrow:     { type: String, default: '' },
  description: { type: String, default: '' },
})
</script>
