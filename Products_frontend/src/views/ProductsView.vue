<template>
  <section class="card">
    <header class="header">
      <div>
        <h2>Товары</h2>
        <p class="muted">
          Справочник товарных позиций, доступных для заказов и рекомендаций.
        </p>
      </div>
      <div class="header-actions">
        <button class="secondary" type="button" @click="loadProducts" :disabled="loading">
          {{ loading ? 'Обновление...' : 'Обновить список' }}
        </button>
        <button
          v-if="isAdmin"
          class="primary"
          type="button"
          @click="openCreate"
        >
          Добавить товар
        </button>
      </div>
    </header>

    <div class="toolbar">
      <input
        v-model="search"
        type="text"
        class="search"
        placeholder="Поиск по названию или штрихкоду..."
      />
    </div>

    <table class="table">
      <thead>
        <tr>
          <th class="col-photo">Фото</th>
          <th>ID</th>
          <th>Название</th>
          <th>Цена</th>
          <th>Штрихкод</th>
          <th v-if="isAdmin" class="col-actions">Действия</th>
        </tr>
      </thead>
      <tbody>
        <tr v-for="p in filteredProducts" :key="p.id">
          <td class="col-photo">
            <div class="product-photo">
              <img
                v-if="!photoErrorIds.has(p.id)"
                :src="productImageUrl(p.id) + (imageVersion[p.id] ? '?v=' + imageVersion[p.id] : '')"
                :alt="p.name"
                @error="addPhotoError(p.id)"
              />
              <span v-else class="photo-placeholder">нет фото</span>
            </div>
          </td>
          <td>{{ p.id }}</td>
          <td>{{ p.name }}</td>
          <td>{{ p.default_price.toFixed(2) }}</td>
          <td>{{ p.barcode || '—' }}</td>
          <td v-if="isAdmin" class="col-actions">
            <button
              type="button"
              class="btn-edit"
              :disabled="deleteId === p.id"
              title="Изменить"
              @click="startEditProduct(p)"
            >
              Изменить
            </button>
            <label class="btn-upload">
              <input
                type="file"
                accept="image/*"
                class="hidden-input"
                @change="(e: Event) => uploadProductImage(p.id, e)"
              />
              {{ uploadId === p.id ? '…' : 'Фото' }}
            </label>
            <button
              type="button"
              class="btn-delete"
              :disabled="deleteId === p.id"
              title="Удалить"
              @click="confirmDeleteProduct(p)"
            >
              {{ deleteId === p.id ? '…' : 'Удалить' }}
            </button>
          </td>
        </tr>
        <tr v-if="!loading && !filteredProducts.length">
          <td :colspan="isAdmin ? 6 : 5" class="empty">
            Нет товаров по заданным условиям.
          </td>
        </tr>
      </tbody>
    </table>

    <p v-if="error" class="error">
      {{ error }}
    </p>

    <div v-if="isAdmin" class="forms-row">
      <section v-if="editProduct" class="create-block edit-block">
        <h3>Изменение выбранного товара</h3>
        <form class="form" @submit.prevent="saveEditProduct">
          <label class="field">
            <span>Название</span>
            <input v-model="editProduct.name" type="text" required />
          </label>
          <label class="field">
            <span>Цена</span>
            <input v-model.number="editProduct.default_price" type="number" step="0.01" min="0" required />
          </label>
          <label class="field">
            <span>Штрихкод</span>
            <input v-model="editProduct.barcode" type="text" placeholder="—" />
          </label>
          <label class="field">
            <span>Категория</span>
            <select v-model="editProduct.category_id">
              <option :value="null">— не выбрано</option>
              <option v-for="c in categories" :key="c.id" :value="c.id">{{ c.name }}</option>
            </select>
          </label>
          <label class="field">
            <span>Производитель</span>
            <select v-model="editProduct.manufacturer_id">
              <option :value="null">— не выбрано</option>
              <option v-for="m in manufacturers" :key="m.id" :value="m.id">{{ m.name }}</option>
            </select>
          </label>
          <div class="form-actions">
            <button type="submit" class="primary" :disabled="updateLoading">
              {{ updateLoading ? 'Сохранение...' : 'Сохранить' }}
            </button>
            <button type="button" class="secondary" @click="cancelEditProduct">Отмена</button>
          </div>
        </form>
      </section>

      <section class="create-block">
        <h3>Новый товар</h3>
      <form class="form" @submit.prevent="createProduct">
        <label class="field">
          <span>Название</span>
          <input
            ref="nameInputRef"
            v-model="newName"
            type="text"
            required
            placeholder="Например: Молоко 3.2%"
          />
        </label>
        <label class="field">
          <span>Цена (обязательно)</span>
          <input
            v-model="newPrice"
            type="number"
            step="0.01"
            min="0"
            required
            placeholder="0.00"
          />
        </label>
        <label class="field">
          <span>Штрихкод (необязательно)</span>
          <input
            v-model="newBarcode"
            type="text"
            placeholder="Например: 4601234567890"
          />
        </label>
        <label class="field">
          <span>Категория</span>
          <div class="field-row">
            <select v-model="newCategoryId">
              <option :value="null">— не выбрано</option>
              <option
                v-for="c in categories"
                :key="c.id"
                :value="c.id"
              >
                {{ c.name }}
              </option>
            </select>
            <button
              type="button"
              class="secondary small"
              @click="showCategoryCreate = !showCategoryCreate"
            >
              {{ showCategoryCreate ? 'Отмена' : 'Создать категорию' }}
            </button>
          </div>
          <div v-if="showCategoryCreate" class="inline-create">
            <input
              v-model="newCategoryName"
              type="text"
              placeholder="Название категории"
              @keydown.enter.prevent="addCategory"
            />
            <button
              type="button"
              class="primary small"
              :disabled="createCategoryLoading || !newCategoryName.trim()"
              @click="addCategory"
            >
              {{ createCategoryLoading ? '...' : 'Добавить' }}
            </button>
          </div>
        </label>
        <label class="field">
          <span>Производитель</span>
          <div class="field-row">
            <select v-model="newManufacturerId">
              <option :value="null">— не выбрано</option>
              <option
                v-for="m in manufacturers"
                :key="m.id"
                :value="m.id"
              >
                {{ m.name }}
              </option>
            </select>
            <button
              type="button"
              class="secondary small"
              @click="showManufacturerCreate = !showManufacturerCreate"
            >
              {{ showManufacturerCreate ? 'Отмена' : 'Создать производителя' }}
            </button>
          </div>
          <div v-if="showManufacturerCreate" class="inline-create">
            <input
              v-model="newManufacturerName"
              type="text"
              placeholder="Название производителя"
              @keydown.enter.prevent="addManufacturer"
            />
            <button
              type="button"
              class="primary small"
              :disabled="createManufacturerLoading || !newManufacturerName.trim()"
              @click="addManufacturer"
            >
              {{ createManufacturerLoading ? '...' : 'Добавить' }}
            </button>
          </div>
        </label>
        <label class="field">
          <span>Фото (необязательно)</span>
          <input
            ref="imageInputRef"
            type="file"
            accept="image/*"
            @change="onImageSelect"
          />
          <span v-if="newImageFile" class="file-name">{{ newImageFile.name }}</span>
        </label>
        <button type="submit" class="primary" :disabled="createLoading">
          {{ createLoading ? 'Сохранение...' : 'Сохранить' }}
        </button>
      </form>
      </section>
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import {
  api,
  productImageUrl,
  type Product,
  type Category,
  type Manufacturer,
  type CreateProductDto,
} from '../api/http'

const products = ref<Product[]>([])
const categories = ref<Category[]>([])
const manufacturers = ref<Manufacturer[]>([])
const loading = ref(false)
const error = ref<string | null>(null)
const search = ref('')
const isAdmin = ref(false)

const newName = ref('')
const newPrice = ref<string>('')
const newBarcode = ref('')
const newCategoryId = ref<number | null>(null)
const newManufacturerId = ref<number | null>(null)
const createLoading = ref(false)
const nameInputRef = ref<HTMLInputElement | null>(null)

const showCategoryCreate = ref(false)
const showManufacturerCreate = ref(false)
const newCategoryName = ref('')
const newManufacturerName = ref('')
const createCategoryLoading = ref(false)
const createManufacturerLoading = ref(false)
const deleteId = ref<number | null>(null)
const uploadId = ref<number | null>(null)
const photoErrorIds = ref<Set<number>>(new Set())
const editProduct = ref<Product | null>(null)
const updateLoading = ref(false)
const newImageFile = ref<File | null>(null)
const imageInputRef = ref<HTMLInputElement | null>(null)
const imageVersion = ref<Record<number, number>>({})

function addPhotoError(id: number) {
  photoErrorIds.value = new Set(photoErrorIds.value).add(id)
}

function startEditProduct(p: Product) {
  editProduct.value = {
    id: p.id,
    name: p.name,
    default_price: p.default_price,
    barcode: p.barcode ?? null,
    category_id: p.category_id ?? null,
    manufacturer_id: p.manufacturer_id ?? null,
  }
}

function cancelEditProduct() {
  editProduct.value = null
}

async function saveEditProduct() {
  if (!editProduct.value) return
  const payload = {
    id: editProduct.value.id,
    name: editProduct.value.name.trim(),
    default_price: Number(editProduct.value.default_price),
    barcode: editProduct.value.barcode?.trim() || null,
    category_id: editProduct.value.category_id,
    manufacturer_id: editProduct.value.manufacturer_id,
  }
  if (!payload.name || Number.isNaN(payload.default_price) || payload.default_price < 0) return
  updateLoading.value = true
  error.value = null
  try {
    const { data } = await api.put<Product>(`/v1/products/${editProduct.value.id}`, payload)
    const idx = products.value.findIndex((x) => x.id === data.id)
    if (idx !== -1) products.value[idx] = data
    editProduct.value = null
  } catch (e: unknown) {
    const msg = (e as { response?: { data?: { error?: string } } })?.response?.data?.error
    error.value = msg || 'Не удалось изменить товар'
  } finally {
    updateLoading.value = false
  }
}

function onImageSelect(e: Event) {
  const input = e.target as HTMLInputElement
  newImageFile.value = input.files?.[0] ?? null
}

async function uploadProductImage(productId: number, e: Event) {
  const input = e.target as HTMLInputElement
  const file = input.files?.[0]
  if (!file) return
  uploadId.value = productId
  error.value = null
  try {
    const fd = new FormData()
    fd.append('image', file)
    await api.post(`/v1/products/${productId}/image`, fd, {
      headers: { 'Content-Type': 'multipart/form-data' },
    })
    const next = new Set(photoErrorIds.value)
    next.delete(productId)
    photoErrorIds.value = next
    imageVersion.value = { ...imageVersion.value, [productId]: Date.now() }
  } catch (err: unknown) {
    const msg = (err as { response?: { data?: { error?: string } } })?.response?.data?.error
    error.value = msg || 'Не удалось загрузить фото'
  } finally {
    uploadId.value = null
    input.value = ''
  }
}

const loadProducts = async () => {
  loading.value = true
  error.value = null
  try {
    const { data } = await api.get<Product[]>('/v1/products')
    products.value = data
  } catch (e) {
    error.value = 'Ошибка загрузки списка товаров'
  } finally {
    loading.value = false
  }
}

const loadCategories = async () => {
  try {
    const { data } = await api.get<Category[]>('/v1/categories')
    categories.value = data
  } catch {
    categories.value = []
  }
}

const loadManufacturers = async () => {
  try {
    const { data } = await api.get<Manufacturer[]>('/v1/manufacturers')
    manufacturers.value = data
  } catch {
    manufacturers.value = []
  }
}

const filteredProducts = computed(() => {
  const term = search.value.trim().toLowerCase()
  if (!term) return products.value
  return products.value.filter((p) => {
    return (
      p.name.toLowerCase().includes(term) ||
      (p.barcode && p.barcode.toLowerCase().includes(term))
    )
  })
})

const addCategory = async () => {
  const name = newCategoryName.value.trim()
  if (!name) return
  createCategoryLoading.value = true
  error.value = null
  try {
    const { data } = await api.post<Category>('/v1/categories', { name })
    await loadCategories()
    newCategoryId.value = data.id
    newCategoryName.value = ''
    showCategoryCreate.value = false
  } catch (e: unknown) {
    const msg = (e as { response?: { data?: { error?: string } } })?.response?.data?.error
    error.value = msg || 'Не удалось создать категорию'
  } finally {
    createCategoryLoading.value = false
  }
}

const addManufacturer = async () => {
  const name = newManufacturerName.value.trim()
  if (!name) return
  createManufacturerLoading.value = true
  error.value = null
  try {
    const { data } = await api.post<Manufacturer>('/v1/manufacturers', { name })
    await loadManufacturers()
    newManufacturerId.value = data.id
    newManufacturerName.value = ''
    showManufacturerCreate.value = false
  } catch (e: unknown) {
    const msg = (e as { response?: { data?: { error?: string } } })?.response?.data?.error
    error.value = msg || 'Не удалось создать производителя'
  } finally {
    createManufacturerLoading.value = false
  }
}

const createProduct = async () => {
  const name = newName.value.trim()
  const price = parseFloat(newPrice.value)
  if (!name || Number.isNaN(price) || price < 0) return
  createLoading.value = true
  error.value = null
  try {
    const payload: CreateProductDto = {
      name,
      default_price: price,
      category_id: newCategoryId.value ?? undefined,
      manufacturer_id: newManufacturerId.value ?? undefined,
      barcode: newBarcode.value.trim() || undefined,
    }
    const { data } = await api.post<Product>('/v1/products', payload)
    products.value = [...products.value, data]
    if (newImageFile.value) {
      const fd = new FormData()
      fd.append('image', newImageFile.value)
      try {
        await api.post(`/v1/products/${data.id}/image`, fd, {
          headers: { 'Content-Type': 'multipart/form-data' },
        })
        const next = new Set(photoErrorIds.value)
        next.delete(data.id)
        photoErrorIds.value = next
        imageVersion.value = { ...imageVersion.value, [data.id]: Date.now() }
      } catch {
        // фото не загрузилось — товар уже создан
      }
    }
    newName.value = ''
    newPrice.value = ''
    newBarcode.value = ''
    newCategoryId.value = null
    newManufacturerId.value = null
    newImageFile.value = null
    imageInputRef.value?.value && (imageInputRef.value.value = '')
    nameInputRef.value?.focus()
  } catch (e) {
    error.value = 'Не удалось создать товар'
  } finally {
    createLoading.value = false
  }
}

const openCreate = () => {
  nameInputRef.value?.focus()
}

const confirmDeleteProduct = async (p: Product) => {
  if (!confirm(`Удалить товар «${p.name}»?`)) return
  deleteId.value = p.id
  error.value = null
  try {
    await api.delete(`/v1/products/${p.id}`)
    products.value = products.value.filter((x) => x.id !== p.id)
  } catch (e: unknown) {
    const msg = (e as { response?: { data?: { error?: string } } })?.response?.data?.error
    error.value = msg || 'Не удалось удалить товар'
  } finally {
    deleteId.value = null
  }
}

onMounted(() => {
  const role = localStorage.getItem('currentUserRole') || 'user'
  isAdmin.value = role === 'admin'
  void loadProducts()
  if (isAdmin.value) {
    void loadCategories()
    void loadManufacturers()
  }
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
  margin-bottom: 14px;
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

.toolbar {
  margin-bottom: 10px;
}

.search {
  width: 100%;
  border-radius: 999px;
  border: 1px solid rgba(148, 163, 184, 0.6);
  padding: 7px 12px;
  font-size: 13px;
  background: rgba(15, 23, 42, 0.95);
  color: #e5e7eb;
}

.table {
  width: 100%;
  border-collapse: collapse;
  font-size: 13px;
}

th,
td {
  padding: 6px 8px;
  text-align: left;
}

thead {
  background: rgba(15, 23, 42, 0.9);
}

tbody tr:nth-child(even) {
  background: rgba(15, 23, 42, 0.9);
}

tbody tr:nth-child(odd) {
  background: rgba(15, 23, 42, 0.8);
}

.empty {
  text-align: center;
  color: #9ca3af;
}

.col-photo {
  width: 64px;
  vertical-align: middle;
}

.product-photo {
  width: 48px;
  height: 48px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: #f3f4f6;
  border-radius: 4px;
  overflow: hidden;
}

.product-photo img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.photo-placeholder {
  font-size: 10px;
  color: #9ca3af;
}

.file-name {
  font-size: 12px;
  color: #6b7280;
  margin-left: 8px;
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

.btn-upload {
  display: inline-block;
  padding: 4px 8px;
  font-size: 12px;
  border-radius: 4px;
  border: 1px solid #6b7280;
  background: #fff;
  color: #374151;
  cursor: pointer;
  margin-right: 6px;
}

.btn-upload:hover {
  background: #f9fafb;
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

.forms-row {
  display: flex;
  gap: 24px;
  flex-wrap: wrap;
  margin-top: 20px;
  padding-top: 16px;
  border-top: 1px solid #e5e7eb;
}

.forms-row .create-block {
  flex: 1;
  min-width: 280px;
}

.forms-row .edit-block {
  flex: 1;
  min-width: 280px;
}

.form-actions {
  display: flex;
  gap: 8px;
  margin-top: 4px;
}

.hidden-input {
  position: absolute;
  width: 0;
  height: 0;
  opacity: 0;
}

.error {
  margin-top: 8px;
  font-size: 13px;
  color: #fca5a5;
}

.create-block {
  margin-top: 20px;
  padding-top: 16px;
  border-top: 1px solid #e5e7eb;
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

.form input,
.form select {
  padding: 6px 10px;
  border: 1px solid #d1d5db;
  border-radius: 4px;
  font-size: 13px;
}

.field-row {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
}

.field-row select {
  flex: 1;
  min-width: 140px;
}

.inline-create {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-top: 6px;
}

.inline-create input {
  flex: 1;
  min-width: 120px;
}

.small {
  padding: 4px 10px;
  font-size: 12px;
}
</style>

