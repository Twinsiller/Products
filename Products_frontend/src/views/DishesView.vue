<template>
  <section class="card">
    <header class="header">
      <div>
        <h2>Блюда</h2>
        <p class="muted">Админ задаёт блюда и товары, из которых они состоят.</p>
      </div>
      <button class="primary" type="button" @click="loadAll" :disabled="loading">
        {{ loading ? 'Загрузка...' : 'Обновить' }}
      </button>
    </header>

    <div v-if="!isAdmin" class="guard">
      <p class="muted">Только администратор может создавать и изменять блюда.</p>
    </div>

    <div v-else class="admin-content">
      <form class="new-dish" @submit.prevent="createDish">
        <h3>Новое блюдо</h3>
        <div class="row">
          <input v-model="newDishName" type="text" placeholder="Название блюда" required />
          <button class="primary" type="submit" :disabled="creatingDish">
            {{ creatingDish ? 'Создание...' : 'Создать' }}
          </button>
        </div>
        <p v-if="createStatusText" class="save-hint">{{ createStatusText }}</p>
      </form>

      <div class="tiles-grid">
        <article v-for="dish in dishes" :key="dish.id" class="tile">
          <header class="tile-header">
            <input
              v-model="editNames[dish.id]"
              type="text"
              class="name-input"
              @input="onDishNameInput(dish.id)"
            />
            <div class="tile-actions">
              <button
                class="secondary"
                type="button"
                :disabled="!isDishDirty(dish.id) || dishSaveState[dish.id] === 'saving'"
                @click="saveDishAll(dish.id)"
              >
                {{ dishSaveState[dish.id] === 'saving' ? 'Сохранение...' : 'Сохранить' }}
              </button>
              <button class="danger" type="button" @click="deleteDish(dish.id)">Удалить</button>
            </div>
          </header>
          <p class="save-hint" :class="saveHintClass(dish.id)">
            {{ saveHintText(dish.id) }}
          </p>

          <div class="requirements">
            <h4>Состав блюда (товары)</h4>
            <p class="inline-note">
              Изменения ниже применяются только после нажатия "Сохранить" у этого блюда.
            </p>
            <ul v-if="currentRequirements(dish.id).length" class="req-list">
              <li v-for="req in currentRequirements(dish.id)" :key="req.id" class="req-item">
                <span>
                  {{ reqProductName(req.product_id) }}: {{ req.quantity }} шт.
                  <span v-if="req.id < 0" class="muted">(новый пункт)</span>
                </span>
                <button class="link danger-link" type="button" @click="deleteRequirement(dish.id, req.id)">
                  Удалить
                </button>
              </li>
            </ul>
            <p v-else class="muted">Пока нет товаров в составе блюда.</p>

            <div class="req-form">
              <select v-model.number="newReq[dish.id].product_id">
                <option :value="0">Выберите товар</option>
                <option v-for="p in products" :key="p.id" :value="p.id">{{ p.name }}</option>
              </select>
              <input v-model.number="newReq[dish.id].quantity" type="number" min="1" placeholder="Кол-во" />
              <button class="secondary" type="button" @click="addRequirement(dish.id)">Добавить товар</button>
            </div>
          </div>
        </article>
      </div>
    </div>

    <p v-if="error" class="error">{{ error }}</p>
  </section>
</template>

<script setup lang="ts">
import { onBeforeUnmount, onMounted, reactive, ref } from 'vue'
import type { AxiosError } from 'axios'
import { api, type Dish, type Product } from '../api/http'

type NewReq = { product_id: number; quantity: number }
type ReqDraft = { id: number; product_id: number; quantity: number }

const dishes = ref<Dish[]>([])
const products = ref<Product[]>([])
const loading = ref(false)
const error = ref<string | null>(null)
const newDishName = ref('')
const creatingDish = ref(false)
const createStatusText = ref('')
const isAdmin = ref(false)
const editNames = reactive<Record<number, string>>({})
const persistedNames = reactive<Record<number, string>>({})
const dishSaveState = reactive<Record<number, 'idle' | 'saving' | 'saved' | 'error'>>({})
const newReq = reactive<Record<number, NewReq>>({})
const persistedReqs = reactive<Record<number, ReqDraft[]>>({})
const draftReqs = reactive<Record<number, ReqDraft[]>>({})
let tempReqId = -1

function ensureReq(dishId: number) {
  if (!newReq[dishId]) {
    newReq[dishId] = { product_id: 0, quantity: 1 }
  }
}

function isDishNameDirty(dishId: number): boolean {
  const current = (editNames[dishId] || '').trim()
  const persisted = (persistedNames[dishId] || '').trim()
  return current !== persisted
}

function normalizeReqs(reqs: ReqDraft[]): string {
  const rows = reqs
    .map((r) => ({
      id: r.id > 0 ? r.id : 0,
      product_id: r.product_id,
      quantity: r.quantity,
    }))
    .sort((a, b) => {
      if (a.id !== b.id) return a.id - b.id
      if (a.product_id !== b.product_id) return a.product_id - b.product_id
      if (a.quantity !== b.quantity) return a.quantity - b.quantity
      return 0
    })
  return JSON.stringify(rows)
}

function isDishReqDirty(dishId: number): boolean {
  const draft = draftReqs[dishId] || []
  const persisted = persistedReqs[dishId] || []
  return normalizeReqs(draft) !== normalizeReqs(persisted)
}

function isDishDirty(dishId: number): boolean {
  return isDishNameDirty(dishId) || isDishReqDirty(dishId)
}

function onDishNameInput(dishId: number) {
  if (dishSaveState[dishId] !== 'saving') {
    dishSaveState[dishId] = 'idle'
  }
}

function saveHintText(dishId: number): string {
  const state = dishSaveState[dishId] || 'idle'
  if (state === 'saving') return 'Сохраняем изменения...'
  if (state === 'saved') return 'Сохранено'
  if (state === 'error') return 'Ошибка сохранения'
  if (isDishDirty(dishId)) return 'Есть несохранённые изменения — нажмите "Сохранить"'
  return 'Изменений нет'
}

function saveHintClass(dishId: number): string {
  const state = dishSaveState[dishId] || 'idle'
  if (state === 'saved') return 'hint-ok'
  if (state === 'error') return 'hint-error'
  if (isDishDirty(dishId)) return 'hint-pending'
  return ''
}

function currentRequirements(dishId: number): ReqDraft[] {
  return draftReqs[dishId] || []
}

function reqProductName(productId: number): string {
  const p = products.value.find((x) => x.id === productId)
  return p?.name || `Товар #${productId}`
}

function syncAuthState() {
  isAdmin.value = localStorage.getItem('currentUserRole') === 'admin'
}

function getApiErrorMessage(e: unknown, fallback: string): string {
  const err = e as AxiosError<{ error?: string }>
  const serverMsg = err.response?.data?.error
  if (typeof serverMsg === 'string' && serverMsg.trim()) {
    return serverMsg
  }
  if (err.response?.status === 401) return 'Требуется вход в систему'
  if (err.response?.status === 403) return 'Доступ только для администратора'
  if (err.response?.status === 404) return 'Маршрут не найден. Перезапустите backend'
  return fallback
}

async function loadDishes() {
  const { data } = await api.get<Dish[]>('/v1/dishes')
  dishes.value = data
  for (const d of data) {
    editNames[d.id] = d.name
    persistedNames[d.id] = d.name
    if (!dishSaveState[d.id] || dishSaveState[d.id] === 'saved') {
      dishSaveState[d.id] = 'idle'
    }
    const productReqs: ReqDraft[] = (d.products || []).map((p) => ({
      id: p.id,
      product_id: p.product_id,
      quantity: p.quantity,
    }))
    persistedReqs[d.id] = productReqs
    draftReqs[d.id] = productReqs.map((x) => ({ ...x }))
    ensureReq(d.id)
  }
}

async function loadProducts() {
  const { data } = await api.get<Product[]>('/v1/products')
  products.value = data
}

async function loadAll() {
  loading.value = true
  error.value = null
  try {
    await Promise.all([loadDishes(), loadProducts()])
  } catch (e) {
    error.value = getApiErrorMessage(e, 'Не удалось загрузить блюда/товары')
  } finally {
    loading.value = false
  }
}

async function createDish() {
  const name = newDishName.value.trim()
  if (!name) return
  error.value = null
  createStatusText.value = ''
  creatingDish.value = true
  try {
    await api.post('/v1/dishes', { name })
    newDishName.value = ''
    await loadDishes()
    createStatusText.value = 'Блюдо успешно создано'
  } catch (e) {
    error.value = getApiErrorMessage(e, 'Не удалось создать блюдо')
    createStatusText.value = 'Не удалось создать блюдо'
  } finally {
    creatingDish.value = false
  }
}

async function saveDishAll(dishId: number) {
  const name = (editNames[dishId] || '').trim()
  if (!name || !isDishDirty(dishId)) return
  error.value = null
  dishSaveState[dishId] = 'saving'
  try {
    if (isDishNameDirty(dishId)) {
      await api.put(`/v1/dishes/${dishId}`, { name })
      persistedNames[dishId] = name
      editNames[dishId] = name
    }

    const persisted = persistedReqs[dishId] || []
    const draft = draftReqs[dishId] || []
    const draftPersistedIds = new Set(draft.filter((x) => x.id > 0).map((x) => x.id))
    const toDelete = persisted.filter((x) => x.id > 0 && !draftPersistedIds.has(x.id))
    const toCreate = draft.filter((x) => x.id <= 0)

    for (const req of toDelete) {
      await api.delete(`/v1/dishes/${dishId}/products/${req.id}`)
    }
    for (const req of toCreate) {
      await api.post(`/v1/dishes/${dishId}/products`, {
        product_id: req.product_id,
        quantity: req.quantity,
      })
    }

    await loadDishes()
    dishSaveState[dishId] = 'saved'
    setTimeout(() => {
      if (dishSaveState[dishId] === 'saved') {
        dishSaveState[dishId] = 'idle'
      }
    }, 1500)
  } catch (e) {
    dishSaveState[dishId] = 'error'
    error.value = getApiErrorMessage(e, 'Не удалось обновить блюдо')
  }
}

async function deleteDish(dishId: number) {
  error.value = null
  try {
    await api.delete(`/v1/dishes/${dishId}`)
    await loadDishes()
  } catch (e) {
    error.value = getApiErrorMessage(e, 'Не удалось удалить блюдо')
  }
}

async function addRequirement(dishId: number) {
  const payload = newReq[dishId]
  if (!payload || !payload.product_id || payload.quantity <= 0) return
  const exists = (draftReqs[dishId] || []).some((x) => x.product_id === payload.product_id)
  if (exists) {
    error.value = 'Такой товар уже добавлен в блюдо'
    return
  }
  if (!draftReqs[dishId]) draftReqs[dishId] = []
  draftReqs[dishId].push({
    id: tempReqId--,
    product_id: payload.product_id,
    quantity: payload.quantity,
  })
  newReq[dishId] = { product_id: 0, quantity: 1 }
  dishSaveState[dishId] = 'idle'
}

async function deleteRequirement(dishId: number, reqId: number) {
  draftReqs[dishId] = (draftReqs[dishId] || []).filter((x) => x.id !== reqId)
  dishSaveState[dishId] = 'idle'
}

onMounted(() => {
  syncAuthState()
  window.addEventListener('auth-changed', syncAuthState)
  void loadAll()
})

onBeforeUnmount(() => {
  window.removeEventListener('auth-changed', syncAuthState)
})
</script>

<style scoped>
.card { background: #fff; border: 1px solid #e5e7eb; border-radius: 8px; padding: 16px; }
.header { display: flex; justify-content: space-between; gap: 12px; margin-bottom: 12px; }
h2 { margin: 0 0 4px; font-size: 18px; }
h3 { margin: 0 0 8px; font-size: 15px; }
h4 { margin: 0 0 6px; font-size: 13px; }
.muted { margin: 0; color: #6b7280; font-size: 13px; }
.error { margin-top: 10px; color: #dc2626; font-size: 13px; }
.primary, .secondary, .danger { border: 1px solid #d1d5db; border-radius: 6px; padding: 6px 10px; font-size: 12px; cursor: pointer; }
.primary { background: #111827; color: #fff; }
.secondary { background: #fff; color: #111827; }
.danger { background: #fff1f2; color: #b91c1c; border-color: #fecdd3; }
.primary:disabled, .secondary:disabled, .danger:disabled { opacity: 0.6; cursor: default; }
.new-dish { border: 1px solid #e5e7eb; border-radius: 8px; padding: 12px; margin-bottom: 12px; }
.row { display: flex; gap: 8px; }
input, select { border: 1px solid #d1d5db; border-radius: 6px; padding: 6px 8px; font-size: 13px; }
.row input { flex: 1; }
.tiles-grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(360px, 1fr)); gap: 10px; }
.tile { border: 1px solid #e5e7eb; border-radius: 8px; padding: 10px; background: #fafafa; }
.tile-header { display: flex; gap: 8px; justify-content: space-between; align-items: center; }
.tile-actions { display: flex; gap: 6px; }
.name-input { font-weight: 600; flex: 1; }
.requirements { margin-top: 10px; }
.inline-note { margin: 0 0 8px; font-size: 12px; color: #6b7280; }
.req-list { margin: 0; padding-left: 18px; }
.req-item { display: flex; justify-content: space-between; gap: 8px; margin-bottom: 4px; font-size: 13px; }
.link { border: none; background: transparent; cursor: pointer; text-decoration: underline; font-size: 12px; }
.danger-link { color: #b91c1c; }
.req-form { margin-top: 8px; display: grid; grid-template-columns: 1.2fr 100px auto; gap: 6px; }
.save-hint { margin: 6px 0 0; font-size: 12px; color: #6b7280; min-height: 16px; }
.hint-pending { color: #92400e; }
.hint-ok { color: #065f46; }
.hint-error { color: #b91c1c; }
</style>
