from __future__ import annotations

import argparse
import json
import math
from collections import defaultdict
from datetime import datetime, timezone
from typing import Any

import pandas as pd
from sqlalchemy import create_engine, inspect, text

Product = dict[str, Any]
Order = dict[str, Any]
Dish = dict[str, Any]
RecommendedDish = dict[str, Any]
UserProfile = dict[str, Any]

KBZHU_KEYS = ("calories_kcal", "protein_g", "fat_g", "carbs_g")


def build_product_features(product: Product) -> dict[str, float]:
    return {
        "calories_kcal": float(product.get("calories_kcal") or 0.0),
        "protein_g": float(product.get("protein_g") or 0.0),
        "fat_g": float(product.get("fat_g") or 0.0),
        "carbs_g": float(product.get("carbs_g") or 0.0),
        "price": float(product.get("price") or 0.0),
    }


def build_product_index(products: list[Product]) -> dict[int, Product]:
    return {int(p["id"]): p for p in products if p.get("availability", True)}


def _minmax_normalize(values: dict[int, float], eps: float = 1e-9) -> dict[int, float]:
    if not values:
        return {}
    lo = min(values.values())
    hi = max(values.values())
    span = hi - lo
    if span < eps:
        return {k: 1.0 for k in values}
    return {k: (v - lo) / span for k, v in values.items()}


def _parse_ts(ts: Any) -> datetime | None:
    if ts is None:
        return None
    if isinstance(ts, datetime):
        return ts if ts.tzinfo else ts.replace(tzinfo=timezone.utc)
    try:
        dt = datetime.fromisoformat(str(ts).replace("Z", "+00:00"))
        return dt if dt.tzinfo else dt.replace(tzinfo=timezone.utc)
    except ValueError:
        return None


def build_user_kbzhu_profile(user_profile: UserProfile | None, orders: list[Order], product_index: dict[int, Product]) -> dict[str, float]:
    if user_profile:
        profile = {
            "calories_kcal": float(user_profile.get("target_kcal") or 0),
            "protein_g": float(user_profile.get("target_protein") or 0),
            "fat_g": float(user_profile.get("target_fat") or 0),
            "carbs_g": float(user_profile.get("target_carbs") or 0),
        }
        if all(v > 0 for v in profile.values()):
            return profile

    totals: dict[str, float] = defaultdict(float)
    count = 0
    for order in orders:
        for item in order.get("items", []):
            pid = int(item.get("product_id", 0))
            qty = int(item.get("quantity", 1) or 1)
            if pid in product_index:
                feat = build_product_features(product_index[pid])
                for k in KBZHU_KEYS:
                    totals[k] += feat[k] * qty
                count += qty
    if count > 0:
        return {k: totals[k] / count for k in KBZHU_KEYS}
    return {"calories_kcal": 667.0, "protein_g": 25.0, "fat_g": 22.0, "carbs_g": 83.0}


def score_content_based(product_ids: list[int], product_index: dict[int, Product], user_kbzhu: dict[str, float]) -> dict[int, float]:
    raw: dict[int, float] = {}
    for pid in product_ids:
        feat = build_product_features(product_index[pid])
        dist_sq = sum((feat[k] - user_kbzhu.get(k, 0.0)) ** 2 for k in KBZHU_KEYS)
        raw[pid] = 1.0 / (1.0 + math.sqrt(dist_sq))
    return _minmax_normalize(raw)


def build_purchase_history(user_id: int, orders: list[Order]) -> dict[int, dict[str, Any]]:
    history: dict[int, dict[str, Any]] = defaultdict(lambda: {"total_qty": 0, "last_bought_at": None})
    for order in orders:
        if int(order.get("user_id", -1)) != user_id:
            continue
        ts = _parse_ts(order.get("created_at"))
        for item in order.get("items", []):
            pid = int(item.get("product_id", 0))
            qty = int(item.get("quantity", 1) or 1)
            if pid <= 0:
                continue
            history[pid]["total_qty"] += qty
            prev = history[pid]["last_bought_at"]
            if ts and (prev is None or ts > prev):
                history[pid]["last_bought_at"] = ts
    return dict(history)


def score_collaborative(product_ids: list[int], purchase_history: dict[int, dict[str, Any]], all_orders: list[Order], decay_days: float = 30.0, popularity_weight: float = 0.3) -> dict[int, float]:
    now = datetime.now(timezone.utc)
    global_counts: dict[int, int] = defaultdict(int)
    for order in all_orders:
        for item in order.get("items", []):
            pid = int(item.get("product_id", 0))
            qty = int(item.get("quantity", 1) or 1)
            if pid > 0:
                global_counts[pid] += qty

    raw: dict[int, float] = {}
    for pid in product_ids:
        personal = 0.0
        if pid in purchase_history:
            info = purchase_history[pid]
            qty = float(info.get("total_qty", 0))
            last = info.get("last_bought_at")
            if last:
                delta_days = (now - last).total_seconds() / 86400
                decay = math.exp(-math.log(2) * delta_days / max(decay_days, 1))
            else:
                decay = 0.5
            personal = qty * decay
        popularity = float(global_counts.get(pid, 0))
        raw[pid] = (1 - popularity_weight) * personal + popularity_weight * popularity
    return _minmax_normalize(raw)


def score_recency(product_ids: list[int], purchase_history: dict[int, dict[str, Any]], decay_days: float = 14.0) -> dict[int, float]:
    now = datetime.now(timezone.utc)
    raw: dict[int, float] = {}
    for pid in product_ids:
        info = purchase_history.get(pid)
        if info and info.get("last_bought_at"):
            delta_days = (now - info["last_bought_at"]).total_seconds() / 86400
            raw[pid] = math.exp(-math.log(2) * delta_days / max(decay_days, 1))
        else:
            raw[pid] = 0.0
    return raw


def score_meal_aware(product_ids: list[int], recommended_dishes: list[RecommendedDish], dish_index: dict[int, Dish], purchase_history: dict[int, dict[str, Any]], missing_ingredient_bonus: float = 1.5) -> dict[int, float]:
    if not recommended_dishes or not dish_index:
        return {pid: 0.0 for pid in product_ids}
    product_dish_score: dict[int, float] = defaultdict(float)
    for rd in recommended_dishes:
        dish = dish_index.get(int(rd.get("dish_id", 0)))
        if not dish:
            continue
        dish_score = float(rd.get("score", 1.0))
        ingredients = dish.get("ingredients", [])
        total_qty = sum(float(i.get("required_qty", 1) or 1) for i in ingredients) or 1.0
        for ing in ingredients:
            pid = int(ing.get("product_id", 0))
            qty = float(ing.get("required_qty", 1) or 1)
            if pid <= 0:
                continue
            weight = qty / total_qty
            bonus = missing_ingredient_bonus if pid not in purchase_history else 1.0
            product_dish_score[pid] += dish_score * weight * bonus
    raw = {pid: product_dish_score.get(pid, 0.0) for pid in product_ids}
    return _minmax_normalize(raw)


DEFAULT_WEIGHTS = {"w_cb": 0.30, "w_cf": 0.35, "w_meal": 0.25, "w_recency": 0.10}


def top_n_recommendations(user_id: int, products: list[Product], orders: list[Order], dishes: list[Dish], recommended_dishes: list[RecommendedDish], user_profile: UserProfile | None = None, n: int = 10) -> list[dict[str, Any]]:
    product_index = build_product_index(products)
    if not product_index:
        return []

    user_orders = [o for o in orders if int(o.get("user_id", -1)) == user_id]
    purchase_history = build_purchase_history(user_id, orders)
    dish_index = {int(d["dish_id"]): d for d in dishes}
    candidate_ids = list(product_index.keys())
    user_kbzhu = build_user_kbzhu_profile(user_profile, user_orders, product_index)

    cb_scores = score_content_based(candidate_ids, product_index, user_kbzhu)
    cf_scores = score_collaborative(candidate_ids, purchase_history, orders)
    meal_scores = score_meal_aware(candidate_ids, recommended_dishes, dish_index, purchase_history)
    recency_scores = score_recency(candidate_ids, purchase_history)

    weights = dict(DEFAULT_WEIGHTS)
    if len(user_orders) < 3:
        extra = weights["w_cf"] * 0.5
        weights["w_cf"] -= extra
        weights["w_cb"] += extra

    scores: dict[int, float] = {}
    for pid in candidate_ids:
        s = (
            weights["w_cb"] * cb_scores.get(pid, 0.0)
            + weights["w_cf"] * cf_scores.get(pid, 0.0)
            + weights["w_meal"] * meal_scores.get(pid, 0.0)
            - weights["w_recency"] * recency_scores.get(pid, 0.0)
        )
        scores[pid] = max(0.0, min(1.0, s))

    top = sorted(scores.items(), key=lambda x: x[1], reverse=True)[:n]
    dish_by_id = {int(d.get("dish_id", 0)): d for d in dishes}
    ranked_dishes = sorted(recommended_dishes, key=lambda x: float(x.get("score", 0.0)), reverse=True)

    def dishes_for_product(pid: int) -> list[dict[str, Any]]:
        linked: list[dict[str, Any]] = []
        for rd in ranked_dishes:
            dish_id = int(rd.get("dish_id", 0))
            dish = dish_by_id.get(dish_id)
            if not dish:
                continue
            ingredients = dish.get("ingredients", [])
            ingredient_ids = {int(i.get("product_id", 0)) for i in ingredients}
            if pid not in ingredient_ids:
                continue
            missing_cnt = 0
            for ing in ingredients:
                ipid = int(ing.get("product_id", 0))
                if ipid not in purchase_history:
                    missing_cnt += 1
            linked.append(
                {
                    "dish_id": dish_id,
                    "dish_name": str(dish.get("name", "")),
                    "dish_score": round(float(rd.get("score", 0.0)), 6),
                    "missing_ingredients_estimate": missing_cnt,
                }
            )
            if len(linked) >= 3:
                break
        return linked

    out: list[dict[str, Any]] = []
    for pid, score in top:
        p = product_index[pid]
        cb = float(cb_scores.get(pid, 0.0))
        cf = float(cf_scores.get(pid, 0.0))
        meal = float(meal_scores.get(pid, 0.0))
        rec = float(recency_scores.get(pid, 0.0))
        linked = dishes_for_product(pid)
        # Короткое человекочитаемое объяснение — без отладочных чисел в скобках.
        reason_parts = []
        if cb > 0:
            reason_parts.append("подходит по балансу КБЖУ")
        if cf > 0:
            reason_parts.append("похоже на прошлые покупки")
        if meal > 0:
            reason_parts.append("сочетается с подходящими блюдами")
        reason = "Подходит: " + ", ".join(reason_parts) + "." if reason_parts else ""
        out.append(
            {
                "product_id": pid,
                "name": p.get("name", ""),
                "final_score": round(float(score), 6),
                "cb_score": round(cb, 6),
                "cf_score": round(cf, 6),
                "meal_score": round(meal, 6),
                "recency_score": round(rec, 6),
                "reason": reason,
                "linked_dishes": linked,
            }
        )
    return out


def _load_products(engine) -> list[Product]:
    q = text(
        """
        SELECT
          id, name,
          COALESCE(calories_kcal, 0) AS calories_kcal,
          COALESCE(protein_g, 0) AS protein_g,
          COALESCE(fat_g, 0) AS fat_g,
          COALESCE(carbs_g, 0) AS carbs_g,
          category_id,
          COALESCE(default_price, 0) AS price
        FROM products
        """
    )
    rows = pd.read_sql(q, engine)
    return rows.to_dict("records")


def _order_user_column(engine) -> str:
    cols = {c["name"] for c in inspect(engine).get_columns("orders")}
    if "user_id" in cols:
        return "user_id"
    if "cashier_id" in cols:
        return "cashier_id"
    raise RuntimeError("orders must have user_id or cashier_id")


def _load_orders(engine) -> list[Order]:
    user_col = _order_user_column(engine)
    q = text(
        f"""
        SELECT
          o.id AS order_id,
          o.{user_col} AS user_id,
          o.created_at,
          oi.product_id,
          oi.quantity
        FROM orders o
        JOIN order_items oi ON oi.order_id = o.id
        """
    )
    df = pd.read_sql(q, engine)
    grouped: dict[tuple[int, int, str], list[dict[str, Any]]] = defaultdict(list)
    for r in df.to_dict("records"):
        key = (int(r["user_id"]), int(r["order_id"]), str(r["created_at"]))
        grouped[key].append({"product_id": int(r["product_id"]), "quantity": int(r.get("quantity", 1) or 1)})
    out: list[Order] = []
    for (uid, oid, created), items in grouped.items():
        out.append({"user_id": uid, "order_id": oid, "created_at": created, "items": items})
    return out


def _load_dishes(engine) -> list[Dish]:
    q = text(
        """
        SELECT
          d.id AS dish_id,
          d.name AS dish_name,
          dp.product_id,
          dp.quantity
        FROM dishes d
        LEFT JOIN dish_products dp ON dp.dish_id = d.id
        """
    )
    df = pd.read_sql(q, engine)
    dishes: dict[int, Dish] = {}
    for r in df.to_dict("records"):
        did = int(r["dish_id"])
        if did not in dishes:
            dishes[did] = {"dish_id": did, "name": r.get("dish_name", ""), "ingredients": []}
        if pd.notna(r.get("product_id")):
            dishes[did]["ingredients"].append(
                {"product_id": int(r["product_id"]), "required_qty": float(r.get("quantity", 1) or 1)}
            )
    return list(dishes.values())


def _recommended_dishes_for_user(user_id: int, dishes: list[Dish], orders: list[Order]) -> list[RecommendedDish]:
    history = build_purchase_history(user_id, orders)
    if not history:
        return [{"dish_id": int(d["dish_id"]), "score": 1.0} for d in dishes[:10]]
    out: list[RecommendedDish] = []
    for d in dishes:
        ingredients = d.get("ingredients", [])
        if not ingredients:
            continue
        score = 0.0
        for ing in ingredients:
            pid = int(ing.get("product_id", 0))
            qty = float(ing.get("required_qty", 1) or 1)
            if pid in history:
                score += float(history[pid]["total_qty"]) * qty
        if score > 0:
            out.append({"dish_id": int(d["dish_id"]), "score": score})
    out.sort(key=lambda x: float(x["score"]), reverse=True)
    return out[:20]


def _load_user_profile(engine, user_id: int) -> UserProfile | None:
    cols = {c["name"] for c in inspect(engine).get_columns("users")}
    select_cols = []
    if "target_daily_kcal" in cols:
        select_cols.append("target_daily_kcal AS target_kcal")
    elif "target_calories_kcal" in cols:
        select_cols.append("target_calories_kcal AS target_kcal")
    if "goal" in cols:
        select_cols.append("goal")
    if not select_cols:
        return None
    q = text(f"SELECT {', '.join(select_cols)} FROM users WHERE id = :user_id")
    df = pd.read_sql(q, engine, params={"user_id": user_id})
    if df.empty:
        return None
    row = df.iloc[0].to_dict()
    return {
        "target_kcal": row.get("target_kcal"),
        "target_protein": row.get("target_protein"),
        "target_fat": row.get("target_fat"),
        "target_carbs": row.get("target_carbs"),
        "goal": row.get("goal"),
    }


def main() -> None:
    parser = argparse.ArgumentParser(description="Hybrid product recommender")
    parser.add_argument("--db-url", required=True)
    parser.add_argument("--user-id", type=int, required=True)
    parser.add_argument("--k", type=int, default=10)
    args = parser.parse_args()

    engine = create_engine(args.db_url, future=True)
    products = _load_products(engine)
    orders = _load_orders(engine)
    dishes = _load_dishes(engine)
    recommended_dishes = _recommended_dishes_for_user(args.user_id, dishes, orders)
    user_profile = _load_user_profile(engine, args.user_id)

    recs = top_n_recommendations(
        user_id=args.user_id,
        products=products,
        orders=orders,
        dishes=dishes,
        recommended_dishes=recommended_dishes,
        user_profile=user_profile,
        n=args.k,
    )
    print(json.dumps({"recommendations": recs}, ensure_ascii=False))


if __name__ == "__main__":
    main()
