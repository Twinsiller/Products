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
          <p v-if="r.reason" class="reason">{{ r.reason }}</p>

          <div class="why-row">
            <span class="why-chip" v-if="r.cb_score > 0">подходит по КБЖУ</span>
            <span class="why-chip" v-if="r.cf_score > 0">похоже на ваши прошлые покупки</span>
            <span class="why-chip" v-if="r.meal_score > 0">пригодится для блюд</span>
            <span class="why-chip neutral" v-if="r.recency_score < 0">недавно уже покупали</span>
          </div>

          <div v-if="r.linked_dishes?.length" class="dish-links">
            <p class="dish-links-title">Из этого товара можно приготовить:</p>
            <ul>
              <li v-for="d in r.linked_dishes" :key="`${r.product.id}-${d.dish_id}`">
                {{ d.dish_name }}
                <span class="missing" v-if="d.missing_ingredients_estimate > 0">
                  — не хватает ещё {{ d.missing_ingredients_estimate }} {{ ingredientWord(d.missing_ingredients_estimate) }}
                </span>
                <span class="ready" v-else>— все ингредиенты есть</span>
              </li>
            </ul>
          </div>
        </article>
      </div>
      <p v-else-if="!loading" class="muted hint-empty">
        Пока нет рекомендаций. Добавьте несколько товаров в избранное и оформите хотя бы один заказ — система начнёт подсказывать.
      </p>

      <h3 class="section-title">Подобранные блюда</h3>
      <div v-if="dishRecommendations.length" class="list">
        <article v-for="(r, idx) in dishRecommendations" :key="`${r.recipe_id}-${idx}`" class="item">
          <h3>{{ idx + 1 }}. {{ r.recipe_name }}</h3>
          <p v-if="r.kcal !== undefined" class="kbju">
            <strong>{{ r.kcal?.toFixed(0) }} ккал</strong>
            <span class="kbju-sep">·</span> Б <strong>{{ r.protein_g?.toFixed(1) }}</strong>
            <span class="kbju-sep">·</span> Ж <strong>{{ r.fat_g?.toFixed(1) }}</strong>
            <span class="kbju-sep">·</span> У <strong>{{ r.carbs_g?.toFixed(1) }}</strong>
          </p>
        </article>
      </div>
      <p v-else-if="!loading" class="muted hint-empty">
        Пока не подобрали блюд. Добавьте больше товаров в корзину или избранное.
      </p>

      <p v-if="error" class="error">{{ error }}</p>
    </div>
  </section>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import type { AxiosError } from 'axios'
import {
  api,
  type FinalRecommendationItem,
  type FinalRecommendationsResponse,
  type ProductRecommendationItem,
} from '../api/http'

const currentUserId = ref<number | null>(null)
const loading = ref(false)
const error = ref<string | null>(null)
const precision = ref<number | null>(null)
const dishRecommendations = ref<FinalRecommendationItem[]>([])
const productRecommendations = ref<ProductRecommendationItem[]>([])
const limit = ref(5)

function toFixedSafe(value: number | undefined, digits = 3): string {
  if (typeof value !== 'number' || Number.isNaN(value)) return '0.000'
  return value.toFixed(digits)
}

function ingredientWord(n: number): string {
  const mod10 = n % 10
  const mod100 = n % 100
  if (mod10 === 1 && mod100 !== 11) return 'ингредиента'
  if (mod10 >= 2 && mod10 <= 4 && (mod100 < 12 || mod100 > 14)) return 'ингредиентов'
  return 'ингредиентов'
}

async function loadRecommendations() {
  if (!currentUserId.value) return
  loading.value = true
  error.value = null

  const [productRes, finalRes] = await Promise.allSettled([
    api.get<ProductRecommendationItem[]>(
      `/v1/users/${currentUserId.value}/recommendations/products`,
      { params: { limit: limit.value } },
    ),
    api.get<FinalRecommendationsResponse>(
      `/v1/users/${currentUserId.value}/recommendations/final`,
      { params: { limit: limit.value } },
    ),
  ])

  if (productRes.status === 'fulfilled') {
    productRecommendations.value = Array.isArray(productRes.value.data) ? productRes.value.data : []
  } else {
    productRecommendations.value = []
  }

  if (finalRes.status === 'fulfilled') {
    precision.value = Number(finalRes.value.data.precision_at_5 ?? 0)
    dishRecommendations.value = Array.isArray(finalRes.value.data.recommendations)
      ? finalRes.value.data.recommendations
      : []
  } else {
    precision.value = null
    dishRecommendations.value = []
  }

  // Никаких подробностей серверной ошибки наружу — только дружелюбный текст.
  if (productRes.status === 'rejected' && finalRes.status === 'rejected') {
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
  margin: 20px 0 10px;
  font-size: 16px;
  color: #111827;
}

.reason {
  margin: 4px 0 8px;
  font-size: 13px;
  color: #374151;
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

.dish-links {
  margin-top: 10px;
  padding-top: 8px;
  border-top: 1px dashed #e5e7eb;
}

.dish-links-title {
  margin: 0 0 4px;
  font-size: 13px;
  font-weight: 600;
  color: #111827;
}

.dish-links ul {
  margin: 0;
  padding-left: 18px;
  font-size: 13px;
  color: #4b5563;
}

.dish-links li {
  margin: 3px 0;
}

.missing {
  color: #b45309;
}

.ready {
  color: #15803d;
  font-weight: 500;
}

.kbju {
  margin: 4px 0 0;
  font-size: 13px;
  color: #374151;
}

.kbju strong {
  color: #111827;
}

.kbju-sep {
  color: #9ca3af;
  margin: 0 4px;
}
</style>
