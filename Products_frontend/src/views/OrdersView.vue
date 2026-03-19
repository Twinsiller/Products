<template>
  <section class="card">
    <div v-if="!currentUser" class="guard">
      <h2>Раздел заказов доступен только после входа</h2>
      <p class="muted">
        Авторизуйтесь, чтобы просматривать историю заказов и оформлять новые.
      </p>
      <div class="guard-actions">
        <RouterLink to="/" class="primary-link">Войти</RouterLink>
        <RouterLink to="/register" class="secondary-link">Зарегистрироваться</RouterLink>
      </div>
    </div>

    <div v-else>
      <header class="header">
        <div>
          <h2>Заказы</h2>
          <p class="muted">
            Список оформленных заказов.
          </p>
        </div>
        <button
          class="primary"
          type="button"
          @click="loadOrders"
          :disabled="loading"
        >
          {{ loading ? 'Загрузка...' : 'Обновить' }}
        </button>
      </header>

      <table class="table">
        <thead>
          <tr>
            <th>ID</th>
            <th>Кассир</th>
            <th>Дата</th>
            <th>Сумма</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="o in orders" :key="o.id">
            <td>{{ o.id }}</td>
            <td>{{ o.cashier_id }}</td>
            <td>{{ new Date(o.created_at).toLocaleString() }}</td>
            <td>{{ o.total_amount.toFixed(2) }}</td>
          </tr>
          <tr v-if="!loading && !orders.length">
            <td colspan="4" class="empty">
              Заказов пока нет.
            </td>
          </tr>
        </tbody>
      </table>

      <p v-if="error" class="error">
        {{ error }}
      </p>
    </div>
  </section>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { RouterLink } from 'vue-router'
import { api, type Order } from '../api/http'

const currentUser = ref<{ id: number } | null>(null)
const orders = ref<Order[]>([])
const loading = ref(false)
const error = ref<string | null>(null)

const loadOrders = async () => {
  loading.value = true
  error.value = null
  try {
    const { data } = await api.get<Order[]>('/v1/orders')
    orders.value = data
  } catch (e) {
    error.value = 'Ошибка загрузки заказов'
  } finally {
    loading.value = false
  }
}

onMounted(() => {
  const storedId = localStorage.getItem('currentUserId')
  if (storedId) {
    currentUser.value = { id: Number(storedId) }
    void loadOrders()
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

.error {
  margin-top: 8px;
  font-size: 13px;
  color: #fca5a5;
}
</style>

