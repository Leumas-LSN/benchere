<template>
  <div class="page">
    <PageHeader
      :eyebrow="t('profiles.title')"
      :title="t('profiles.title')"
      :description="t('profiles.description')"
    >
      <template #actions>
        <button class="btn-primary btn-sm" @click="showImport = true">
          <Icon name="upload" :size="14" />
          {{ t('profiles.new') }}
        </button>
      </template>
    </PageHeader>

    <div v-if="error" class="alert-error mb-6">
      <Icon name="x_circle" :size="18" class="mt-0.5 shrink-0" />
      <span>{{ error }}</span>
    </div>

    <div v-if="profiles.length === 0 && !error" class="surface">
      <EmptyState
        icon="layers"
        title="Aucun profil"
        description="Importez votre premier profil elbencho pour commencer."
      >
        <template #action>
          <button class="btn-primary btn-sm" @click="showImport = true">
            <Icon name="upload" :size="14" /> Importer
          </button>
        </template>
      </EmptyState>
    </div>

    <div class="grid grid-cols-1 gap-3">
      <div
        v-for="profile in profiles"
        :key="profile.id"
        class="card transition-all duration-150"
        :class="editId === profile.id ? 'ring-2 ring-brand-500/40' : ''"
      >
        <div class="flex items-start justify-between gap-4">
          <div class="flex items-start gap-3 min-w-0 flex-1">
            <div class="w-10 h-10 rounded-lg bg-brand-50 text-brand-600 dark:bg-brand-500/10 dark:text-brand-400 flex items-center justify-center shrink-0">
              <Icon name="flask" :size="18" />
            </div>
            <div class="min-w-0 flex-1">
              <div class="flex items-center gap-2 flex-wrap">
                <span class="font-mono text-sm font-semibold fg-primary truncate">{{ profile.name }}</span>
                <span
                  v-if="profile.is_builtin"
                  class="text-[10px] font-semibold uppercase tracking-wide px-1.5 py-0.5 rounded bg-soft fg-muted"
                >
                  intégré
                </span>
              </div>
              <p
                v-if="profile.description"
                class="text-sm fg-secondary mt-1"
              >{{ profile.description }}</p>
              <p v-else class="text-sm fg-faint italic mt-1">{{ t('common.noDescription') }}</p>

              <div
                v-if="parsedThresholds(profile)"
                class="mt-3 flex flex-wrap gap-1.5"
              >
                <span
                  v-if="parsedThresholds(profile).min_iops_read"
                  class="pill"
                >
                  IOPS R ≥ <span class="num ml-1 fg-primary">{{ parsedThresholds(profile).min_iops_read.toLocaleString() }}</span>
                </span>
                <span
                  v-if="parsedThresholds(profile).min_iops_write"
                  class="pill"
                >
                  IOPS W ≥ <span class="num ml-1 fg-primary">{{ parsedThresholds(profile).min_iops_write.toLocaleString() }}</span>
                </span>
                <span
                  v-if="parsedThresholds(profile).max_latency_ms"
                  class="pill"
                >
                  latence ≤ <span class="num ml-1 fg-primary">{{ parsedThresholds(profile).max_latency_ms }} ms</span>
                </span>
              </div>
            </div>
          </div>
          <div class="flex items-center gap-1.5 shrink-0">
            <button class="btn-ghost btn-sm" @click="startEdit(profile)" title="Éditer">
              <Icon name="pencil" :size="14" />
              <span class="hidden md:inline">Éditer</span>
            </button>
            <button
              v-if="!profile.is_builtin"
              class="btn-danger-ghost btn-sm"
              @click="doDelete(profile)"
              title="Supprimer"
            >
              <Icon name="trash" :size="14" />
            </button>
          </div>
        </div>

        <!-- Inline edit -->
        <div
          v-if="editId === profile.id"
          class="mt-5 pt-5 space-y-4 animate-fade-in"
          style="border-top: 1px solid var(--border-subtle);"
        >
          <div>
            <label class="label">Description</label>
            <input v-model="editForm.description" class="input" />
          </div>
          <div>
            <label class="label">Seuils de validation (JSON)</label>
            <textarea
              v-model="editForm.thresholds_json"
              class="textarea font-mono text-xs"
              rows="3"
              placeholder='{"min_iops_read":5000,"min_iops_write":3000,"max_latency_ms":2.0}'
            ></textarea>
            <p class="helper">
              Champs supportés : <span class="code-inline">min_iops_read</span>,
              <span class="code-inline">min_iops_write</span>, <span class="code-inline">max_latency_ms</span>.
              Laisser vide pour désactiver.
            </p>
          </div>
          <div v-if="editError" class="alert-error">
            <Icon name="x_circle" :size="16" /> {{ editError }}
          </div>
          <div class="flex items-center gap-2">
            <button class="btn-primary btn-sm" :disabled="saving" @click="doSave(profile.id)">
              <Spinner v-if="saving" :size="14" />
              <Icon v-else name="check" :size="14" />
              Sauvegarder
            </button>
            <button class="btn-ghost btn-sm" @click="editId = null; editError = ''">{{ t("common.cancel") }}</button>
          </div>
        </div>
      </div>
    </div>

    <!-- Import modal -->
    <Teleport to="body">
      <div v-if="showImport" class="modal-backdrop" @click.self="showImport = false">
        <div class="modal-panel p-6 space-y-5">
          <div class="flex items-center justify-between">
            <h3 class="text-lg font-semibold fg-primary">{{ t('profiles.new') }} elbencho</h3>
            <button class="btn-ghost btn-sm h-8 w-8 px-0" @click="showImport = false" aria-label="Fermer">
              <Icon name="x" :size="16" />
            </button>
          </div>
          <div class="space-y-4">
            <div>
              <label class="label">Nom du profil</label>
              <input v-model="importForm.name" class="input" placeholder="ex: 16k_50read_100random" />
            </div>
            <div>
              <label class="label">Description (optionnelle)</label>
              <input v-model="importForm.description" class="input" placeholder="ex: Mixte 16K read-heavy" />
            </div>
            <div>
              <label class="label">Contenu du fichier .elbencho</label>
              <textarea
                v-model="importForm.config"
                class="textarea font-mono text-xs"
                rows="6"
                placeholder="block=16k&#10;rwmixpct=50&#10;rand=true&#10;threads=8&#10;iterations=1"
              ></textarea>
            </div>
            <div v-if="importError" class="alert-error">
              <Icon name="x_circle" :size="16" /> {{ importError }}
            </div>
          </div>
          <div class="flex items-center justify-end gap-2 pt-1">
            <button class="btn-ghost" @click="showImport = false; importError = ''">{{ t("common.cancel") }}</button>
            <button class="btn-primary" :disabled="importing" @click="doImport">
              <Spinner v-if="importing" :size="14" />
              <Icon v-else name="upload" :size="14" />
              Importer
            </button>
          </div>
        </div>
      </div>
    </Teleport>
  </div>
</template>

<script setup>
import { useI18n } from 'vue-i18n'
import { ref, onMounted } from 'vue'
import { api } from '../api/client.js'
import PageHeader from '../components/PageHeader.vue'
import EmptyState from '../components/EmptyState.vue'
import Icon       from '../components/Icon.vue'
import Spinner    from '../components/Spinner.vue'

const { t } = useI18n()

const profiles    = ref([])
const error       = ref('')
const showImport  = ref(false)
const importing   = ref(false)
const importError = ref('')
const importForm  = ref({ name: '', config: '', description: '' })

const editId    = ref(null)
const editForm  = ref({ description: '', thresholds_json: '' })
const editError = ref('')
const saving    = ref(false)

async function load() {
  try { profiles.value = (await api.listProfiles()) ?? []; error.value = '' }
  catch { error.value = 'Impossible de charger les profils.' }
}

function parsedThresholds(profile) {
  if (!profile.thresholds_json) return null
  try { return JSON.parse(profile.thresholds_json) }
  catch { return null }
}

function startEdit(profile) {
  editId.value    = profile.id
  editError.value = ''
  editForm.value  = {
    description:     profile.description    ?? '',
    thresholds_json: profile.thresholds_json ?? '',
  }
}

async function doSave(id) {
  editError.value = ''
  if (editForm.value.thresholds_json) {
    try { JSON.parse(editForm.value.thresholds_json) }
    catch { editError.value = 'JSON invalide'; return }
  }
  saving.value = true
  try {
    await api.updateProfile(id, {
      description:     editForm.value.description,
      thresholds_json: editForm.value.thresholds_json,
    })
    editId.value = null
    await load()
  } catch (e) {
    editError.value = e.message ?? 'Erreur lors de la sauvegarde'
  } finally {
    saving.value = false
  }
}

async function doImport() {
  importError.value = ''
  if (!importForm.value.name || !importForm.value.config) {
    importError.value = 'Nom et contenu obligatoires.'
    return
  }
  importing.value = true
  try {
    await api.createProfile({
      name:        importForm.value.name,
      config:      importForm.value.config,
      description: importForm.value.description,
    })
    showImport.value = false
    importForm.value = { name: '', config: '', description: '' }
    await load()
  } catch (e) {
    importError.value = e.message ?? "Erreur lors de l'import"
  } finally {
    importing.value = false
  }
}

async function doDelete(profile) {
  if (!confirm('Supprimer le profil ' + profile.name + ' ?')) return
  try { await api.deleteProfile(profile.id); await load() }
  catch { error.value = 'Erreur lors de la suppression.' }
}

onMounted(load)
</script>
