<template>
  <section class="card-flush mb-6">
    <header class="card-header">
      <span class="card-title">Methodologie</span>
    </header>
    <div class="p-5 grid grid-cols-1 md:grid-cols-2 gap-x-8 gap-y-3 text-sm">
      <div class="flex items-baseline justify-between">
        <span class="fg-muted">Moteur</span><span class="font-mono fg-primary">{{ engine || 'fio' }}</span>
      </div>
      <div class="flex items-baseline justify-between">
        <span class="fg-muted">Workers</span><span class="num fg-primary">{{ workers.length }}</span>
      </div>
      <div class="flex items-baseline justify-between">
        <span class="fg-muted">Working set par worker</span><span class="num fg-primary">{{ workingSetPerWorkerGB }} GB</span>
      </div>
      <div class="flex items-baseline justify-between">
        <span class="fg-muted">Working set total</span><span class="num fg-primary">{{ workingSetTotalGB }} GB</span>
      </div>
      <div class="flex items-baseline justify-between">
        <span class="fg-muted">Warmup</span><span class="num fg-primary">{{ warmup }} s</span>
      </div>
      <div class="flex items-baseline justify-between">
        <span class="fg-muted">Runtime par profil</span><span class="num fg-primary">{{ runtime }} s</span>
      </div>
      <div class="flex items-baseline justify-between">
        <span class="fg-muted">I/O</span><span class="font-mono fg-primary">direct={{ directIO ? 1 : 0 }}, libaio</span>
      </div>
      <div class="flex items-baseline justify-between">
        <span class="fg-muted">Mesure</span><span class="fg-primary">guest worker (client distribue)</span>
      </div>
      <div class="flex items-baseline justify-between md:col-span-2">
        <span class="fg-muted">Cluster Proxmox</span>
        <span class="font-mono fg-primary truncate text-right">{{ cluster.join(', ') || 'inconnu' }}</span>
      </div>
      <div class="flex items-baseline justify-between md:col-span-2">
        <span class="fg-muted">Storage pool</span>
        <span class="font-mono fg-primary truncate text-right">{{ storagePool || 'inconnu' }}</span>
      </div>
    </div>
  </section>
</template>

<script setup>
defineProps({
  engine:               { type: String,  default: 'fio' },
  workers:              { type: Array,   default: () => [] },
  cluster:              { type: Array,   default: () => [] },
  storagePool:          { type: String,  default: '' },
  workingSetPerWorkerGB:{ type: Number,  default: 0 },
  workingSetTotalGB:    { type: Number,  default: 0 },
  runtime:              { type: Number,  default: 300 },
  warmup:               { type: Number,  default: 30 },
  directIO:             { type: Boolean, default: true },
})
</script>
