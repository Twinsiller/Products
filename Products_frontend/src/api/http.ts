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
  gender?: string
}

export interface CreateUserDto {
  name: string
  gender?: string
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
  calories_kcal?: number
  protein_g?: number
  fat_g?: number
  carbs_g?: number
}

export interface CreateProductDto {
  name: string
  category_id?: number | null
  manufacturer_id?: number | null
  barcode?: string | null
  default_price: number
  calories_kcal?: number
  protein_g?: number
  fat_g?: number
  carbs_g?: number
}

export interface MissingIngredient {
  needed_ingredient?: string
  needed_category_id: number
  needed_category_name: string
  qty_short: number
  suggestions: Product[]
}

export interface DishCategoryRequirement {
  id: number
  dish_id: number
  category_id?: number | null
  ingredient_name: string
  quantity: number
  note?: string
  category?: Category
}

export interface DishProduct {
  id: number
  dish_id: number
  product_id: number
  quantity: number
  price_per_unit?: number
  discount?: number
  product?: Product
}

export interface Dish {
  id: number
  name: string
  products?: DishProduct[]
  category_requirements?: DishCategoryRequirement[]
}

export interface OrderMealRecommendation {
  dish: {
    id: number
    name: string
    category_requirements?: DishCategoryRequirement[]
    products?: Array<{
      product_id: number
      quantity: number
      product?: Product
    }>
  }
  score: number
  makeable: boolean
  matched_items: number
  required_items: number
  match_ratio: number
  total_calories_kcal: number
  total_protein_g: number
  total_fat_g: number
  total_carbs_g: number
  missing?: MissingIngredient[]
  meal_target_kcal: number
}

export interface Order {
  id: number
  cashier_id: number
  created_at: string
  total_amount: number
}

export interface FinalRecommendationItem {
  recipe_id: number
  recipe_name: string
  score: number
  kcal?: number
  protein_g?: number
  fat_g?: number
  carbs_g?: number
}

export interface FinalRecommendationsResponse {
  precision_at_5: number
  recommendations: FinalRecommendationItem[]
}

export interface ProductRecommendationDishRef {
  dish_id: number
  dish_name: string
  dish_score: number
  missing_ingredients_estimate: number
}

export interface ProductRecommendationItem {
  product: Product
  score: number
  cb_score?: number
  cf_score?: number
  meal_score?: number
  recency_score?: number
  reason?: string
  linked_dishes?: ProductRecommendationDishRef[]
}

