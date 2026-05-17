<template>
  <section class="card">
    <div v-if="!isAdmin" class="guard">
      <h2>Раздел только для администратора</h2>
      <p class="muted">Просмотр пользователей и их данных доступен только администратору.</p>
    </div>

    <div v-else>
      <header class="header">
        <div>
          <h2>Пользователи</h2>
          <p class="muted">
            Все зарегистрированные пользователи: профиль, заказы, состав заказов и избранное.
          </p>
        </div>
        <button class="primary" type="button" @click="loadUsers" :disabled="loading">
          {{ loading ? 'Загрузка…' : 'Обновить' }}
        </button>
      </header>

      <div class="users-grid">
        <article v-for="u in users" :key="u.id" class="user-tile">
          <h3>№{{ u.id }} — {{ u.name }}</h3>
          <p class="meta">
            Роль: {{ humanRole(u.role) }} · Пол: {{ humanGender(u.gender) }} · Создан: {{ formatDate(u.hired_at) }}
          </p>

          <section class="block">
            <h4>Заказы ({{ u.orders?.length || 0 }})</h4>
            <div v-if="u.orders?.length" class="list">
              <div v-for="o in u.orders" :key="o.id" class="line">
                <strong>Заказ №{{ o.id }}</strong> — {{ formatDate(o.created_at) }} — {{ toMoney(o.total_amount) }} ₽
                <div v-if="o.items?.length" class="sublist">
                  <div v-for="it in o.items" :key="it.id">
                    {{ it.product?.name || ('Товар №' + it.product_id) }} × {{ it.quantity }} — {{ toMoney(it.price_per_unit) }} ₽
                  </div>
                </div>
              </div>
            </div>
            <p v-else class="muted small">Заказов нет</p>
          </section>

          <section class="block">
            <h4>Избранные товары ({{ u.favourite_products?.length || 0 }})</h4>
            <div v-if="u.favourite_products?.length" class="chips">
              <span v-for="fp in u.favourite_products" :key="fp.id" class="chip">
                {{ fp.product?.name || ('Товар №' + fp.product_id) }}
              </span>
            </div>
            <p v-else class="muted small">Избранных товаров нет</p>
          </section>

          <section class="block">
            <h4>Избранные блюда ({{ u.favourite_dishes?.length || 0 }})</h4>
            <div v-if="u.favourite_dishes?.length" class="chips">
              <span v-for="fd in u.favourite_dishes" :key="fd.id" class="chip">
                {{ fd.dish?.name || ('Блюдо №' + fd.dish_id) }}
              </span>
            </div>
            <p v-else class="muted small">Избранных блюд нет</p>
          </section>
        </article>
      </div>

      <p v-if="!loading && !users.length" class="muted">В системе пока нет зарегистрированных пользователей.</p>
      <p v-if="error" class="error">{{ error }}</p>
    </div>
  </section>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { api } from '../api/http'

interface OrderItemLite {
  id: number
  product_id: number
  quantity: number
  price_per_unit: number
  product?: { id: number; name: string }
}

interface OrderLite {
  id: number
  created_at: string
  total_amount: number
  items?: OrderItemLite[]
}

interface FavouriteProductLite {
  id: number
  product_id: number
  product?: { id: number; name: string }
}

interface FavouriteDishLite {
  id: number
  dish_id: number
  dish?: { id: number; name: string }
}

interface AdminUserView {
  id: number
  name: string
  role: string
  gender?: string
  hired_at: string
  orders?: OrderLite[]
  favourite_products?: FavouriteProductLite[]
  favourite_dishes?: FavouriteDishLite[]
}

const users = ref<AdminUserView[]>([])
const loading = ref(false)
const error = ref<string | null>(null)
const isAdmin = ref(false)

const toMoney = (v: number) => Number(v || 0).toFixed(2)
const formatDate = (s: string) => (s ? new Date(s).toLocaleString() : '—')

function humanRole(role: string): string {
  if (role === 'admin') return 'администратор'
  if (role === 'cashier') return 'кассир'
  if (role === 'user') return 'пользователь'
  return role || 'пользователь'
}

function humanGender(g?: string): string {
  if (g === 'male') return 'мужской'
  if (g === 'female') return 'женский'
  return 'не указан'
}

const loadUsers = async () => {
  loading.value = true
  error.value = null
  try {
    const { data } = await api.get<AdminUserView[]>('/v1/users/full')
    users.value = data
  } catch (e: unknown) {
    error.value = 'Не удалось загрузить список пользователей. Попробуйте обновить.'
  } finally {
    loading.value = false
  }
}

onMounted(() => {
  const role = localStorage.getItem('currentUserRole') || 'user'
  isAdmin.value = role === 'admin'
  if (isAdmin.value) {
    void loadUsers()
  }
})
</script>

<style scoped>
.card { background: #fff; border-radius: 8px; padding: 16px 18px; border: 1px solid #e5e7eb; }
.header { display: flex; justify-content: space-between; gap: 12px; margin-bottom: 12px; }
h2 { margin: 0 0 4px; font-size: 18px; }
.muted { margin: 0; font-size: 13px; color: #6b7280; }
.small { font-size: 12px; }
.primary { border-radius: 4px; padding: 6px 12px; border: 1px solid #d1d5db; font-size: 13px; cursor: pointer; background: #111827; color: #f9fafb; font-weight: 500; }
.primary:disabled { opacity: 0.6; cursor: default; }

.users-grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(320px, 1fr)); gap: 12px; }
.user-tile { border: 1px solid #e5e7eb; border-radius: 8px; padding: 10px 12px; background: #fafafa; }
.user-tile h3 { margin: 0 0 4px; font-size: 16px; }
.meta { margin: 0 0 10px; font-size: 12px; color: #6b7280; }
.block { margin-top: 10px; }
.block h4 { margin: 0 0 6px; font-size: 13px; }
.line { font-size: 12px; margin-bottom: 6px; }
.sublist { margin: 4px 0 0 10px; color: #4b5563; }
.chips { display: flex; flex-wrap: wrap; gap: 6px; }
.chip { border: 1px solid #d1d5db; border-radius: 999px; padding: 2px 8px; font-size: 12px; background: #fff; }
.error { margin-top: 8px; font-size: 13px; color: #fca5a5; }
</style>
