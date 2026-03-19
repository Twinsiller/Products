import axios from 'axios'

const API_BASE = 'http://localhost:8080'

export const api = axios.create({
  baseURL: API_BASE,
})

/** URL изображения товара (GET; 404, если фото нет). */
export function productImageUrl(productId: number): string {
  return `${API_BASE}/v1/products/${productId}/image`
}

api.interceptors.request.use((config) => {
  const userId = localStorage.getItem('currentUserId')
  const token = localStorage.getItem('authToken')
  if (userId) {
    config.headers['X-User-Id'] = userId
  }
  if (token) {
    config.headers.Authorization = `Bearer ${token}`
  }
  return config
})

export interface User {
  id: number
  name: string
  role: string
}

export interface CreateUserDto {
  name: string
}

export interface AuthResponse {
  user: User
  token: string
}

export interface Category {
  id: number
  name: string
}

export interface CreateCategoryDto {
  name: string
}

export interface Manufacturer {
  id: number
  name: string
}

export interface Product {
  id: number
  name: string
  category_id?: number | null
  manufacturer_id?: number | null
  barcode?: string | null
  default_price: number
}

export interface CreateProductDto {
  name: string
  category_id?: number | null
  manufacturer_id?: number | null
  barcode?: string | null
  default_price: number
}

export interface Order {
  id: number
  cashier_id: number
  created_at: string
  total_amount: number
}

