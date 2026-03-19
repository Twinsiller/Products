<template>
  <section class="card">
    <h2>Регистрация покупателя</h2>
    <p class="muted">
      Создайте учётную запись покупателя, чтобы получать персональные рекомендации и управлять покупками.
    </p>

    <form class="form" @submit.prevent="registerUser">
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
          minlength="4"
          placeholder="Минимум 4 символа"
        />
      </label>

      <div class="actions">
        <button type="submit" class="primary" :disabled="loading">
          {{ loading ? 'Создание...' : 'Зарегистрировать и войти' }}
        </button>
        <RouterLink to="/" class="link">У меня уже есть аккаунт</RouterLink>
      </div>
    </form>

    <p v-if="error" class="error">{{ error }}</p>
  </section>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { api, type AuthResponse, type CreateUserDto, type User } from '../api/http'

const router = useRouter()

const name = ref('')
const password = ref('')
const loading = ref(false)
const error = ref<string | null>(null)

const setCurrentUser = (user: User) => {
  localStorage.setItem('currentUserId', String(user.id))
  localStorage.setItem('currentUserName', user.name)
  localStorage.setItem('currentUserRole', user.role)
}

const registerUser = async () => {
  if (!name.value || !password.value) return
  loading.value = true
  error.value = null
  try {
    const payload: CreateUserDto & { password: string } = {
      name: name.value,
      password: password.value,
    }
    const { data } = await api.post<AuthResponse>('/register', payload)
    setCurrentUser(data.user)
    localStorage.setItem('authToken', data.token)
    window.dispatchEvent(new Event('auth-changed'))
    router.push({ name: 'recommendations' })
  } catch (e) {
    error.value = 'Не удалось зарегистрировать пользователя'
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
  margin-top: 8px;
  display: flex;
  align-items: center;
  gap: 10px;
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

