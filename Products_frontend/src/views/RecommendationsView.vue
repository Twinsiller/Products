<template>
  <section class="card">
    <div v-if="!currentUserId" class="guard">
      <h2>Рекомендации</h2>
      <p class="muted">Войдите в аккаунт, чтобы получить персональные рекомендации.</p>
    </div>

    <div v-else>
      <header class="header">
        <div>
          <h2>ML модель в действии</h2>
          <p class="muted">Запуск Python-рекомендера в реальном времени (CF/CB + fallback).</p>
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
          <button class="primary" type="button" :disabled="loading" @click="loadFinalRecommendations">
            {{ loading ? 'Запуск модели...' : 'Запустить ML модель' }}
          </button>
        </div>
      </header>

      <p v-if="precision !== null" class="muted">
        Метрика модели: precision@5 = <strong>{{ precision.toFixed(3) }}</strong>
      </p>
      <p class="muted hint">
        Endpoint: <code>/v1/users/{{ currentUserId }}/recommendations/final?limit={{ limit }}</code>
      </p>

      <div v-if="recommendations.length" class="list">
        <article v-for="(r, idx) in recommendations" :key="`${r.recipe_id}-${idx}`" class="item">
          <h3>{{ idx + 1 }}. {{ r.recipe_name }}</h3>
          <p class="meta">ID: {{ r.recipe_id }} · score: {{ r.score.toFixed(4) }}</p>
          <p v-if="r.kcal !== undefined" class="meta">
            КБЖУ: {{ r.kcal?.toFixed(0) }} ккал · Б {{ r.protein_g?.toFixed(1) }} · Ж {{ r.fat_g?.toFixed(1) }} · У {{ r.carbs_g?.toFixed(1) }}
          </p>
        </article>
      </div>
      <p v-else-if="!loading" class="muted">Пока нет рекомендаций.</p>

      <p v-if="error" class="error">{{ error }}</p>
    </div>
  </section>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import type { AxiosError } from 'axios'
import { api, type FinalRecommendationItem, type FinalRecommendationsResponse } from '../api/http'

const currentUserId = ref<number | null>(null)
const loading = ref(false)
const error = ref<string | null>(null)
const precision = ref<number | null>(null)
const recommendations = ref<FinalRecommendationItem[]>([])
const limit = ref(5)

async function loadFinalRecommendations() {
  if (!currentUserId.value) return
  loading.value = true
  error.value = null
  try {
    const { data } = await api.get<FinalRecommendationsResponse>(
      `/v1/users/${currentUserId.value}/recommendations/final`,
      { params: { limit: limit.value } },
    )
    precision.value = Number(data.precision_at_5 ?? 0)
    recommendations.value = Array.isArray(data.recommendations) ? data.recommendations : []
  } catch (e) {
    const err = e as AxiosError<{ error?: string }>
    const serverMsg = err.response?.data?.error
    error.value = typeof serverMsg === 'string' && serverMsg.trim()
      ? `Не удалось получить финальные рекомендации: ${serverMsg}`
      : 'Не удалось получить финальные рекомендации'
    recommendations.value = []
  } finally {
    loading.value = false
  }
}

onMounted(() => {
  const raw = localStorage.getItem('currentUserId')
  currentUserId.value = raw ? Number(raw) : null
  if (currentUserId.value) {
    void loadFinalRecommendations()
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
</style>
