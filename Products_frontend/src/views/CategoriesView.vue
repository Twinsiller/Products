<template>
  <section class="card">
    <header class="header">
      <div>
        <h2>Категории</h2>
        <p class="muted">
          Справочник категорий товарных позиций.
        </p>
      </div>
      <div class="header-actions">
        <button class="secondary" type="button" @click="loadCategories" :disabled="loading">
          {{ loading ? 'Обновление...' : 'Обновить' }}
        </button>
        <button
          v-if="isAdmin"
          class="primary"
          type="button"
          @click="openCreate"
        >
          Добавить категорию
        </button>
      </div>
    </header>

    <div class="tiles-grid">
      <article v-for="c in categories" :key="c.id" class="tile">
        <p class="tile-meta">ID {{ c.id }}</p>
        <input
          v-if="editingId === c.id"
          v-model="editName"
          type="text"
          class="inline-edit-input"
          @keydown.enter.prevent="saveEditCategory"
          @keydown.esc="cancelEdit"
        />
        <h3 v-else class="tile-title">{{ c.name }}</h3>
        <div v-if="isAdmin" class="tile-actions">
          <template v-if="editingId === c.id">
            <button type="button" class="btn-save small" :disabled="updateLoading" @click="saveEditCategory">
              {{ updateLoading ? '…' : 'Сохранить' }}
            </button>
            <button type="button" class="secondary small" :disabled="updateLoading" @click="cancelEdit">
              Отмена
            </button>
          </template>
          <template v-else>
            <button
              type="button"
              class="btn-edit"
              :disabled="deleteId === c.id"
              title="Изменить"
              @click="startEditCategory(c)"
            >
              Изменить
            </button>
            <button
              type="button"
              class="btn-delete"
              :disabled="deleteId === c.id"
              title="Удалить"
              @click="confirmDeleteCategory(c)"
            >
              {{ deleteId === c.id ? '…' : 'Удалить' }}
            </button>
          </template>
        </div>
      </article>
      <p v-if="!loading && !categories.length" class="empty">
        Категорий пока нет.
      </p>
    </div>

    <section v-if="isAdmin" class="create-block create-block-top">
      <h3>Новая категория</h3>
      <form class="form" @submit.prevent="createCategory">
        <label class="field">
          <span>Название</span>
          <input
            ref="nameInputRef"
            v-model="newName"
            type="text"
            required
            placeholder="Например: Молочные продукты"
          />
        </label>
        <button type="submit" class="primary" :disabled="createLoading">
          {{ createLoading ? 'Сохранение...' : 'Сохранить' }}
        </button>
      </form>
    </section>

    <p v-if="error" class="error">
      {{ error }}
    </p>
  </section>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { api, type Category } from '../api/http'

const categories = ref<Category[]>([])
const loading = ref(false)
const error = ref<string | null>(null)
const isAdmin = ref(false)
const newName = ref('')
const createLoading = ref(false)
const nameInputRef = ref<HTMLInputElement | null>(null)
const deleteId = ref<number | null>(null)
const editingId = ref<number | null>(null)
const editName = ref('')
const updateLoading = ref(false)

const loadCategories = async () => {
  loading.value = true
  error.value = null
  try {
    const { data } = await api.get<Category[]>('/v1/categories')
    categories.value = data
  } catch (e) {
    error.value = 'Ошибка загрузки категорий'
  } finally {
    loading.value = false
  }
}

const createCategory = async () => {
  if (!newName.value.trim()) return
  createLoading.value = true
  error.value = null
  try {
    const { data } = await api.post<Category>('/v1/categories', { name: newName.value.trim() })
    categories.value = [...categories.value, data]
    newName.value = ''
    nameInputRef.value?.focus()
  } catch (e: unknown) {
    const msg = (e as { response?: { data?: { error?: string } } })?.response?.data?.error
    error.value = msg || 'Не удалось создать категорию'
  } finally {
    createLoading.value = false
  }
}

const openCreate = () => {
  nameInputRef.value?.focus()
}

function startEditCategory(c: Category) {
  editingId.value = c.id
  editName.value = c.name
}

function cancelEdit() {
  editingId.value = null
}

async function saveEditCategory() {
  if (editingId.value == null || !editName.value.trim()) return
  updateLoading.value = true
  error.value = null
  try {
    const { data } = await api.put<Category>(`/v1/categories/${editingId.value}`, {
      id: editingId.value,
      name: editName.value.trim(),
    })
    const idx = categories.value.findIndex((x) => x.id === data.id)
    if (idx !== -1) categories.value[idx] = data
    editingId.value = null
  } catch (e: unknown) {
    const msg = (e as { response?: { data?: { error?: string } } })?.response?.data?.error
    error.value = msg || 'Не удалось изменить категорию'
  } finally {
    updateLoading.value = false
  }
}

const confirmDeleteCategory = async (c: Category) => {
  if (!confirm(`Удалить категорию «${c.name}»?`)) return
  deleteId.value = c.id
  error.value = null
  try {
    await api.delete(`/v1/categories/${c.id}`)
    categories.value = categories.value.filter((x) => x.id !== c.id)
  } catch (e: unknown) {
    const msg = (e as { response?: { data?: { error?: string } } })?.response?.data?.error
    error.value = msg || 'Не удалось удалить категорию'
  } finally {
    deleteId.value = null
  }
}

onMounted(() => {
  const role = localStorage.getItem('currentUserRole') || 'user'
  isAdmin.value = role === 'admin'
  void loadCategories()
})
</script>

<style scoped>
.card {
  background: #ffffff;
  border-radius: 8px;
  padding: 16px 18px;
  border: 1px solid #e5e7eb;
}

.header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  margin-bottom: 10px;
}

.header-actions {
  display: flex;
  align-items: center;
  gap: 8px;
}

h2 {
  margin: 0 0 4px;
  font-size: 18px;
}

.muted {
  margin: 0;
  font-size: 13px;
  color: #6b7280;
}

.primary {
  border-radius: 4px;
  padding: 6px 12px;
  border: 1px solid #d1d5db;
  font-size: 13px;
  cursor: pointer;
  background: #111827;
  color: #f9fafb;
  font-weight: 500;
}

.secondary {
  border-radius: 4px;
  padding: 6px 12px;
  border: 1px solid #d1d5db;
  font-size: 13px;
  cursor: pointer;
  background: #ffffff;
  color: #111827;
}

.primary:disabled {
  opacity: 0.6;
  cursor: default;
}

.tiles-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(220px, 1fr));
  gap: 12px;
}

.tile {
  border: 1px solid #e5e7eb;
  border-radius: 8px;
  padding: 10px 12px;
  background: #f9fafb;
}

.tile-meta {
  margin: 0 0 6px;
  font-size: 11px;
  color: #6b7280;
}

.tile-title {
  margin: 0;
  font-size: 30px;
  font-weight: 500;
}

.tile-actions {
  margin-top: 10px;
  display: flex;
  gap: 6px;
  flex-wrap: wrap;
}

.empty {
  color: #9ca3af;
}

.col-actions {
  white-space: nowrap;
}

.btn-delete {
  padding: 4px 8px;
  font-size: 12px;
  border-radius: 4px;
  border: 1px solid #dc2626;
  background: #fff;
  color: #dc2626;
  cursor: pointer;
}

.btn-delete:hover:not(:disabled) {
  background: #fef2f2;
}

.btn-delete:disabled {
  opacity: 0.6;
  cursor: default;
}

.btn-edit {
  padding: 4px 8px;
  font-size: 12px;
  border-radius: 4px;
  border: 1px solid #6b7280;
  background: #fff;
  color: #374151;
  cursor: pointer;
  margin-right: 6px;
}

.btn-edit:hover:not(:disabled) {
  background: #f9fafb;
}

.btn-save {
  padding: 4px 8px;
  font-size: 12px;
  border-radius: 4px;
  border: 1px solid #059669;
  background: #059669;
  color: #fff;
  cursor: pointer;
  margin-right: 6px;
}

.btn-save:disabled {
  opacity: 0.6;
  cursor: default;
}

.small {
  padding: 4px 8px;
  font-size: 12px;
}

.inline-edit-input {
  padding: 4px 8px;
  border: 1px solid #d1d5db;
  border-radius: 4px;
  font-size: 13px;
  width: 100%;
  max-width: 240px;
}

.error {
  margin-top: 8px;
  font-size: 13px;
  color: #fca5a5;
}

.create-block {
  padding: 12px 0;
  border-bottom: 1px solid #e5e7eb;
}

.create-block-top {
  margin-bottom: 14px;
}

.create-block h3 {
  margin: 0 0 12px;
  font-size: 15px;
}

.form {
  display: flex;
  flex-direction: column;
  gap: 10px;
  max-width: 320px;
}

.form .field {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.form .field span {
  font-size: 12px;
  color: #6b7280;
}

.form input {
  padding: 6px 10px;
  border: 1px solid #d1d5db;
  border-radius: 4px;
  font-size: 13px;
}
</style>

