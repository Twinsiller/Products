<template>
  <section class="card">
    <h2>Вход</h2>
    <p class="muted">
      Войдите в систему покупателя по имени и паролю.
    </p>

    <form class="form" @submit.prevent="login">
      <label class="field">
        <span>Имя покупателя</span>
        <input
          v-model="name"
          type="text"
          required
          placeholder="Например: Иван"
        />
      </label>

      <label class="field">
        <span>Пароль</span>
        <input
          v-model="password"
          type="password"
          required
          placeholder="Ваш пароль"
        />
      </label>

      <div class="actions">
        <button type="submit" class="primary" :disabled="loading">
          {{ loading ? 'Проверка...' : 'Войти' }}
        </button>
        <RouterLink to="/register" class="link">
          Зарегистрироваться
        </RouterLink>
      </div>
    </form>

    <div v-if="currentUser" class="current">
      <h4>Текущий пользователь</h4>
      <p>
        <strong>ID:</strong> {{ currentUser.id }}
        <span v-if="currentUser.name">· <strong>Имя:</strong> {{ currentUser.name }}</span>
        <span v-if="currentUser.role">· <strong>Роль:</strong> {{ currentUser.role }}</span>
      </p>
    </div>

    <p v-if="error" class="error">{{ error }}</p>
  </section>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { api, type AuthResponse, type User } from '../api/http'

const router = useRouter()

const name = ref<string>('')
const password = ref<string>('')
const loading = ref(false)
const error = ref<string | null>(null)

const currentUser = ref<{ id: number; name?: string; role?: string } | null>(null)

onMounted(() => {
  const storedId = localStorage.getItem('currentUserId')
  const storedName = localStorage.getItem('currentUserName') || undefined
  const storedRole = localStorage.getItem('currentUserRole') || undefined

  if (storedId) {
    currentUser.value = {
      id: Number(storedId),
      name: storedName || undefined,
      role: storedRole || undefined,
    }
  }
})

const setCurrentUser = (user: User) => {
  currentUser.value = {
    id: user.id,
    name: user.name,
    role: user.role,
  }
  localStorage.setItem('currentUserId', String(user.id))
  localStorage.setItem('currentUserName', user.name)
  localStorage.setItem('currentUserRole', user.role)
  if (user.gender) {
    localStorage.setItem('currentUserGender', user.gender)
  } else {
    localStorage.removeItem('currentUserGender')
  }
}

const login = async () => {
  if (!name.value || !password.value) return
  loading.value = true
  error.value = null
  try {
    const { data } = await api.post<AuthResponse>('/login', {
      name: name.value,
      password: password.value,
    })
    setCurrentUser(data.user)
    localStorage.setItem('authToken', data.token)
    window.dispatchEvent(new Event('auth-changed'))
    router.push({ name: 'recommendations' })
  } catch (e) {
    error.value = 'Неверное имя или пароль'
  } finally {
    loading.value = false
  }
}
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
  font-size: 20px;
}

.muted {
  margin: 0 0 16px;
  font-size: 13px;
  color: #6b7280;
}

.columns {
  display: none;
}

.column {
  border-top: 1px solid #e5e7eb;
  padding-top: 8px;
}

.column h3 {
  margin: 0 0 8px;
  font-size: 14px;
}

.form {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.field {
  display: flex;
  flex-direction: column;
  gap: 4px;
  font-size: 13px;
}

input {
  border-radius: 4px;
  border: 1px solid #d1d5db;
  padding: 8px 10px;
  font-size: 14px;
  background: #ffffff;
  color: #111827;
}

input::placeholder {
  color: #6b7280;
}

.actions {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-top: 4px;
}

.primary {
  border-radius: 4px;
  padding: 8px 14px;
  border: none;
  font-size: 14px;
  cursor: pointer;
  background: #111827;
  color: #f9fafb;
  font-weight: 500;
}

.primary:disabled {
  opacity: 0.65;
  cursor: default;
}

.current {
  margin-top: 12px;
  padding-top: 8px;
  border-top: 1px solid #e5e7eb;
  font-size: 13px;
}

.link {
  font-size: 13px;
  color: #1d4ed8;
  text-decoration: none;
}

.error {
  margin-top: 12px;
  font-size: 13px;
  color: #b91c1c;
}
</style>
