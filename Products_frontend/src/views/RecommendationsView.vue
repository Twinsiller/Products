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
          <p class="muted">Показываем, почему товар рекомендован: оценка, факторы и блюда, которые можно приготовить.</p>
        </div>
        <div class="controls">
          <label class="inline-label">
            Top-K:
            <select v-model.number="limit" :disabled="loading">
              <option :value="3">3</option>
              <option :value="5">5</option>
              <option :value="10">10</option>
            </select>
          </label>
          <button class="primary" type="button" :disabled="loading" @click="loadRecommendations">
            {{ loading ? 'Загрузка...' : 'Обновить рекомендации' }}
          </button>
        </div>
      </header>

      <p class="muted hint">
        Товары: <code>/v1/users/{{ currentUserId }}/recommendations/products?limit={{ limit }}</code>
      </p>
      <p class="muted hint">
        Блюда (ML): <code>/v1/users/{{ currentUserId }}/recommendations/final?limit={{ limit }}</code>
      </p>
      <p v-if="precision !== null" class="muted">
        Метрика ML-блюд: precision@5 = <strong>{{ precision.toFixed(3) }}</strong>
      </p>

      <h3 class="section-title">Что купить и почему</h3>
      <div v-if="productRecommendations.length" class="list">
        <article v-for="(r, idx) in productRecommendations" :key="`${r.product.id}-${idx}`" class="item">
          <h3>{{ idx + 1 }}. {{ r.product.name }}</h3>
          <p class="meta">
            Итоговая оценка: <strong>{{ toFixedSafe(r.score, 4) }}</strong>
            · Цена: {{ Number(r.product.default_price || 0).toFixed(2) }} ₽
          </p>
          <p class="meta">
            Компоненты: КБЖУ={{ toFixedSafe(r.cb_score, 3) }} · История={{ toFixedSafe(r.cf_score, 3) }}
            · Блюда={{ toFixedSafe(r.meal_score, 3) }} · Штраф за недавнюю покупку={{ toFixedSafe(r.recency_score, 3) }}
          </p>
          <p v-if="r.reason" class="meta reason">{{ r.reason }}</p>

          <div v-if="r.linked_dishes?.length" class="dish-links">
            <p class="meta"><strong>Если взять этот товар, подходят блюда:</strong></p>
            <p v-for="d in r.linked_dishes" :key="`${r.product.id}-${d.dish_id}`" class="meta">
              • {{ d.dish_name }} (оценка блюда: {{ toFixedSafe(d.dish_score, 3) }}, недостающих ингредиентов: {{ d.missing_ingredients_estimate }})
            </p>
          </div>
        </article>
      </div>
      <p v-else-if="!loading" class="muted">Пока нет товарных рекомендаций.</p>

      <h3 class="section-title">Рекомендованные блюда (ML)</h3>
      <div v-if="dishRecommendations.length" class="list">
        <article v-for="(r, idx) in dishRecommendations" :key="`${r.recipe_id}-${idx}`" class="item">
          <h3>{{ idx + 1 }}. {{ r.recipe_name }}</h3>
          <p class="meta">ID: {{ r.recipe_id }} · score: {{ r.score.toFixed(4) }}</p>
          <p v-if="r.kcal !== undefined" class="meta">
            КБЖУ: {{ r.kcal?.toFixed(0) }} ккал · Б {{ r.protein_g?.toFixed(1) }} · Ж {{ r.fat_g?.toFixed(1) }} · У {{ r.carbs_g?.toFixed(1) }}
          </p>
        </article>
      </div>
      <p v-else-if="!loading" class="muted">Пока нет рекомендаций блюд.</p>

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

async function loadRecommendations() {
  if (!currentUserId.value) return
  loading.value = true
  error.value = null
  try {
    const [productRes, finalRes] = await Promise.all([
      api.get<ProductRecommendationItem[]>(
        `/v1/users/${currentUserId.value}/recommendations/products`,
        { params: { limit: limit.value } },
      ),
      api.get<FinalRecommendationsResponse>(
        `/v1/users/${currentUserId.value}/recommendations/final`,
        { params: { limit: limit.value } },
      ),
    ])
    productRecommendations.value = Array.isArray(productRes.data) ? productRes.data : []
    precision.value = Number(finalRes.data.precision_at_5 ?? 0)
    dishRecommendations.value = Array.isArray(finalRes.data.recommendations) ? finalRes.data.recommendations : []
  } catch (e) {
    const err = e as AxiosError<{ error?: string }>
    const serverMsg = err.response?.data?.error
    error.value = typeof serverMsg === 'string' && serverMsg.trim()
      ? `Не удалось получить рекомендации: ${serverMsg}`
      : 'Не удалось получить рекомендации'
    productRecommendations.value = []
    dishRecommendations.value = []
  } finally {
    loading.value = false
  }
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
  padding: 10px 12px;
  background: #fafafa;
}

h3 {
  margin: 0 0 4px;
  font-size: 14px;
}

.meta {
  margin: 0;
  font-size: 12px;
  color: #4b5563;
}

.error {
  margin-top: 10px;
  color: #b91c1c;
  font-size: 13px;
}

.hint {
  margin-top: 6px;
}

.section-title {
  margin: 16px 0 8px;
  font-size: 15px;
}

.reason {
  margin-top: 6px;
}

.dish-links {
  margin-top: 6px;
}
</style>
