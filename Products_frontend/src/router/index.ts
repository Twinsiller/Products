import { createRouter, createWebHistory, type RouteRecordRaw } from 'vue-router'

import LoginView from '../views/LoginView.vue'
import RegisterView from '../views/RegisterView.vue'
import ProductsView from '../views/ProductsView.vue'
import OrdersView from '../views/OrdersView.vue'
import CategoriesView from '../views/CategoriesView.vue'
import ManufacturersView from '../views/ManufacturersView.vue'
import DishesView from '../views/DishesView.vue'
import RecommendationsView from '../views/RecommendationsView.vue'
import FavouritesView from '../views/FavouritesView.vue'
import UsersView from '../views/UsersView.vue'

const routes: RouteRecordRaw[] = [
  { path: '/', name: 'login', component: LoginView },
  { path: '/register', name: 'register', component: RegisterView },
  { path: '/recommendations', name: 'recommendations', component: RecommendationsView },
  { path: '/products', name: 'products', component: ProductsView },
  { path: '/favourites', name: 'favourites', component: FavouritesView },
  { path: '/orders', name: 'orders', component: OrdersView },
  { path: '/categories', name: 'categories', component: CategoriesView },
  { path: '/manufacturers', name: 'manufacturers', component: ManufacturersView },
  { path: '/dishes', name: 'dishes', component: DishesView },
  { path: '/users', name: 'users', component: UsersView },
]

export const router = createRouter({
  history: createWebHistory(),
  routes,
})
