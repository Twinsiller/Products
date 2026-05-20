<template>
  <section class="card">
    <div v-if="!currentUserId" class="guard">
      <h2>Рекомендации</h2>
      <p class="muted">Войдите в аккаунт, чтобы получить персональные рекомендации.</p>
    </div>

    <div v-else>
      <header class="header">
        <div>
          <h2>Персональные рекомендации</h2>
          <p class="muted">Подбираем товары и блюда под ваши вкусы и сбалансированный рацион.</p>
        </div>
        <div class="controls">
          <label class="inline-label">
            Сколько показывать:
            <select v-model.number="limit" :disabled="loading">
              <option :value="3">3</option>
              <option :value="5">5</option>
              <option :value="10">10</option>
            </select>
          </label>
          <button class="primary" type="button" :disabled="loading" @click="loadRecommendations">
            {{ loading ? 'Загрузка...' : 'Обновить' }}
          </button>
        </div>
      </header>

      <h3 class="section-title">Что стоит купить</h3>
      <div v-if="productRecommendations.length" class="list">
        <article v-for="(r, idx) in productRecommendations" :key="`${r.product.id}-${idx}`" class="item">
          <div class="item-head">
            <h3>{{ idx + 1 }}. {{ r.product.name }}</h3>
            <span class="price">{{ Number(r.product.default_price || 0).toFixed(2) }} ₽</span>
          </div>

          <div class="why-row">
            <span class="why-chip" v-if="(r.cb_score ?? 0) > 0">подходит по КБЖУ</span>
            <span class="why-chip" v-if="(r.cf_score ?? 0) > 0 && hasOrderHistory">похоже на ваши прошлые покупки</span>
            <span class="why-chip" v-if="(r.meal_score ?? 0) > 0">пригодится для блюд</span>
            <span class="why-chip neutral" v-if="(r.recency_score ?? 0) < 0">недавно уже покупали</span>
          </div>
        </article>
      </div>
      <p v-else-if="!loading" class="muted hint-empty">
        Пока нет рекомендаций. Добавьте несколько товаров в избранное и оформите хотя бы один заказ — система начнёт подсказывать.
      </p>

      <h3 class="section-title">Блюда, которые можно приготовить из подсказок</h3>
      <p v-if="derivedDishRecommendations.length" class="section-hint">
        Подобраны из тех же товаров, что и в списке выше.
      </p>
      <div v-if="derivedDishRecommendations.length" class="list">
        <article v-for="(d, idx) in derivedDishRecommendations" :key="`${d.dish_id}-${idx}`" class="item">
          <div class="item-head">
            <h3>{{ idx + 1 }}. {{ d.dish_name }}</h3>
            <span class="dish-count-chip">{{ d.usedCount }} {{ productWord(d.usedCount) }} из рекомендуемых</span>
          </div>
          <p class="dish-ingredients" v-if="d.usedProductNames.length">
            Использует: <strong>{{ d.usedProductNames.join(', ') }}</strong>
          </p>
          <p class="missing-hint" v-if="d.minMissing > 0">
            Не хватает ещё {{ d.minMissing }} {{ ingredientWord(d.minMissing) }} — посмотрите в каталоге.
          </p>
          <p class="ready-hint" v-else>
            Все ингредиенты есть в подсказках выше.
          </p>
        </article>
      </div>
      <p v-else-if="!loading" class="muted hint-empty">
        Пока не удалось подобрать блюда. Добавьте больше товаров в избранное — каталог подскажет, что приготовить.
      </p>

      <p v-if="error" class="error">{{ error }}</p>
    </div>
  </section>
</template>

<script setup lang="ts">
import { onMounted, ref, computed } from 'vue'
import type { AxiosError } from 'axios'
import {
  api,
  type ProductRecommendationItem,
  type Order,
} from '../api/http'

const currentUserId = ref<number | null>(null)
const loading = ref(false)
const error = ref<string | null>(null)
const productRecommendations = ref<ProductRecommendationItem[]>([])
const hasOrderHistory = ref(false)
const limit = ref(5)

type DerivedDish = {
  dish_id: number
  dish_name: string
  usedCount: number
  usedProductNames: string[]
  minMissing: number
}

// Блюда выводятся напрямую из рекомендованных товаров:
// агрегируем linked_dishes по всем товарам и считаем, сколько раз каждое блюдо встретилось.
const derivedDishRecommendations = computed<DerivedDish[]>(() => {
  const acc = new Map<number, DerivedDish>()
  for (const r of productRecommendations.value) {
    if (!r.linked_dishes?.length) continue
    for (const ld of r.linked_dishes) {
      const existing = acc.get(ld.dish_id)
      if (existing) {
        existing.usedCount += 1
        existing.usedProductNames.push(r.product.name)
        existing.minMissing = Math.min(existing.minMissing, ld.missing_ingredients_estimate)
      } else {
        acc.set(ld.dish_id, {
          dish_id: ld.dish_id,
          dish_name: ld.dish_name,
          usedCount: 1,
          usedProductNames: [r.product.name],
          minMissing: ld.missing_ingredients_estimate,
        })
      }
    }
  }
  return Array.from(acc.values())
    .sort((a, b) => {
      if (b.usedCount !== a.usedCount) return b.usedCount - a.usedCount
      return a.minMissing - b.minMissing
    })
    .slice(0, limit.value)
})

function ingredientWord(n: number): string {
  const mod10 = n % 10
  const mod100 = n % 100
  if (mod10 === 1 && mod100 !== 11) return 'ингредиента'
  if (mod10 >= 2 && mod10 <= 4 && (mod100 < 12 || mod100 > 14)) return 'ингредиентов'
  return 'ингредиентов'
}

function productWord(n: number): string {
  const mod10 = n % 10
  const mod100 = n % 100
  if (mod10 === 1 && mod100 !== 11) return 'товар'
  if (mod10 >= 2 && mod10 <= 4 && (mod100 < 12 || mod100 > 14)) return 'товара'
  return 'товаров'
}

async function loadRecommendations() {
  if (!currentUserId.value) return
  loading.value = true
  error.value = null

  const [productRes, ordersRes] = await Promise.allSettled([
    api.get<ProductRecommendationItem[]>(
      `/v1/users/${currentUserId.value}/recommendations/products`,
      { params: { limit: limit.value } },
    ),
    api.get<Order[]>('/v1/orders'),
  ])

  if (productRes.status === 'fulfilled') {
    productRecommendations.value = Array.isArray(productRes.value.data) ? productRes.value.data : []
  } else {
    productRecommendations.value = []
  }

  if (ordersRes.status === 'fulfilled') {
    hasOrderHistory.value = Array.isArray(ordersRes.value.data) && ordersRes.value.data.length > 0
  } else {
    hasOrderHistory.value = false
  }

  // Никаких подробностей серверной ошибки наружу — только дружелюбный текст.
  if (productRes.status === 'rejected') {
    const status = ((productRes.reason as AxiosError)?.response?.status) ?? 0
    if (status === 401) {
      error.value = 'Сессия истекла. Пожалуйста, войдите заново.'
    } else {
      error.value = 'Сервис рекомендаций временно недоступен. Попробуйте обновить позже.'
    }
  }

  loading.value = false
}

onMounted(() => {
  const raw = localStorage.getItem('currentUserId')
  currentUserId.value = raw ? Number(raw) : null
  if (currentUserId.value) {
    void loadRecommendations()
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

.header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  margin-bottom: 10px;
}

.controls {
  display: flex;
  align-items: center;
  gap: 8px;
}

.inline-label {
  font-size: 12px;
  color: #4b5563;
  display: flex;
  align-items: center;
  gap: 6px;
}

.inline-label select {
  border: 1px solid #d1d5db;
  border-radius: 6px;
  padding: 4px 6px;
  background: #fff;
}

.primary {
  border-radius: 6px;
  border: 1px solid #111827;
  background: #111827;
  color: #fff;
  padding: 6px 12px;
  cursor: pointer;
}

.list {
  display: flex;
  flex-direction: column;
  gap: 8px;
  margin-top: 10px;
}

.item {
  border: 1px solid #e5e7eb;
  border-radius: 8px;
  padding: 12px 14px;
  background: #fafafa;
}

.item-head {
  display: flex;
  justify-content: space-between;
  align-items: baseline;
  gap: 8px;
}

h3 {
  margin: 0 0 4px;
  font-size: 15px;
}

.price {
  font-size: 14px;
  font-weight: 600;
  color: #111827;
  white-space: nowrap;
}

.error {
  margin-top: 10px;
  color: #b91c1c;
  font-size: 13px;
}

.hint-empty {
  padding: 12px;
  background: #f9fafb;
  border: 1px dashed #d1d5db;
  border-radius: 8px;
  margin-top: 10px;
}

.section-title {
  margin: 20px 0 6px;
  font-size: 16px;
  color: #111827;
}

.section-hint {
  margin: 0 0 8px;
  font-size: 12px;
  color: #6b7280;
  font-style: italic;
}

.why-row {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  margin: 6px 0;
}

.why-chip {
  display: inline-block;
  padding: 3px 9px;
  background: #ecfdf5;
  color: #065f46;
  border: 1px solid #a7f3d0;
  border-radius: 12px;
  font-size: 12px;
}

.why-chip.neutral {
  background: #fef3c7;
  color: #92400e;
  border-color: #fcd34d;
}

.dish-count-chip {
  display: inline-block;
  padding: 3px 9px;
  background: #eff6ff;
  color: #1d4ed8;
  border: 1px solid #bfdbfe;
  border-radius: 12px;
  font-size: 12px;
  white-space: nowrap;
}

.dish-ingredients {
  margin: 6px 0 0;
  font-size: 13px;
  color: #374151;
}

.dish-ingredients strong {
  color: #111827;
  font-weight: 600;
}

.missing-hint {
  margin: 4px 0 0;
  font-size: 12px;
  color: #b45309;
}

.ready-hint {
  margin: 4px 0 0;
  font-size: 12px;
  color: #15803d;
  font-weight: 500;
}
</style>
