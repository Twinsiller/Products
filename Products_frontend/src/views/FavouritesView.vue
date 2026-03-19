<template>
  <section class="card">
    <header class="header">
      <div>
        <h2>Избранные товары</h2>
        <p class="muted">
          Товары, помеченные пользователем как избранные.
        </p>
      </div>
      <button
        class="primary"
        type="button"
        @click="loadFavourites"
        :disabled="loading || !currentUser"
      >
        {{ loading ? 'Загрузка...' : 'Обновить' }}
      </button>
    </header>

    <div v-if="!currentUser" class="guard">
      <p class="warning">
        Избранное доступно только зарегистрированным пользователям. Пожалуйста, войдите или
        зарегистрируйтесь.
      </p>
      <div class="guard-actions">
        <RouterLink to="/" class="primary-link">Войти</RouterLink>
        <RouterLink to="/register" class="secondary-link">Зарегистрироваться</RouterLink>
      </div>
    </div>

    <ul v-else class="list">
      <li v-for="item in favourites" :key="item.product.id" class="item">
        <div class="title">{{ item.product.name }}</div>
        <div class="meta">
          <span v-if="item.product.default_price">
            Цена: {{ item.product.default_price.toFixed(2) }}
          </span>
          <span v-if="item.product.barcode">· Штрихкод: {{ item.product.barcode }}</span>
        </div>
      </li>
      <li v-if="!loading && !favourites.length" class="empty">
        У пользователя пока нет избранных товаров.
      </li>
    </ul>

    <p v-if="error" class="error">
      {{ error }}
    </p>
  </section>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { RouterLink } from 'vue-router'
import { api, type Product } from '../api/http'

interface FavouriteProduct {
  id: number
  user_id: number
  product_id: number
  product: Product
}

const currentUser = ref<{ id: number } | null>(null)
const favourites = ref<FavouriteProduct[]>([])
const loading = ref(false)
const error = ref<string | null>(null)

const loadFavourites = async () => {
  if (!currentUser.value) return
  loading.value = true
  error.value = null
  try {
    const { data } = await api.get<FavouriteProduct[]>(
      `/v1/users/${currentUser.value.id}/favourites/products`,
    )
    favourites.value = data
  } catch (e) {
    error.value = 'Ошибка загрузки избранных товаров'
  } finally {
    loading.value = false
  }
}

onMounted(() => {
  const storedId = localStorage.getItem('currentUserId')
  if (storedId) {
    currentUser.value = { id: Number(storedId) }
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

.header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  margin-bottom: 10px;
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

.primary:disabled {
  opacity: 0.6;
  cursor: default;
}

.warning {
  font-size: 13px;
  color: #fbbf24;
}

.guard-actions {
  margin-top: 8px;
  display: flex;
  gap: 8px;
}

.primary-link,
.secondary-link {
  font-size: 13px;
  text-decoration: none;
  border-radius: 4px;
  padding: 6px 12px;
}

.primary-link {
  background: #111827;
  color: #f9fafb;
}

.secondary-link {
  border: 1px solid #d1d5db;
  color: #111827;
  background: #ffffff;
}

.list {
  list-style: none;
  padding: 0;
  margin: 10px 0 0;
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.item {
  padding: 10px 11px;
  border-radius: 12px;
  background: rgba(15, 23, 42, 0.9);
  border: 1px solid rgba(55, 65, 81, 0.9);
}

.title {
  font-size: 14px;
  font-weight: 500;
}

.meta {
  margin-top: 4px;
  font-size: 12px;
  color: #9ca3af;
}

.empty {
  padding: 6px 0;
  font-size: 13px;
  color: #9ca3af;
}

.error {
  margin-top: 8px;
  font-size: 13px;
  color: #fca5a5;
}
</style>

