<template>
  <section class="card">
    <div v-if="!currentUser" class="guard">
      <h2>Войдите, чтобы пользоваться корзиной и заказами</h2>
      <p class="muted">
        После входа вы сможете собирать корзину, оформлять заказы и видеть их историю.
      </p>
      <div class="guard-actions">
        <RouterLink to="/" class="primary-link">Войти</RouterLink>
        <RouterLink to="/register" class="secondary-link">Зарегистрироваться</RouterLink>
      </div>
    </div>

    <div v-else>
      <section class="cart-block">
        <header class="header cart-header">
          <div>
            <h2>Корзина</h2>
            <p class="muted">Добавляйте товары на странице «Товары», а здесь оформляйте заказ.</p>
          </div>
          <div class="cart-actions">
            <button class="secondary" type="button" @click="clearCart" :disabled="!cartItems.length">
              Очистить корзину
            </button>
            <button class="primary" type="button" @click="createOrderFromCart" :disabled="creatingOrder || !cartItems.length">
              {{ creatingOrder ? 'Оформление...' : 'Оформить заказ' }}
            </button>
          </div>
        </header>
        <div v-if="cartItems.length" class="cart-list">
          <article v-for="item in cartItems" :key="item.product_id" class="cart-item">
            <div>
              <strong>{{ item.name }}</strong>
              <p class="muted small">{{ item.price_per_unit.toFixed(2) }} ₽ за шт.</p>
            </div>
            <div class="cart-item-controls">
              <button type="button" class="qty-btn" @click="decQty(item.product_id)">−</button>
              <input
                :value="item.quantity"
                type="number"
                min="1"
                class="qty-input"
                @change="setQty(item.product_id, ($event.target as HTMLInputElement).value)"
              />
              <button type="button" class="qty-btn" @click="incQty(item.product_id)">+</button>
              <button type="button" class="link-btn danger" @click="removeItem(item.product_id)">Убрать</button>
            </div>
          </article>
          <p class="cart-total">Итого: <strong>{{ cartTotal.toFixed(2) }} ₽</strong></p>
        </div>
        <p v-else class="muted empty-cart">
          Корзина пока пуста. Перейдите в раздел «Товары» и добавьте, что захотите.
        </p>
      </section>

      <header class="header">
        <div>
          <h2>Мои заказы</h2>
          <p class="muted">
            История заказов: товары, количество, цены и итог.
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

      <div v-if="orders.length" class="orders-grid">
        <article v-for="o in orders" :key="o.id" class="order-card">
          <header class="order-card-header">
            <div>
              <h3>Заказ №{{ o.id }}</h3>
              <p class="muted small">{{ new Date(o.created_at).toLocaleString() }}</p>
            </div>
            <div class="order-total">{{ o.total_amount.toFixed(2) }} ₽</div>
          </header>

          <div v-if="itemsLoadingByOrder[o.id]" class="muted small">Загружаем состав заказа…</div>
          <div v-else-if="orderItemsByOrder[o.id]?.length" class="items-list">
            <div v-for="it in orderItemsByOrder[o.id]" :key="it.id" class="item-line">
              <div class="item-name">{{ it.product?.name || ('Товар №' + it.product_id) }}</div>
              <div class="item-meta">
                <span>× {{ it.quantity }} шт.</span>
                <span>{{ it.price_per_unit.toFixed(2) }} ₽</span>
                <strong>{{ lineTotal(it).toFixed(2) }} ₽</strong>
              </div>
            </div>
          </div>
          <span v-else class="muted small">В этом заказе нет позиций</span>
        </article>
      </div>
      <p v-else-if="!loading" class="empty">У вас ещё нет оформленных заказов.</p>

      <p v-if="error" class="error">
        {{ error }}
      </p>
    </div>
  </section>
</template>

<script setup lang="ts">
import { onBeforeUnmount, onMounted, computed, ref } from 'vue'
import { RouterLink } from 'vue-router'
import { api, type Order } from '../api/http'

const currentUser = ref<{ id: number } | null>(null)
const orders = ref<Order[]>([])
const loading = ref(false)
const error = ref<string | null>(null)
const itemsLoadingByOrder = ref<Record<number, boolean>>({})
const orderItemsByOrder = ref<Record<number, OrderItemView[]>>({})
const creatingOrder = ref(false)

type CartItem = {
  product_id: number
  name: string
  price_per_unit: number
  quantity: number
}
type OrderItemView = {
  id: number
  order_id: number
  product_id: number
  quantity: number
  price_per_unit: number
  discount: number
  product?: { id: number; name: string }
}

const cartItems = ref<CartItem[]>([])
const cartTotal = computed(() =>
  cartItems.value.reduce((acc, x) => acc + x.price_per_unit * x.quantity, 0),
)

function loadCart() {
  const raw = localStorage.getItem('cartItems')
  cartItems.value = raw ? JSON.parse(raw) : []
}

function saveCart() {
  localStorage.setItem('cartItems', JSON.stringify(cartItems.value))
  window.dispatchEvent(new Event('cart-changed'))
}

function clearCart() {
  cartItems.value = []
  saveCart()
}

function removeItem(productId: number) {
  cartItems.value = cartItems.value.filter((x) => x.product_id !== productId)
  saveCart()
}

function incQty(productId: number) {
  const item = cartItems.value.find((x) => x.product_id === productId)
  if (!item) return
  item.quantity += 1
  saveCart()
}

function decQty(productId: number) {
  const item = cartItems.value.find((x) => x.product_id === productId)
  if (!item) return
  item.quantity = Math.max(1, item.quantity - 1)
  saveCart()
}

function setQty(productId: number, raw: string) {
  const item = cartItems.value.find((x) => x.product_id === productId)
  if (!item) return
  item.quantity = Math.max(1, Math.floor(Number(raw) || 1))
  saveCart()
}

const loadOrders = async () => {
  loading.value = true
  error.value = null
  try {
    const { data } = await api.get<Order[]>('/v1/orders')
    orders.value = data
    const loadingMap: Record<number, boolean> = {}
    for (const o of data) loadingMap[o.id] = true
    itemsLoadingByOrder.value = loadingMap
    await Promise.all(data.map(async (o) => {
      try {
        const { data: items } = await api.get<OrderItemView[]>(`/v1/orders/${o.id}/items`)
        orderItemsByOrder.value[o.id] = items
      } catch {
        orderItemsByOrder.value[o.id] = []
      } finally {
        itemsLoadingByOrder.value[o.id] = false
      }
    }))
  } catch (e) {
    error.value = 'Не удалось загрузить заказы. Попробуйте обновить страницу.'
  } finally {
    loading.value = false
  }
}

function lineTotal(it: OrderItemView): number {
  const base = Number(it.price_per_unit || 0) * Number(it.quantity || 0)
  const d = Number(it.discount || 0)
  const discount = d > 0 ? Math.min(100, d) : 0
  return base * (1 - discount / 100)
}

const createOrderFromCart = async () => {
  if (!currentUser.value || !cartItems.value.length) return
  creatingOrder.value = true
  error.value = null
  try {
    // Перед оформлением заказа проверим, что все товары из корзины
    // ещё существуют в каталоге (могли быть удалены, или БД пересоздана).
    let availableIds = new Set<number>()
    try {
      const { data: products } = await api.get<Array<{ id: number }>>('/v1/products')
      availableIds = new Set(products.map((p) => p.id))
    } catch {
      // Если каталог не отдаётся — продолжаем без фильтра, серверная сторона сама отсечёт.
    }

    const validItems = availableIds.size
      ? cartItems.value.filter((it) => availableIds.has(it.product_id))
      : cartItems.value

    if (!validItems.length) {
      error.value = 'В корзине не осталось доступных товаров. Очистите её и подберите заново.'
      creatingOrder.value = false
      return
    }

    const total = validItems.reduce((acc, x) => acc + x.price_per_unit * x.quantity, 0)

    // Создаём заказ: user_id подставит сам сервер из JWT, total_amount передаём сразу.
    const { data: order } = await api.post<Order>('/v1/orders', {
      total_amount: total,
    })

    // Добавляем позиции заказа.
    const failedItems: number[] = []
    for (const item of validItems) {
      try {
        await api.post('/v1/order-items', {
          order_id: order.id,
          product_id: item.product_id,
          quantity: item.quantity,
          price_per_unit: item.price_per_unit,
          discount: 0,
        })
      } catch {
        failedItems.push(item.product_id)
      }
    }

    if (failedItems.length === validItems.length) {
      // Все позиции не добавились — заказ останется пустым, лучше показать ошибку.
      error.value = 'Не удалось добавить товары в заказ. Попробуйте оформить заказ заново.'
      return
    }

    clearCart()
    await loadOrders()
  } catch (e) {
    error.value = 'Не удалось оформить заказ. Проверьте, что вы вошли в аккаунт, и попробуйте ещё раз.'
  } finally {
    creatingOrder.value = false
  }
}

onMounted(() => {
  const storedId = localStorage.getItem('currentUserId')
  if (storedId) {
    currentUser.value = { id: Number(storedId) }
    loadCart()
    window.addEventListener('cart-changed', loadCart)
    void loadOrders()
  }
})

onBeforeUnmount(() => {
  window.removeEventListener('cart-changed', loadCart)
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

.empty {
  text-align: left;
  color: #9ca3af;
}

.orders-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(380px, 1fr));
  gap: 12px;
}

.order-card {
  border: 1px solid #e5e7eb;
  border-radius: 12px;
  padding: 12px;
  background: linear-gradient(180deg, #ffffff 0%, #f8fafc 100%);
}

.order-card-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  margin-bottom: 8px;
}

.order-card-header h3 {
  margin: 0;
  font-size: 15px;
}

.order-total {
  background: #111827;
  color: #fff;
  border-radius: 999px;
  padding: 4px 10px;
  font-size: 12px;
  font-weight: 600;
}

.items-list {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.item-line {
  border: 1px solid #e5e7eb;
  background: #fff;
  border-radius: 8px;
  padding: 8px;
}

.item-name {
  font-size: 13px;
  font-weight: 600;
  margin-bottom: 4px;
}

.item-meta {
  display: flex;
  gap: 10px;
  font-size: 12px;
  color: #4b5563;
}

.error {
  margin-top: 8px;
  font-size: 13px;
  color: #fca5a5;
}

.small {
  font-size: 12px;
}

.link-btn {
  background: none;
  border: none;
  color: #2563eb;
  cursor: pointer;
  font-size: 12px;
  text-decoration: underline;
  padding: 0;
}

.link-btn.active {
  font-weight: 600;
  color: #111827;
  text-decoration: none;
}

.meals-section {
  margin-top: 24px;
  padding-top: 16px;
  border-top: 1px solid #e5e7eb;
}

.meals-section h3 {
  margin: 0 0 8px;
  font-size: 16px;
}

.meals-list {
  margin-top: 14px;
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.meal-card {
  border: 1px solid #e5e7eb;
  border-radius: 8px;
  padding: 12px 14px;
  background: #fafafa;
}

.meal-card header {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 8px;
}

.meal-rank {
  font-size: 12px;
  color: #6b7280;
}

.meal-card h4 {
  margin: 0;
  flex: 1;
  font-size: 15px;
}

.badge {
  font-size: 11px;
  padding: 2px 8px;
  border-radius: 999px;
}

.badge.ok {
  background: #d1fae5;
  color: #065f46;
}

.badge.warn {
  background: #fef3c7;
  color: #92400e;
}

.meal-meta {
  margin: 8px 0 0;
  font-size: 12px;
  color: #4b5563;
}

.missing-block {
  margin-top: 10px;
}

.missing-title {
  margin: 0 0 6px;
  font-size: 13px;
}

.missing-row {
  margin-bottom: 10px;
  font-size: 13px;
}

.suggestions {
  margin: 6px 0 0;
  padding-left: 18px;
  font-size: 12px;
}

.kbzh-mini {
  color: #9ca3af;
  margin-left: 4px;
}

.cart-block {
  border: 1px solid #e5e7eb;
  border-radius: 8px;
  padding: 12px 14px;
  margin-bottom: 14px;
  background: #fafafa;
}

.cart-header {
  margin-bottom: 8px;
}

.cart-actions {
  display: flex;
  gap: 8px;
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

.cart-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.cart-item {
  display: flex;
  justify-content: space-between;
  gap: 12px;
  align-items: center;
  border: 1px solid #e5e7eb;
  border-radius: 8px;
  padding: 8px 10px;
  background: #fff;
}

.cart-item-controls {
  display: flex;
  align-items: center;
  gap: 6px;
}

.qty-btn {
  width: 28px;
  height: 28px;
  border: 1px solid #d1d5db;
  border-radius: 6px;
  background: #fff;
  cursor: pointer;
}

.qty-input {
  width: 56px;
  border: 1px solid #d1d5db;
  border-radius: 6px;
  padding: 4px 6px;
  font-size: 12px;
  text-align: center;
}

.link-btn.danger {
  color: #dc2626;
}

.cart-total {
  margin: 6px 0 0;
  text-align: right;
}
</style>
