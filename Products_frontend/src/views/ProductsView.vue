<template>
  <section class="card">
    <header class="header">
      <div>
        <h2>Товары</h2>
        <p class="muted">
          Каталог продуктов: добавляйте в корзину или в избранное.
        </p>
      </div>
      <div class="header-actions">
        <button class="secondary" type="button" @click="loadProducts" :disabled="loading">
          {{ loading ? 'Обновление…' : 'Обновить' }}
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
        placeholder="Поиск по названию или штрихкоду…"
      />
      <select v-model="selectedSort" class="sort-select">
        <option value="name_asc">По названию: А → Я</option>
        <option value="name_desc">По названию: Я → А</option>
        <option value="price_asc">По цене: дешевле сначала</option>
        <option value="price_desc">По цене: дороже сначала</option>
      </select>
    </div>

    <div class="categories-filter">
      <button
        type="button"
        class="chip"
        :class="{ active: selectedCategoryId === null }"
        @click="selectedCategoryId = null"
      >
        Все категории
      </button>
      <button
        v-for="c in categoriesAlphabetical"
        :key="c.id"
        type="button"
        class="chip"
        :class="{ active: selectedCategoryId === c.id }"
        @click="selectedCategoryId = c.id"
      >
        {{ c.name }}
      </button>
    </div>

    <div class="tiles-grid">
      <article v-for="p in filteredProducts" :key="p.id" class="tile">
        <button
          v-if="currentUserId"
          type="button"
          class="fav-btn"
          :class="{ active: isFavourite(p.id) }"
          :title="isFavourite(p.id) ? 'Убрать из избранного' : 'В избранное'"
          @click="toggleFavourite(p)"
        >
          <span class="heart">♥</span>
          <span>{{ isFavourite(p.id) ? 'В избранном' : 'В избранное' }}</span>
        </button>
        <div class="product-photo tile-photo">
          <img
            v-if="!photoErrorIds.has(p.id)"
            :src="productImageUrl(p.id) + (imageVersion[p.id] ? '?v=' + imageVersion[p.id] : '')"
            :alt="p.name"
            @error="addPhotoError(p.id)"
          />
          <span v-else class="photo-placeholder">нет фото</span>
        </div>
        <p v-if="isAdmin" class="tile-meta">№{{ p.id }} · {{ p.barcode || 'без штрихкода' }}</p>
        <h3 class="tile-title">{{ p.name }}</h3>
        <p class="tile-sub price-line">
          {{ p.default_price.toFixed(2) }} ₽
        </p>
        <p class="tile-sub muted-sub">
          {{ categoryName(p.category_id) }}
        </p>
        <p class="tile-sub kbzh-cell">
          {{ formatKbzh(p) }}
        </p>
        <div v-if="currentUserId" class="cart-actions">
          <input
            v-model.number="cartQuantities[p.id]"
            type="number"
            min="1"
            class="qty-input"
          />
          <button type="button" class="primary small" @click="addToCart(p)">
            В корзину
          </button>
        </div>
        <div v-if="isAdmin" class="tile-actions">
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
        </div>
      </article>

      <p v-if="!loading && !filteredProducts.length" class="empty">
        По вашему запросу ничего не нашлось. Попробуйте изменить поиск или сбросить фильтр категории.
      </p>
    </div>

    <p v-if="error" class="error">
      {{ error }}
    </p>
    <p v-if="notice" class="notice">
      {{ notice }}
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
              <option :value="null" disabled>Выберите категорию</option>
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
          <p class="field-hint">Пищевая ценность на одну единицу товара</p>
          <label class="field">
            <span>Ккал</span>
            <input v-model.number="editProduct.calories_kcal" type="number" step="0.01" min="0" />
          </label>
          <label class="field">
            <span>Белки, г</span>
            <input v-model.number="editProduct.protein_g" type="number" step="0.01" min="0" />
          </label>
          <label class="field">
            <span>Жиры, г</span>
            <input v-model.number="editProduct.fat_g" type="number" step="0.01" min="0" />
          </label>
          <label class="field">
            <span>Углеводы, г</span>
            <input v-model.number="editProduct.carbs_g" type="number" step="0.01" min="0" />
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
        <p class="field-hint">Пищевая ценность на одну единицу товара</p>
        <label class="field">
          <span>Ккал</span>
          <input v-model="newCaloriesKcal" type="number" step="0.01" min="0" placeholder="0" />
        </label>
        <label class="field">
          <span>Белки, г</span>
          <input v-model="newProteinG" type="number" step="0.01" min="0" placeholder="0" />
        </label>
        <label class="field">
          <span>Жиры, г</span>
          <input v-model="newFatG" type="number" step="0.01" min="0" placeholder="0" />
        </label>
        <label class="field">
          <span>Углеводы, г</span>
          <input v-model="newCarbsG" type="number" step="0.01" min="0" placeholder="0" />
        </label>
        <label class="field">
          <span>Категория</span>
          <div class="field-row">
            <select v-model="newCategoryId">
              <option :value="null" disabled>Выберите категорию</option>
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
const notice = ref<string | null>(null)
const search = ref('')
const selectedSort = ref<'name_asc' | 'name_desc' | 'price_asc' | 'price_desc'>('name_asc')
const selectedCategoryId = ref<number | null>(null)
const isAdmin = ref(false)
const currentUserId = ref<number | null>(null)
const cartQuantities = ref<Record<number, number>>({})

const newName = ref('')
const newPrice = ref<string>('')
const newBarcode = ref('')
const newCaloriesKcal = ref<string>('')
const newProteinG = ref<string>('')
const newFatG = ref<string>('')
const newCarbsG = ref<string>('')
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

function formatKbzh(p: Product) {
  const k = Number(p.calories_kcal ?? 0)
  const pr = Number(p.protein_g ?? 0)
  const f = Number(p.fat_g ?? 0)
  const c = Number(p.carbs_g ?? 0)
  if (k === 0 && pr === 0 && f === 0 && c === 0) return 'КБЖУ не указано'
  return `${k.toFixed(0)} ккал · Б ${pr.toFixed(1)} · Ж ${f.toFixed(1)} · У ${c.toFixed(1)}`
}

function categoryName(categoryId?: number | null): string {
  if (!categoryId) return 'без категории'
  return categories.value.find((c) => c.id === categoryId)?.name || 'без категории'
}

function normalizeQty(v: number | undefined): number {
  if (!v || Number.isNaN(v)) return 1
  return Math.max(1, Math.floor(v))
}

function addToCart(p: Product) {
  if (!currentUserId.value) {
    error.value = 'Войдите, чтобы добавлять товары в корзину'
    return
  }
  const qty = normalizeQty(cartQuantities.value[p.id])
  cartQuantities.value[p.id] = qty
  const raw = localStorage.getItem('cartItems')
  const items: Array<{ product_id: number; name: string; price_per_unit: number; quantity: number }> =
    raw ? JSON.parse(raw) : []
  const idx = items.findIndex((x) => x.product_id === p.id)
  if (idx >= 0) {
    items[idx].quantity += qty
  } else {
    items.push({
      product_id: p.id,
      name: p.name,
      price_per_unit: p.default_price,
      quantity: qty,
    })
  }
  localStorage.setItem('cartItems', JSON.stringify(items))
  notice.value = `Добавлено в корзину: ${p.name} × ${qty}`
  error.value = null
  window.dispatchEvent(new Event('cart-changed'))
}

const favouriteProductIds = ref<Set<number>>(new Set())

function isFavourite(productId: number): boolean {
  return favouriteProductIds.value.has(productId)
}

async function loadFavouriteProductIds() {
  if (!currentUserId.value) return
  try {
    const { data } = await api.get<Array<{ product_id: number }>>('/v1/favourites/products')
    favouriteProductIds.value = new Set((data || []).map((x) => Number(x.product_id)).filter((v) => Number.isFinite(v)))
  } catch {
    // silently ignore, favourites are optional for products page
  }
}

async function addToFavourites(p: Product): Promise<boolean> {
  if (!currentUserId.value) {
    error.value = 'Войдите, чтобы добавлять товары в избранное'
    return false
  }
  error.value = null
  try {
    await api.post('/v1/favourites/product', { user_id: currentUserId.value, product_id: p.id })
    notice.value = `Добавлено в избранное: ${p.name}`
    return true
  } catch (e: unknown) {
    const resp = (e as { response?: { status?: number; data?: { error?: string } } })?.response
    const msg = resp?.data?.error
    if (resp?.status === 401) {
      error.value = 'Сессия истекла. Войдите заново.'
    } else if (resp?.status === 404) {
      error.value = 'Маршрут избранного не найден. Перезапустите backend.'
    } else {
      error.value = msg || 'Не удалось добавить товар в избранное'
    }
    return false
  }
}

async function removeFromFavourites(p: Product): Promise<boolean> {
  if (!currentUserId.value) return false
  error.value = null
  try {
    await api.delete(`/v1/favourites/product/${p.id}`)
    const next = new Set(favouriteProductIds.value)
    next.delete(p.id)
    favouriteProductIds.value = next
    notice.value = `Убрано из избранного: ${p.name}`
    return true
  } catch (e: unknown) {
    const msg = (e as { response?: { data?: { error?: string } } })?.response?.data?.error
    error.value = msg || 'Не удалось удалить товар из избранного'
    return false
  }
}

async function toggleFavourite(p: Product) {
  if (isFavourite(p.id)) {
    await removeFromFavourites(p)
    return
  }
  const ok = await addToFavourites(p)
  if (!ok) return
  const next = new Set(favouriteProductIds.value)
  next.add(p.id)
  favouriteProductIds.value = next
}

function startEditProduct(p: Product) {
  editProduct.value = {
    id: p.id,
    name: p.name,
    default_price: p.default_price,
    barcode: p.barcode ?? null,
    category_id: p.category_id ?? null,
    manufacturer_id: p.manufacturer_id ?? null,
    calories_kcal: p.calories_kcal ?? 0,
    protein_g: p.protein_g ?? 0,
    fat_g: p.fat_g ?? 0,
    carbs_g: p.carbs_g ?? 0,
  }
}

function cancelEditProduct() {
  editProduct.value = null
}

async function saveEditProduct() {
  if (!editProduct.value) return
  if (!editProduct.value.category_id) {
    error.value = 'Укажите категорию товара'
    return
  }
  const payload = {
    id: editProduct.value.id,
    name: editProduct.value.name.trim(),
    default_price: Number(editProduct.value.default_price),
    barcode: editProduct.value.barcode?.trim() || null,
    category_id: editProduct.value.category_id,
    manufacturer_id: editProduct.value.manufacturer_id,
    calories_kcal: Number(editProduct.value.calories_kcal ?? 0),
    protein_g: Number(editProduct.value.protein_g ?? 0),
    fat_g: Number(editProduct.value.fat_g ?? 0),
    carbs_g: Number(editProduct.value.carbs_g ?? 0),
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

const categoriesAlphabetical = computed(() => {
  return [...categories.value].sort((a, b) => a.name.localeCompare(b.name, 'ru'))
})

const filteredProducts = computed(() => {
  const term = search.value.trim().toLowerCase()
  let list = products.value.filter((p) => {
    const matchesSearch =
      !term ||
      p.name.toLowerCase().includes(term) ||
      (p.barcode && p.barcode.toLowerCase().includes(term))
    const matchesCategory =
      selectedCategoryId.value == null || p.category_id === selectedCategoryId.value
    return matchesSearch && matchesCategory
  })

  list = [...list].sort((a, b) => {
    switch (selectedSort.value) {
      case 'name_desc':
        return b.name.localeCompare(a.name, 'ru')
      case 'price_asc':
        return a.default_price - b.default_price
      case 'price_desc':
        return b.default_price - a.default_price
      case 'name_asc':
      default:
        return a.name.localeCompare(b.name, 'ru')
    }
  })
  return list
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
  if (!newCategoryId.value) {
    error.value = 'Выберите категорию для товара'
    return
  }
  createLoading.value = true
  error.value = null
  try {
    const payload: CreateProductDto = {
      name,
      default_price: price,
      category_id: newCategoryId.value,
      manufacturer_id: newManufacturerId.value ?? undefined,
      barcode: newBarcode.value.trim() || undefined,
      calories_kcal: parseFloat(newCaloriesKcal.value) || 0,
      protein_g: parseFloat(newProteinG.value) || 0,
      fat_g: parseFloat(newFatG.value) || 0,
      carbs_g: parseFloat(newCarbsG.value) || 0,
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
    newCaloriesKcal.value = ''
    newProteinG.value = ''
    newFatG.value = ''
    newCarbsG.value = ''
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
  const storedId = localStorage.getItem('currentUserId')
  currentUserId.value = storedId ? Number(storedId) : null
  const role = localStorage.getItem('currentUserRole') || 'user'
  isAdmin.value = role === 'admin'
  void loadProducts()
  void loadCategories()
  if (currentUserId.value) {
    void loadFavouriteProductIds()
  }
  if (isAdmin.value) {
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
  display: grid;
  grid-template-columns: 1fr 280px;
  gap: 8px;
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

.sort-select {
  border-radius: 999px;
  border: 1px solid #d1d5db;
  padding: 7px 12px;
  font-size: 13px;
  background: #ffffff;
  color: #111827;
}

.categories-filter {
  display: flex;
  gap: 8px;
  flex-wrap: wrap;
  margin-bottom: 12px;
}

.chip {
  border: 1px solid #d1d5db;
  background: #ffffff;
  color: #111827;
  border-radius: 999px;
  padding: 6px 12px;
  font-size: 12px;
  cursor: pointer;
}

.chip.active {
  background: #111827;
  color: #f9fafb;
  border-color: #111827;
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
  position: relative;
}

.fav-btn {
  position: absolute;
  top: 8px;
  right: 8px;
  border: 1px solid #d1d5db;
  border-radius: 999px;
  padding: 3px 8px;
  font-size: 11px;
  display: inline-flex;
  align-items: center;
  gap: 4px;
  cursor: pointer;
  background: #374151;
  color: #f9fafb;
}

.fav-btn .heart {
  color: #ffffff;
  font-size: 12px;
  line-height: 1;
}

.fav-btn.active {
  border-color: #ef4444;
  background: #fff1f2;
  color: #111827;
}

.fav-btn.active .heart {
  color: #ef4444;
}

.tile-meta {
  margin: 6px 0 2px;
  font-size: 11px;
  color: #6b7280;
}

.tile-title {
  margin: 0;
  font-size: 30px;
  font-weight: 500;
}

.tile-sub {
  margin: 4px 0 0;
  font-size: 13px;
  color: #374151;
}

.empty {
  color: #9ca3af;
}

.tile-photo {
  margin-bottom: 4px;
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

.tile-actions {
  margin-top: 10px;
  display: flex;
  gap: 6px;
  flex-wrap: wrap;
}

.cart-actions {
  margin-top: 8px;
  display: flex;
  gap: 6px;
  align-items: center;
}

.qty-input {
  width: 64px;
  padding: 5px 8px;
  border: 1px solid #d1d5db;
  border-radius: 6px;
  font-size: 12px;
  background: #fff;
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

.notice {
  margin-top: 8px;
  font-size: 13px;
  color: #065f46;
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

.field-hint {
  margin: 0;
  font-size: 12px;
  color: #6b7280;
}

.col-kbzh {
  font-size: 11px;
  white-space: nowrap;
}

.kbzh-cell {
  color: #9ca3af;
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

