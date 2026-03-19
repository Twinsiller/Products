<template>
  <section class="layout">
    <div v-if="!currentUser" class="guard">
      <h2>Рекомендации доступны только после входа</h2>
      <p class="muted">
        Пожалуйста, зарегистрируйтесь или войдите в систему, чтобы увидеть персональные рекомендации.
      </p>
      <div class="guard-actions">
        <RouterLink to="/" class="primary-link">Войти</RouterLink>
        <RouterLink to="/register" class="secondary-link">Зарегистрироваться</RouterLink>
      </div>
    </div>

    <div v-else>
      <header class="header">
        <div>
          <h2>Персональные рекомендации</h2>
          <p class="muted">
            Товары и блюда, подобранные на основе истории предпочтений пользователя.
          </p>
        </div>
        <div class="user-pill">
          <span class="user-name">
            Пользователь #{{ currentUser.id }}
          </span>
          <span v-if="currentUser.name" class="user-extra">{{ currentUser.name }}</span>
          <span v-if="currentUser.role" class="user-extra">· {{ currentUser.role }}</span>
        </div>
      </header>

      <div class="columns">
      <div class="column">
        <h3>Рекомендуемые товары</h3>
        <button
          class="ghost-btn"
          type="button"
          @click="loadProducts"
          :disabled="loadingProducts"
        >
          {{ loadingProducts ? 'Загрузка...' : 'Обновить рекомендации' }}
        </button>
        <ul class="list" v-if="productRecommendations.length">
          <li
            v-for="p in productRecommendations"
            :key="p.id"
            class="item"
          >
            <div>
              <div class="item-title">
                {{ p.name }}
              </div>
              <div class="item-meta">
                <span v-if="p.default_price">
                  Цена: {{ p.default_price.toFixed(2) }}
                </span>
                <span v-if="p.barcode">· Штрихкод: {{ p.barcode }}</span>
              </div>
            </div>
          </li>
        </ul>
        <p v-else class="muted small">
          Нет данных по рекомендациям товаров. Попробуйте оформить заказы или добавить
          товары в избранное для выбранного пользователя.
        </p>
      </div>

      <div class="column">
        <h3>Рекомендуемые блюда</h3>
        <button
          class="ghost-btn"
          type="button"
          @click="loadDishes"
          :disabled="loadingDishes"
        >
          {{ loadingDishes ? 'Загрузка...' : 'Обновить рекомендации' }}
        </button>
        <ul class="list" v-if="dishRecommendations.length">
          <li
            v-for="d in dishRecommendations"
            :key="d.id"
            class="item"
          >
            <div>
              <div class="item-title">
                {{ d.name }}
              </div>
            </div>
          </li>
        </ul>
        <p v-else class="muted small">
          Рекомендации по блюдам пока отсутствуют.
        </p>
      </div>
      </div>

      <p v-if="error" class="error">
        {{ error }}
      </p>
    </div>
  </section>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { api, type Product } from '../api/http'

interface Dish {
  id: number
  name: string
}

const currentUser = ref<{ id: number; name?: string; role?: string } | null>(null)
const productRecommendations = ref<Product[]>([])
const dishRecommendations = ref<Dish[]>([])
const loadingProducts = ref(false)
const loadingDishes = ref(false)
const error = ref<string | null>(null)

const ensureUser = () => {
  const storedId = localStorage.getItem('currentUserId')
  if (!storedId) {
    return
  }
  currentUser.value = {
    id: Number(storedId),
    name: localStorage.getItem('currentUserName') || undefined,
    role: localStorage.getItem('currentUserRole') || undefined,
  }
}

const loadProducts = async () => {
  if (!currentUser.value) return
  loadingProducts.value = true
  error.value = null
  try {
    const { data } = await api.get<Product[]>(
      `/v1/users/${currentUser.value.id}/recommendations/products`,
    )
    productRecommendations.value = data
  } catch (e) {
    error.value = 'Ошибка загрузки рекомендаций по товарам'
  } finally {
    loadingProducts.value = false
  }
}

const loadDishes = async () => {
  if (!currentUser.value) return
  loadingDishes.value = true
  error.value = null
  try {
    const { data } = await api.get<Dish[]>(
      `/v1/users/${currentUser.value.id}/recommendations/dishes`,
    )
    dishRecommendations.value = data
  } catch (e) {
    error.value = 'Ошибка загрузки рекомендаций по блюдам'
  } finally {
    loadingDishes.value = false
  }
}

onMounted(async () => {
  ensureUser()
  await Promise.all([loadProducts(), loadDishes()])
})
</script>

<style scoped>
.layout {
  display: flex;
  flex-direction: column;
  gap: 18px;
}

.guard {
  background: #ffffff;
  border-radius: 8px;
  padding: 16px 18px;
  border: 1px solid #e5e7eb;
}

.guard h2 {
  margin: 0 0 6px;
  font-size: 18px;
}

.guard-actions {
  margin-top: 12px;
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

.header {
  display: flex;
  flex-wrap: wrap;
  gap: 10px 16px;
  align-items: center;
  justify-content: space-between;
}

h2 {
  margin: 0 0 4px;
  font-size: 20px;
}

.muted {
  margin: 0;
  font-size: 13px;
  color: #6b7280;
}

.small {
  font-size: 12px;
}

.user-pill {
  padding: 6px 10px;
  border-radius: 4px;
  border: 1px solid #e5e7eb;
  font-size: 12px;
  display: inline-flex;
  align-items: center;
  gap: 4px;
  background: #f9fafb;
}

.user-name {
  font-weight: 500;
  color: #111827;
}

.user-extra {
  color: #6b7280;
}

.link {
  font-size: 13px;
  color: #1d4ed8;
  text-decoration: none;
}

.columns {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(260px, 1fr));
  gap: 16px;
}

.column {
  background: #ffffff;
  border-radius: 8px;
  padding: 12px 14px;
  border: 1px solid #e5e7eb;
}

h3 {
  margin: 0 0 10px;
  font-size: 16px;
}

.ghost-btn {
  margin-bottom: 10px;
  border-radius: 4px;
  padding: 6px 12px;
  border: 1px solid #d1d5db;
  background: transparent;
  color: #111827;
  font-size: 13px;
  cursor: pointer;
}

.ghost-btn:disabled {
  opacity: 0.6;
  cursor: default;
}

.list {
  list-style: none;
  padding: 0;
  margin: 0;
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

.item-title {
  font-size: 14px;
  font-weight: 500;
}

.item-meta {
  margin-top: 4px;
  font-size: 12px;
  color: #9ca3af;
  display: flex;
  flex-wrap: wrap;
  gap: 4px;
}

.error {
  margin: 4px 0 0;
  font-size: 13px;
  color: #fca5a5;
}
</style>

