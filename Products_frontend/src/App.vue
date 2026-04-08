<template>
  <div class="app-root">
    <header class="app-header">
      <div>
        <h1 class="app-title">Персонализированный подбор товаров</h1>
        <p class="app-subtitle">
          Информационная система для работы с товарами, заказами и рекомендациями
        </p>
      </div>
      <div class="header-right">
        <nav class="app-nav">
          <RouterLink to="/" class="nav-link">Вход</RouterLink>
          <RouterLink to="/register" class="nav-link">Регистрация</RouterLink>
          <RouterLink to="/recommendations" class="nav-link">Рекомендации</RouterLink>
          <RouterLink to="/products" class="nav-link">Товары</RouterLink>
          <RouterLink to="/favourites" class="nav-link">Избранное</RouterLink>
          <RouterLink to="/orders" class="nav-link">Заказы</RouterLink>
          <RouterLink to="/categories" class="nav-link">Категории</RouterLink>
          <RouterLink to="/manufacturers" class="nav-link">Производители</RouterLink>
          <RouterLink v-if="isAdmin" to="/dishes" class="nav-link">Блюда</RouterLink>
          <RouterLink v-if="isAdmin" to="/users" class="nav-link">Пользователи</RouterLink>
        </nav>
        <div v-if="currentUser" class="user-block">
          <div class="user-pill" :class="{ 'user-pill-admin': isAdmin }">
            <span class="user-name">{{ currentUser.name }}</span>
            <span class="user-role">
              {{ isAdmin ? 'администратор' : 'покупатель' }}
            </span>
          </div>
          <button class="logout-btn" type="button" @click="logout">
            Выйти
          </button>
        </div>
      </div>
    </header>

    <main class="app-main">
      <RouterView />
    </main>
  </div>
</template>

<script setup lang="ts">
import { onMounted, onBeforeUnmount, ref } from 'vue'
import { useRouter, RouterLink, RouterView } from 'vue-router'

const router = useRouter()
const currentUser = ref<{ name: string; role: string } | null>(null)
const isAdmin = ref(false)

const syncFromStorage = () => {
  const name = localStorage.getItem('currentUserName')
  const role = localStorage.getItem('currentUserRole') || 'user'
  if (name) {
    currentUser.value = { name, role }
    isAdmin.value = role === 'admin'
  } else {
    currentUser.value = null
    isAdmin.value = false
  }
}

const logout = () => {
  localStorage.removeItem('currentUserId')
  localStorage.removeItem('currentUserName')
  localStorage.removeItem('currentUserRole')
  localStorage.removeItem('currentUserGender')
  localStorage.removeItem('authToken')
  syncFromStorage()
  router.push({ name: 'login' })
}

onMounted(() => {
  syncFromStorage()
  window.addEventListener('auth-changed', syncFromStorage)
})

onBeforeUnmount(() => {
  window.removeEventListener('auth-changed', syncFromStorage)
})
</script>

<style scoped>
.app-root {
  min-height: 100vh;
  display: flex;
  flex-direction: column;
  background: #f3f4f6;
  color: #111827;
  font-family: system-ui, -apple-system, BlinkMacSystemFont, 'Segoe UI', sans-serif;
}

.app-header {
  padding: 12px 24px;
  border-bottom: 1px solid #e5e7eb;
  background: #ffffff;
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.app-title {
  margin: 0;
  font-size: 18px;
  font-weight: 500;
}

.app-subtitle {
  margin: 2px 0 0;
  font-size: 12px;
  color: #6b7280;
}

.app-nav {
  display: flex;
  flex-wrap: wrap;
  gap: 4px;
}

.header-right {
  display: flex;
  align-items: center;
  gap: 10px;
}

.nav-link {
  padding: 6px 12px;
  border-radius: 4px;
  font-size: 13px;
  text-decoration: none;
  color: #374151;
  border: 1px solid transparent;
}

.nav-link:hover {
  border-color: #d1d5db;
  background: #f9fafb;
}

.nav-link.router-link-active {
  border-color: #3b82f6;
  color: #1d4ed8;
  background: #eff6ff;
}

.user-pill {
  display: flex;
  flex-direction: column;
  align-items: flex-end;
  padding: 4px 8px;
  border-radius: 6px;
  border: 1px solid #e5e7eb;
  font-size: 11px;
  background: #f9fafb;
}

.user-pill-admin {
  border-color: #f97316;
  background: #fff7ed;
}

.user-name {
  font-weight: 500;
}

.user-role {
  color: #6b7280;
}

.user-block {
  display: flex;
  align-items: center;
  gap: 6px;
}

.logout-btn {
  border-radius: 4px;
  padding: 4px 10px;
  border: 1px solid #e5e7eb;
  font-size: 12px;
  cursor: pointer;
  background: #ffffff;
  color: #111827;
}

.app-main {
  flex: 1;
  padding: 16px 24px 24px;
  max-width: 1100px;
  margin: 0 auto;
  width: 100%;
  box-sizing: border-box;
}

@media (max-width: 768px) {
  .app-header {
    padding-inline: 16px;
    flex-direction: column;
    align-items: flex-start;
  }

  .app-main {
    padding-inline: 16px;
  }
}
</style>
