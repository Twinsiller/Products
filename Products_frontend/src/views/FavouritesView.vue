<template>
  <section class="card">
    <h2>Избранное</h2>
    <p class="muted">Ваши избранные товары.</p>

    <div class="actions">
      <button class="secondary" type="button" @click="loadFavourites" :disabled="loading || !currentUserId">
        {{ loading ? 'Загрузка...' : 'Обновить' }}
      </button>
    </div>

    <div v-if="!currentUserId" class="muted">Войдите в аккаунт, чтобы видеть избранное.</div>

    <div v-else-if="items.length" class="list">
      <article v-for="it in items" :key="it.id" class="item">
        <div>
          <strong>{{ it.product?.name || ('Товар #' + it.product_id) }}</strong>
          <p class="muted small">ID товара: {{ it.product_id }}</p>
        </div>
        <button class="danger" type="button" @click="removeFavourite(it.product_id)" :disabled="deletingProductId === it.product_id">
          {{ deletingProductId === it.product_id ? 'Удаление...' : 'Убрать' }}
        </button>
      </article>
    </div>
    <p v-else-if="!loading" class="muted">Пока нет избранных товаров.</p>

    <p v-if="error" class="error">{{ error }}</p>
  </section>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { api, type Product } from '../api/http'

type FavouriteProduct = {
  id: number
  user_id: number
  product_id: number
  product?: Product
}

const currentUserId = ref<number | null>(null)
const loading = ref(false)
const error = ref<string | null>(null)
const items = ref<FavouriteProduct[]>([])
const deletingProductId = ref<number | null>(null)

async function loadFavourites() {
  if (!currentUserId.value) return
  loading.value = true
  error.value = null
  try {
    const { data } = await api.get<FavouriteProduct[]>('/v1/favourites/products')
    items.value = data
  } catch (e: unknown) {
    const resp = (e as { response?: { status?: number; data?: { error?: string } } })?.response
    const msg = resp?.data?.error
    if (resp?.status === 401) {
      error.value = 'Сессия истекла. Войдите заново.'
    } else if (resp?.status === 404) {
      error.value = 'Маршрут избранного не найден. Перезапустите backend.'
    } else {
      error.value = msg || 'Не удалось загрузить избранное'
    }
    items.value = []
  } finally {
    loading.value = false
  }
}

async function removeFavourite(productId: number) {
  deletingProductId.value = productId
  error.value = null
  try {
    await api.delete(`/v1/favourites/product/${productId}`)
    items.value = items.value.filter((x) => x.product_id !== productId)
  } catch (e: unknown) {
    const msg = (e as { response?: { data?: { error?: string } } })?.response?.data?.error
    error.value = msg || 'Не удалось удалить из избранного'
  } finally {
    deletingProductId.value = null
  }
}

onMounted(() => {
  const raw = localStorage.getItem('currentUserId')
  currentUserId.value = raw ? Number(raw) : null
  if (currentUserId.value) {
    void loadFavourites()
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

h2 {
  margin: 0 0 6px;
  font-size: 18px;
}

.muted {
  margin: 0;
  font-size: 13px;
  color: #6b7280;
}

.small {
  font-size: 12px;
}

.actions {
  margin: 10px 0;
}

.list {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.item {
  border: 1px solid #e5e7eb;
  border-radius: 8px;
  padding: 10px 12px;
  background: #fafafa;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
}

.secondary,
.danger {
  border: 1px solid #d1d5db;
  border-radius: 6px;
  padding: 6px 10px;
  font-size: 12px;
  cursor: pointer;
}

.secondary {
  background: #fff;
  color: #111827;
}

.danger {
  background: #fff1f2;
  color: #b91c1c;
  border-color: #fecdd3;
}

.error {
  margin-top: 10px;
  color: #b91c1c;
  font-size: 13px;
}
</style>
