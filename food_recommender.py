"""
Гибридная система рекомендаций продуктов питания
=================================================
Архитектура:
  1. Content-Based (cb)   — близость КБЖУ товара к профилю пользователя
  2. Collaborative (cf)   — частота и давность покупок (item-popularity + user-history)
  3. Meal-Aware (meal)    — буст товаров из рекомендованных блюд
  4. Recency penalty      — понижение недавно купленных (настраиваемо)

Итоговая формула:
  final_score = w_cb * cb_score + w_cf * cf_score + w_meal * meal_score
                - w_recency * recency_score
"""

from __future__ import annotations

import math
from collections import defaultdict
from datetime import datetime, timezone
from typing import Any

# ---------------------------------------------------------------------------
# Типы данных (TypedDict-style, без сторонних зависимостей)
# ---------------------------------------------------------------------------

Product = dict[str, Any]   # {id, name, calories_kcal, protein_g, fat_g, carbs_g,
                            #  category_id, price, availability?}
Order   = dict[str, Any]   # {user_id, order_id, created_at, items:[{product_id, quantity}]}
Dish    = dict[str, Any]   # {dish_id, name, ingredients:[{product_id, required_qty}]}
RecommendedDish = dict[str, Any]  # {dish_id, score}
UserProfile = dict[str, Any]      # {target_kcal, target_protein, target_fat,
                                  #  target_carbs, goal}

# ---------------------------------------------------------------------------
# 1. ПОСТРОЕНИЕ ПРИЗНАКОВ ТОВАРА
# ---------------------------------------------------------------------------

KBZHU_KEYS = ("calories_kcal", "protein_g", "fat_g", "carbs_g")
KBZHU_FALLBACK = 0.0          # значение если поле отсутствует


def build_product_features(product: Product) -> dict[str, float]:
    """
    Возвращает вектор КБЖУ + метаданные для одного товара.
    Отсутствующие поля заполняются KBZHU_FALLBACK.
    """
    return {
        "calories_kcal": float(product.get("calories_kcal") or KBZHU_FALLBACK),
        "protein_g":     float(product.get("protein_g")     or KBZHU_FALLBACK),
        "fat_g":         float(product.get("fat_g")          or KBZHU_FALLBACK),
        "carbs_g":       float(product.get("carbs_g")        or KBZHU_FALLBACK),
        "price":         float(product.get("price")          or 0.0),
    }


def build_product_index(products: list[Product]) -> dict[int, Product]:
    """product_id → product dict. Фильтрует недоступные товары."""
    return {
        p["id"]: p
        for p in products
        if p.get("availability", True)   # если поля нет — считаем доступным
    }


# ---------------------------------------------------------------------------
# 2. ПОСТРОЕНИЕ ПРОФИЛЯ ПОЛЬЗОВАТЕЛЯ (target КБЖУ)
# ---------------------------------------------------------------------------

def build_user_kbzhu_profile(
    user_profile: UserProfile | None,
    orders: list[Order],
    product_index: dict[int, Product],
) -> dict[str, float]:
    """
    Возвращает целевой вектор КБЖУ пользователя.

    Приоритет:
      1) явный user_profile (target_*)
      2) среднее по купленным товарам (из истории заказов)
      3) среднепопуляционные нормы (fallback)
    """
    if user_profile:
        profile = {
            "calories_kcal": float(user_profile.get("target_kcal")     or 0),
            "protein_g":     float(user_profile.get("target_protein")   or 0),
            "fat_g":         float(user_profile.get("target_fat")        or 0),
            "carbs_g":       float(user_profile.get("target_carbs")      or 0),
        }
        if all(v > 0 for v in profile.values()):
            return profile

    # Считаем средние КБЖУ по истории покупок
    totals: dict[str, float] = defaultdict(float)
    count = 0
    for order in orders:
        for item in order.get("items", []):
            pid = item["product_id"]
            if pid in product_index:
                feat = build_product_features(product_index[pid])
                for k in KBZHU_KEYS:
                    totals[k] += feat[k] * item.get("quantity", 1)
                count += item.get("quantity", 1)

    if count > 0:
        return {k: totals[k] / count for k in KBZHU_KEYS}

    # Среднепопуляционные нормы (ВОЗ ~2000 ккал/день, 1 приём)
    return {
        "calories_kcal": 667.0,
        "protein_g":     25.0,
        "fat_g":         22.0,
        "carbs_g":       83.0,
    }


# ---------------------------------------------------------------------------
# 3. НОРМАЛИЗАЦИЯ
# ---------------------------------------------------------------------------

def _minmax_normalize(
    values: dict[int, float],
    eps: float = 1e-9,
) -> dict[int, float]:
    """Min-max нормализация словаря {id: score} → [0, 1]."""
    if not values:
        return {}
    lo = min(values.values())
    hi = max(values.values())
    span = hi - lo
    if span < eps:
        return {k: 1.0 for k in values}
    return {k: (v - lo) / span for k, v in values.items()}


# ---------------------------------------------------------------------------
# 4. CONTENT-BASED СКОРИНГ (по КБЖУ)
# ---------------------------------------------------------------------------

def score_content_based(
    product_ids: list[int],
    product_index: dict[int, Product],
    user_kbzhu: dict[str, float],
    weights: dict[str, float] | None = None,
) -> dict[int, float]:
    """
    Для каждого товара считает «близость» КБЖУ к профилю пользователя
    через взвешенное L2-расстояние (инвертированное → чем ближе, тем выше).

    weights — важность каждого макронутриента (по умолчанию равные).
    Возвращает нормализованный score [0, 1].
    """
    if weights is None:
        weights = {"calories_kcal": 1.0, "protein_g": 1.0,
                   "fat_g": 1.0, "carbs_g": 1.0}

    raw: dict[int, float] = {}
    for pid in product_ids:
        feat = build_product_features(product_index[pid])
        dist_sq = sum(
            weights.get(k, 1.0) * (feat[k] - user_kbzhu.get(k, 0.0)) ** 2
            for k in KBZHU_KEYS
        )
        # Инвертируем: меньше расстояние → больше score
        raw[pid] = 1.0 / (1.0 + math.sqrt(dist_sq))

    return _minmax_normalize(raw)


# ---------------------------------------------------------------------------
# 5. COLLABORATIVE / ИСТОРИЯ ПОКУПОК
# ---------------------------------------------------------------------------

def build_purchase_history(
    user_id: int,
    orders: list[Order],
) -> dict[int, dict[str, Any]]:
    """
    Возвращает {product_id: {total_qty, last_bought_at}} для заданного user_id.
    """
    history: dict[int, dict[str, Any]] = defaultdict(
        lambda: {"total_qty": 0, "last_bought_at": None}
    )
    for order in orders:
        if order["user_id"] != user_id:
            continue
        ts = _parse_ts(order.get("created_at"))
        for item in order.get("items", []):
            pid = item["product_id"]
            history[pid]["total_qty"] += item.get("quantity", 1)
            prev = history[pid]["last_bought_at"]
            if ts and (prev is None or ts > prev):
                history[pid]["last_bought_at"] = ts
    return dict(history)


def score_collaborative(
    product_ids: list[int],
    purchase_history: dict[int, dict[str, Any]],
    all_orders: list[Order],
    decay_days: float = 30.0,
    popularity_weight: float = 0.3,
) -> dict[int, float]:
    """
    Смесь:
      - персональная частота покупки товара (с time-decay)
      - глобальная популярность товара среди всех пользователей

    decay_days — период полураспада: покупка N дней назад весит exp(-ln2*N/decay_days).
    popularity_weight — доля глобальной популярности в итоговом cf-score.
    """
    now = datetime.now(timezone.utc)

    # Глобальная популярность
    global_counts: dict[int, int] = defaultdict(int)
    for order in all_orders:
        for item in order.get("items", []):
            global_counts[item["product_id"]] += item.get("quantity", 1)

    raw: dict[int, float] = {}
    for pid in product_ids:
        # Персональный сигнал
        personal = 0.0
        if pid in purchase_history:
            info = purchase_history[pid]
            qty = info["total_qty"]
            last = info["last_bought_at"]
            if last:
                delta_days = (now - last).total_seconds() / 86400
                decay = math.exp(-math.log(2) * delta_days / max(decay_days, 1))
            else:
                decay = 0.5   # нет даты — умеренный вес
            personal = qty * decay

        # Глобальная популярность (нормализуем позже)
        popularity = global_counts.get(pid, 0)

        raw[pid] = (1 - popularity_weight) * personal + popularity_weight * popularity

    return _minmax_normalize(raw)


def score_recency(
    product_ids: list[int],
    purchase_history: dict[int, dict[str, Any]],
    decay_days: float = 14.0,
) -> dict[int, float]:
    """
    «Штраф за недавнюю покупку»: чем свежее покупка, тем выше штраф [0, 1].
    Вычитается из итогового score (с весом w_recency).
    """
    now = datetime.now(timezone.utc)
    raw: dict[int, float] = {}
    for pid in product_ids:
        info = purchase_history.get(pid)
        if info and info.get("last_bought_at"):
            delta_days = (now - info["last_bought_at"]).total_seconds() / 86400
            raw[pid] = math.exp(-math.log(2) * delta_days / max(decay_days, 1))
        else:
            raw[pid] = 0.0
    return raw   # уже в [0,1] по природе exp, не нормализуем


# ---------------------------------------------------------------------------
# 6. MEAL-AWARE СКОРИНГ
# ---------------------------------------------------------------------------

def build_dish_index(dishes: list[Dish]) -> dict[int, Dish]:
    return {d["dish_id"]: d for d in dishes}


def score_meal_aware(
    product_ids: list[int],
    recommended_dishes: list[RecommendedDish],
    dish_index: dict[int, Dish],
    purchase_history: dict[int, dict[str, Any]],
    missing_ingredient_bonus: float = 1.5,
) -> dict[int, float]:
    """
    Для каждого товара считает сумму dish_score * ingredient_weight по всем блюдам,
    где товар является ингредиентом.

    missing_ingredient_bonus — множитель, если ингредиент ещё не куплен
    (т.е. пользователю нужно его докупить).

    Возвращает нормализованный score [0, 1].
    """
    if not recommended_dishes or not dish_index:
        return {pid: 0.0 for pid in product_ids}

    # Суммарный required_qty по каждому product_id в рекомендованных блюдах
    product_dish_score: dict[int, float] = defaultdict(float)

    for rd in recommended_dishes:
        dish = dish_index.get(rd["dish_id"])
        if not dish:
            continue
        dish_score = float(rd.get("score", 1.0))
        ingredients = dish.get("ingredients", [])
        total_qty = sum(float(i.get("required_qty", 1)) for i in ingredients) or 1.0

        for ing in ingredients:
            pid = ing["product_id"]
            qty = float(ing.get("required_qty", 1))
            weight = qty / total_qty  # нормализованная доля ингредиента в блюде
            bonus = (
                missing_ingredient_bonus
                if pid not in purchase_history
                else 1.0
            )
            product_dish_score[pid] += dish_score * weight * bonus

    raw = {pid: product_dish_score.get(pid, 0.0) for pid in product_ids}
    return _minmax_normalize(raw)


# ---------------------------------------------------------------------------
# 7. ИТОГОВЫЙ ГИБРИДНЫЙ СКОРИНГ
# ---------------------------------------------------------------------------

DEFAULT_WEIGHTS = {
    "w_cb":      0.30,   # content-based (КБЖУ)
    "w_cf":      0.35,   # collaborative (история)
    "w_meal":    0.25,   # meal-aware
    "w_recency": 0.10,   # штраф за недавние покупки
}


def hybrid_score(
    product_ids: list[int],
    cb_scores:      dict[int, float],
    cf_scores:      dict[int, float],
    meal_scores:    dict[int, float],
    recency_scores: dict[int, float],
    weights: dict[str, float] | None = None,
) -> dict[int, float]:
    """
    final_score = w_cb * cb_score + w_cf * cf_score
                + w_meal * meal_score - w_recency * recency_score

    Клиппируем результат в [0, 1].
    """
    w = weights or DEFAULT_WEIGHTS
    scores: dict[int, float] = {}
    for pid in product_ids:
        s = (
            w["w_cb"]      * cb_scores.get(pid, 0.0)
            + w["w_cf"]    * cf_scores.get(pid, 0.0)
            + w["w_meal"]  * meal_scores.get(pid, 0.0)
            - w["w_recency"] * recency_scores.get(pid, 0.0)
        )
        scores[pid] = max(0.0, min(1.0, s))
    return scores


# ---------------------------------------------------------------------------
# 8. TOP-N РЕКОМЕНДАЦИИ
# ---------------------------------------------------------------------------

def top_n_recommendations(
    user_id: int,
    products: list[Product],
    orders: list[Order],
    dishes: list[Dish],
    recommended_dishes: list[RecommendedDish],
    user_profile: UserProfile | None = None,
    n: int = 10,
    exclude_recent_days: int | None = 7,
    weights: dict[str, float] | None = None,
    cf_decay_days: float = 30.0,
    recency_decay_days: float = 14.0,
    cb_kbzhu_weights: dict[str, float] | None = None,
    popularity_weight: float = 0.3,
    missing_ingredient_bonus: float = 1.5,
) -> list[dict[str, Any]]:
    """
    Основная функция: возвращает топ-N рекомендаций для пользователя.

    exclude_recent_days — исключать товары, купленные за последние N дней.
                          None — не исключать.

    Возвращает список:
      [{product_id, name, final_score, cb_score, cf_score,
        meal_score, recency_score, price}]
    """
    product_index = build_product_index(products)
    if not product_index:
        return []

    user_orders = [o for o in orders if o["user_id"] == user_id]
    purchase_history = build_purchase_history(user_id, orders)
    dish_index = build_dish_index(dishes)

    # --- Определяем пул товаров для скоринга ---
    now = datetime.now(timezone.utc)
    candidate_ids: list[int] = []
    for pid in product_index:
        if exclude_recent_days is not None:
            info = purchase_history.get(pid)
            if info and info.get("last_bought_at"):
                delta = (now - info["last_bought_at"]).total_seconds() / 86400
                if delta < exclude_recent_days:
                    continue  # слишком свежая покупка
        candidate_ids.append(pid)

    if not candidate_ids:
        return []

    # --- Пользовательский профиль КБЖУ ---
    user_kbzhu = build_user_kbzhu_profile(user_profile, user_orders, product_index)

    # --- Скоринг по компонентам ---
    cb_scores      = score_content_based(candidate_ids, product_index,
                                          user_kbzhu, cb_kbzhu_weights)
    cf_scores      = score_collaborative(candidate_ids, purchase_history,
                                          orders, cf_decay_days, popularity_weight)
    meal_scores    = score_meal_aware(candidate_ids, recommended_dishes,
                                       dish_index, purchase_history,
                                       missing_ingredient_bonus)
    recency_scores = score_recency(candidate_ids, purchase_history,
                                    recency_decay_days)

    # --- Fallback: если мало истории — усилить CB и meal ---
    _weights = dict(weights or DEFAULT_WEIGHTS)
    if len(user_orders) < 3:
        # Перераспределяем вес от cf к cb
        extra = _weights["w_cf"] * 0.5
        _weights["w_cf"] -= extra
        _weights["w_cb"] += extra

    # --- Итоговый скоринг ---
    final_scores = hybrid_score(
        candidate_ids, cb_scores, cf_scores,
        meal_scores, recency_scores, _weights,
    )

    # --- Сортировка и формирование ответа ---
    top = sorted(final_scores.items(), key=lambda x: x[1], reverse=True)[:n]

    result = []
    for pid, score in top:
        p = product_index[pid]
        result.append({
            "product_id":    pid,
            "name":          p.get("name", ""),
            "final_score":   round(score, 4),
            "cb_score":      round(cb_scores.get(pid, 0.0), 4),
            "cf_score":      round(cf_scores.get(pid, 0.0), 4),
            "meal_score":    round(meal_scores.get(pid, 0.0), 4),
            "recency_score": round(recency_scores.get(pid, 0.0), 4),
            "price":         p.get("price"),
        })
    return result


# ---------------------------------------------------------------------------
# 9. ОЦЕНКА КАЧЕСТВА
# ---------------------------------------------------------------------------

def precision_at_k(
    recommended: list[int],
    relevant: set[int],
    k: int,
) -> float:
    """Доля релевантных среди топ-k рекомендаций."""
    top_k = recommended[:k]
    if not top_k:
        return 0.0
    hits = sum(1 for pid in top_k if pid in relevant)
    return hits / len(top_k)


def recall_at_k(
    recommended: list[int],
    relevant: set[int],
    k: int,
) -> float:
    """Доля найденных релевантных из всего множества релевантных."""
    if not relevant:
        return 0.0
    top_k = recommended[:k]
    hits = sum(1 for pid in top_k if pid in relevant)
    return hits / len(relevant)


def coverage(
    recommended_sets: list[list[int]],
    catalog_ids: set[int],
) -> float:
    """Доля уникальных товаров каталога, попавших хотя бы в одну рекомендацию."""
    if not catalog_ids:
        return 0.0
    all_recommended = set(pid for recs in recommended_sets for pid in recs)
    return len(all_recommended & catalog_ids) / len(catalog_ids)


def evaluate_recommender(
    user_ids: list[int],
    products: list[Product],
    orders: list[Order],
    dishes: list[Dish],
    recommended_dishes_map: dict[int, list[RecommendedDish]],
    user_profiles: dict[int, UserProfile],
    k: int = 10,
    test_orders: list[Order] | None = None,
) -> dict[str, float]:
    """
    Оценивает систему на тестовой выборке заказов.
    test_orders — «будущие» заказы (ground truth).
    Если не переданы — используем leave-one-last-out на существующих.
    """
    if test_orders is None:
        # Leave-one-last-out: последний заказ каждого пользователя — тест
        last_orders: dict[int, Order] = {}
        train_orders: list[Order] = []
        for o in sorted(orders, key=lambda x: _parse_ts(x["created_at"]) or datetime.min):
            uid = o["user_id"]
            if uid in last_orders:
                train_orders.append(last_orders[uid])
            last_orders[uid] = o
        test_orders = list(last_orders.values())
        orders_for_train = train_orders
    else:
        orders_for_train = orders

    p_scores, r_scores = [], []
    recommended_sets: list[list[int]] = []

    for uid in user_ids:
        user_test = [o for o in test_orders if o["user_id"] == uid]
        relevant = {i["product_id"] for o in user_test for i in o.get("items", [])}
        if not relevant:
            continue

        recs = top_n_recommendations(
            user_id=uid,
            products=products,
            orders=orders_for_train,
            dishes=dishes,
            recommended_dishes=recommended_dishes_map.get(uid, []),
            user_profile=user_profiles.get(uid),
            n=k,
            exclude_recent_days=None,  # при оценке не исключаем
        )
        rec_ids = [r["product_id"] for r in recs]
        recommended_sets.append(rec_ids)

        p_scores.append(precision_at_k(rec_ids, relevant, k))
        r_scores.append(recall_at_k(rec_ids, relevant, k))

    catalog_ids = {p["id"] for p in products if p.get("availability", True)}
    cov = coverage(recommended_sets, catalog_ids)

    return {
        f"precision@{k}": round(sum(p_scores) / len(p_scores), 4) if p_scores else 0.0,
        f"recall@{k}":    round(sum(r_scores) / len(r_scores), 4) if r_scores else 0.0,
        "coverage":       round(cov, 4),
        "n_users_evaluated": len(p_scores),
    }


# ---------------------------------------------------------------------------
# ВСПОМОГАТЕЛЬНЫЕ ФУНКЦИИ
# ---------------------------------------------------------------------------

def _parse_ts(ts: Any) -> datetime | None:
    """Парсит ISO-строку или datetime. Возвращает timezone-aware datetime."""
    if ts is None:
        return None
    if isinstance(ts, datetime):
        return ts if ts.tzinfo else ts.replace(tzinfo=timezone.utc)
    try:
        dt = datetime.fromisoformat(str(ts).replace("Z", "+00:00"))
        return dt if dt.tzinfo else dt.replace(tzinfo=timezone.utc)
    except ValueError:
        return None


# ---------------------------------------------------------------------------
# ПРИМЕР ВХОДНЫХ ДАННЫХ И ВЫЗОВ
# ---------------------------------------------------------------------------

if __name__ == "__main__":
    from datetime import timedelta

    now = datetime.now(timezone.utc)

    # --- Каталог товаров ---
    products: list[Product] = [
        {"id": 1,  "name": "Куриная грудка",   "calories_kcal": 165, "protein_g": 31, "fat_g": 3.6, "carbs_g": 0,   "category_id": 1, "price": 350, "availability": True},
        {"id": 2,  "name": "Гречка",            "calories_kcal": 343, "protein_g": 13, "fat_g": 3.4, "carbs_g": 71,  "category_id": 2, "price": 80,  "availability": True},
        {"id": 3,  "name": "Яйца (10 шт)",      "calories_kcal": 155, "protein_g": 13, "fat_g": 11,  "carbs_g": 1.1, "category_id": 1, "price": 120, "availability": True},
        {"id": 4,  "name": "Лосось",             "calories_kcal": 208, "protein_g": 20, "fat_g": 13,  "carbs_g": 0,   "category_id": 1, "price": 680, "availability": True},
        {"id": 5,  "name": "Овсянка",            "calories_kcal": 389, "protein_g": 17, "fat_g": 7,   "carbs_g": 66,  "category_id": 2, "price": 75,  "availability": True},
        {"id": 6,  "name": "Творог 5%",          "calories_kcal": 121, "protein_g": 18, "fat_g": 5,   "carbs_g": 3.3, "category_id": 3, "price": 110, "availability": True},
        {"id": 7,  "name": "Брокколи замор.",    "calories_kcal": 35,  "protein_g": 2.8,"fat_g": 0.4, "carbs_g": 6.6, "category_id": 4, "price": 95,  "availability": True},
        {"id": 8,  "name": "Оливковое масло",    "calories_kcal": 884, "protein_g": 0,  "fat_g": 100, "carbs_g": 0,   "category_id": 5, "price": 390, "availability": True},
        {"id": 9,  "name": "Миндаль",            "calories_kcal": 579, "protein_g": 21, "fat_g": 50,  "carbs_g": 22,  "category_id": 6, "price": 450, "availability": True},
        {"id": 10, "name": "Рис бурый",          "calories_kcal": 370, "protein_g": 8,  "fat_g": 3,   "carbs_g": 77,  "category_id": 2, "price": 130, "availability": True},
        {"id": 11, "name": "Чечевица",           "calories_kcal": 352, "protein_g": 24, "fat_g": 1.1, "carbs_g": 60,  "category_id": 2, "price": 100, "availability": True},
        {"id": 12, "name": "Греческий йогурт",   "calories_kcal": 97,  "protein_g": 10, "fat_g": 5,   "carbs_g": 4,   "category_id": 3, "price": 140, "availability": True},
        {"id": 13, "name": "Тунец конс.",        "calories_kcal": 132, "protein_g": 29, "fat_g": 0.9, "carbs_g": 0,   "category_id": 1, "price": 160, "availability": True},
        {"id": 14, "name": "Шпинат",             "calories_kcal": 23,  "protein_g": 2.9,"fat_g": 0.4, "carbs_g": 3.6, "category_id": 4, "price": 85,  "availability": True},
        {"id": 15, "name": "Батончик протеин.",  "calories_kcal": 380, "protein_g": 30, "fat_g": 10,  "carbs_g": 45,  "category_id": 6, "price": 220, "availability": False},  # недоступен
    ]

    # --- История заказов ---
    orders: list[Order] = [
        {
            "user_id": 42, "order_id": 1001,
            "created_at": (now - timedelta(days=60)).isoformat(),
            "items": [{"product_id": 1, "quantity": 2}, {"product_id": 2, "quantity": 1}],
        },
        {
            "user_id": 42, "order_id": 1002,
            "created_at": (now - timedelta(days=30)).isoformat(),
            "items": [{"product_id": 3, "quantity": 1}, {"product_id": 6, "quantity": 2},
                      {"product_id": 7, "quantity": 1}],
        },
        {
            "user_id": 42, "order_id": 1003,
            "created_at": (now - timedelta(days=5)).isoformat(),  # свежая покупка
            "items": [{"product_id": 1, "quantity": 1}, {"product_id": 10, "quantity": 1}],
        },
        # Другие пользователи (для глобальной популярности)
        {"user_id": 99, "order_id": 2001, "created_at": (now - timedelta(days=10)).isoformat(),
         "items": [{"product_id": 4, "quantity": 1}, {"product_id": 5, "quantity": 2}]},
        {"user_id": 77, "order_id": 3001, "created_at": (now - timedelta(days=3)).isoformat(),
         "items": [{"product_id": 4, "quantity": 1}, {"product_id": 11, "quantity": 1},
                   {"product_id": 14, "quantity": 2}]},
    ]

    # --- Блюда ---
    dishes: list[Dish] = [
        {
            "dish_id": 501, "name": "Боул с лососем",
            "ingredients": [
                {"product_id": 4,  "required_qty": 200},
                {"product_id": 10, "required_qty": 150},
                {"product_id": 14, "required_qty": 50},
                {"product_id": 8,  "required_qty": 10},
            ],
        },
        {
            "dish_id": 502, "name": "Куриный салат",
            "ingredients": [
                {"product_id": 1,  "required_qty": 150},
                {"product_id": 7,  "required_qty": 100},
                {"product_id": 3,  "required_qty": 50},
                {"product_id": 8,  "required_qty": 15},
            ],
        },
        {
            "dish_id": 503, "name": "Протеиновый завтрак",
            "ingredients": [
                {"product_id": 6,  "required_qty": 200},
                {"product_id": 5,  "required_qty": 80},
                {"product_id": 12, "required_qty": 100},
            ],
        },
    ]

    # --- Рекомендованные блюда для пользователя 42 ---
    recommended_dishes_for_user: list[RecommendedDish] = [
        {"dish_id": 501, "score": 0.92},
        {"dish_id": 503, "score": 0.74},
    ]

    # --- Профиль пользователя ---
    user_profile: UserProfile = {
        "target_kcal":    700,
        "target_protein": 40,
        "target_fat":     20,
        "target_carbs":   60,
        "goal":           "muscle_gain",
    }

    # =========================================================
    # ВЫЗОВ: top-10 рекомендаций для пользователя 42
    # =========================================================
    recommendations = top_n_recommendations(
        user_id=42,
        products=products,
        orders=orders,
        dishes=dishes,
        recommended_dishes=recommended_dishes_for_user,
        user_profile=user_profile,
        n=10,
        exclude_recent_days=7,          # исключаем купленное за 7 дней
        weights=DEFAULT_WEIGHTS,
        cf_decay_days=30.0,
        recency_decay_days=14.0,
        popularity_weight=0.3,
        missing_ingredient_bonus=1.5,
    )

    print("=" * 65)
    print(f"  TOP-10 рекомендаций для пользователя 42")
    print("=" * 65)
    header = f"{'#':>2}  {'Товар':<22} {'Score':>6} {'CB':>6} {'CF':>6} {'Meal':>6} {'Rec':>5} {'Цена':>6}"
    print(header)
    print("-" * 65)
    for i, r in enumerate(recommendations, 1):
        print(
            f"{i:>2}. {r['name']:<22} {r['final_score']:>6.4f} "
            f"{r['cb_score']:>6.4f} {r['cf_score']:>6.4f} "
            f"{r['meal_score']:>6.4f} {r['recency_score']:>5.4f} "
            f"{r['price']:>6}"
        )

    print()
    print("=" * 65)
    print("  ОЦЕНКА КАЧЕСТВА (leave-one-last-out, k=5)")
    print("=" * 65)
    metrics = evaluate_recommender(
        user_ids=[42, 99, 77],
        products=products,
        orders=orders,
        dishes=dishes,
        recommended_dishes_map={42: recommended_dishes_for_user, 99: [], 77: []},
        user_profiles={42: user_profile},
        k=5,
    )
    for key, val in metrics.items():
        print(f"  {key}: {val}")

    print()
    print("=" * 65)
    print("  ПАРАМЕТРЫ ДЛЯ ТЮНИНГА")
    print("=" * 65)
    tuning_params = {
        "weights": {
            "w_cb":      "0.30 — вес content-based (КБЖУ). ↑ если профиль точный",
            "w_cf":      "0.35 — вес collaborative. ↑ при богатой истории",
            "w_meal":    "0.25 — вес meal-aware. ↑ если блюдовые рекомендации надёжны",
            "w_recency": "0.10 — штраф за недавние покупки. ↑ чтобы сильнее ротировать",
        },
        "n":                      "10 — размер топа",
        "exclude_recent_days":    "7  — горизонт исключения (None = не исключать)",
        "cf_decay_days":          "30 — полупериод давности покупки (дни)",
        "recency_decay_days":     "14 — полупериод штрафа за недавнюю покупку (дни)",
        "popularity_weight":      "0.3 — доля глобальной популярности в cf-score",
        "missing_ingredient_bonus":"1.5 — множитель для недостающих ингредиентов блюда",
        "cb_kbzhu_weights": {
            "calories_kcal": "1.0",
            "protein_g":     "1.0 — ↑ для цели muscle_gain",
            "fat_g":         "1.0",
            "carbs_g":       "1.0 — ↑ для цели endurance",
        },
        "k (eval)": "5 — глубина оценки precision/recall",
    }
    import json
    print(json.dumps(tuning_params, ensure_ascii=False, indent=2))
