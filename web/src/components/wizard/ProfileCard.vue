<template>
  <button
    type="button"
    class="profile-card"
    :class="{ 'is-selected': selected }"
    @click="$emit('toggle')"
  >
    <span class="checkbox" :class="{ 'is-checked': selected }">
      <Icon v-if="selected" name="check" :size="11" stroke-width="3" />
    </span>
    <div class="min-w-0 flex-1">
      <div class="card-head">
        <span class="card-name num">{{ name }}</span>
        <span v-if="popular && builtIn" class="pill-popular">{{ t('wizard.cards.popular') }}</span>
        <span v-if="!builtIn" class="pill-custom">{{ t('wizard.cards.custom') }}</span>
      </div>
      <p v-if="meta" class="card-meta num">{{ meta }}</p>
    </div>
  </button>
</template>

<script setup>
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import Icon from '../Icon.vue'

const props = defineProps({
  name: { type: String, required: true },
  bs: { type: String, default: '' },
  rw: { type: String, default: '' },
  estMinutes: { type: Number, default: 0 },
  popular: { type: Boolean, default: false },
  builtIn: { type: Boolean, default: true },
  selected: { type: Boolean, default: false },
})

defineEmits(['toggle'])

const { t } = useI18n()

const meta = computed(() => {
  const parts = []
  if (props.bs) parts.push('bs=' + props.bs)
  if (props.rw) parts.push(props.rw)
  if (props.estMinutes > 0) parts.push('~' + props.estMinutes + 'm')
  return parts.join(' . ')
})
</script>

<style scoped>
.profile-card {
  display: flex;
  align-items: flex-start;
  gap: 12px;
  padding: 14px 16px;
  background: var(--bg-elevated);
  border: 1px solid var(--border-default);
  border-radius: 12px;
  text-align: left;
  cursor: pointer;
  transition: border-color 120ms ease, background 120ms ease;
  width: 100%;
}

.profile-card:hover {
  border-color: var(--border-strong);
  background: var(--bg-muted);
}

.profile-card.is-selected {
  border-color: #f97316;
  background: var(--bg-brand-soft);
}

.checkbox {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 18px;
  height: 18px;
  border-radius: 5px;
  background: var(--bg-elevated);
  border: 1px solid var(--border-default);
  color: var(--fg-onbrand);
  flex-shrink: 0;
  margin-top: 1px;
}

.checkbox.is-checked {
  background: #f97316;
  border-color: #f97316;
}

.card-head {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
}

.card-name {
  font-size: 14px;
  font-weight: 600;
  color: var(--fg-primary);
}

.card-meta {
  font-size: 12px;
  color: var(--fg-muted);
  margin-top: 4px;
}

.pill-popular {
  font-size: 10px;
  font-weight: 600;
  letter-spacing: 0.06em;
  padding: 2px 6px;
  border-radius: 4px;
  background: #f97316;
  color: #ffffff;
  text-transform: uppercase;
}

.pill-custom {
  font-size: 10px;
  font-weight: 600;
  letter-spacing: 0.06em;
  padding: 2px 6px;
  border-radius: 4px;
  background: #0ea5e9;
  color: #ffffff;
  text-transform: uppercase;
}
</style>
