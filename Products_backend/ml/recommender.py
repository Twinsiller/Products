from __future__ import annotations

import argparse
import json
from dataclasses import dataclass
from typing import Dict, List, Optional, Tuple

import numpy as np
import pandas as pd
from scipy.sparse import csr_matrix
from sklearn.decomposition import TruncatedSVD
from sqlalchemy import create_engine, inspect, text
from sqlalchemy.engine import Engine


@dataclass
class RecommenderConfig:
    random_state: int = 42
    svd_components: int = 24
    top_k_default: int = 5


@dataclass
class RecommenderArtifacts:
    config: RecommenderConfig
    train_df: pd.DataFrame
    test_df: pd.DataFrame
    interactions_train: csr_matrix
    user_to_index: Dict[int, int]
    index_to_user: Dict[int, int]
    recipe_to_index: Dict[int, int]
    index_to_recipe: Dict[int, int]
    model: TruncatedSVD
    user_factors: np.ndarray
    recipe_factors: np.ndarray
    recipe_features: pd.DataFrame
    user_profiles: pd.DataFrame
    recipe_table: str
    orders_table: str = "orders"


def get_engine(db_url: str) -> Engine:
    return create_engine(db_url, future=True)


def ensure_orders_table(engine: Engine) -> None:
    """Create orders(user_id, recipe_id) only if missing."""
    insp = inspect(engine)
    if insp.has_table("orders"):
        return
    has_recipes = insp.has_table("recipes")
    recipe_ref = "recipes" if has_recipes else "dishes"
    ddl = f"""
    CREATE TABLE orders (
        id BIGSERIAL PRIMARY KEY,
        user_id BIGINT NOT NULL,
        recipe_id BIGINT NOT NULL,
        created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
        CONSTRAINT fk_orders_user FOREIGN KEY (user_id) REFERENCES users(id),
        CONSTRAINT fk_orders_recipe FOREIGN KEY (recipe_id) REFERENCES {recipe_ref}(id)
    );
    """
    with engine.begin() as conn:
        conn.execute(text(ddl))


def _detect_recipe_tables(engine: Engine) -> Tuple[str, str]:
    insp = inspect(engine)
    # Preferred naming from your request.
    if insp.has_table("recipes") and insp.has_table("recipe_ingredients"):
        return "recipes", "recipe_ingredients"
    # Backward compatibility with current project naming.
    if insp.has_table("dishes") and insp.has_table("dish_products"):
        return "dishes", "dish_products"
    raise RuntimeError("No recipe tables found (expected recipes/recipe_ingredients or dishes/dish_products)")


def _load_orders(engine: Engine) -> pd.DataFrame:
    insp = inspect(engine)
    if not insp.has_table("orders"):
        raise RuntimeError("orders table is missing")
    cols = {c["name"] for c in insp.get_columns("orders")}

    if {"user_id", "recipe_id"}.issubset(cols):
        q = text("SELECT user_id, recipe_id, created_at FROM orders")
        return pd.read_sql(q, engine)

    # Existing project schema fallback: no direct recipe_id in orders.
    if {"cashier_id", "id"}.issubset(cols):
        q = text("SELECT id AS order_id, cashier_id AS user_id, created_at FROM orders")
        return pd.read_sql(q, engine)
    raise RuntimeError("orders schema is incompatible for collaborative filtering")


def _load_order_items_with_category(engine: Engine) -> pd.DataFrame:
    q = text(
        """
        SELECT
            oi.order_id,
            oi.product_id,
            oi.quantity,
            p.category_id
        FROM order_items oi
        JOIN products p ON p.id = oi.product_id
        """
    )
    return pd.read_sql(q, engine)


def _load_recipe_requirements_by_category(engine: Engine, recipe_table: str) -> pd.DataFrame:
    insp = inspect(engine)
    if recipe_table == "dishes" and insp.has_table("dish_category_requirements"):
        q = text(
            """
            SELECT
                dish_id AS recipe_id,
                category_id,
                quantity AS req_qty
            FROM dish_category_requirements
            """
        )
        req = pd.read_sql(q, engine)
        if not req.empty:
            return req

    if recipe_table == "recipes":
        q = text(
            """
            SELECT
                ri.recipe_id AS recipe_id,
                p.category_id AS category_id,
                SUM(ri.quantity) AS req_qty
            FROM recipe_ingredients ri
            JOIN products p ON p.id = ri.product_id
            GROUP BY ri.recipe_id, p.category_id
            """
        )
        return pd.read_sql(q, engine)

    q = text(
        """
        SELECT
            dp.dish_id AS recipe_id,
            p.category_id AS category_id,
            SUM(dp.quantity) AS req_qty
        FROM dish_products dp
        JOIN products p ON p.id = dp.product_id
        GROUP BY dp.dish_id, p.category_id
        """
    )
    return pd.read_sql(q, engine)


def _build_interactions_from_order_baskets(
    orders_df: pd.DataFrame,
    order_items_df: pd.DataFrame,
    recipe_requirements_df: pd.DataFrame,
) -> pd.DataFrame:
    if orders_df.empty or order_items_df.empty or recipe_requirements_df.empty:
        return pd.DataFrame(columns=["user_id", "recipe_id", "strength"])

    order_cat = (
        order_items_df.dropna(subset=["category_id"])
        .groupby(["order_id", "category_id"], as_index=False)["quantity"]
        .sum()
        .rename(columns={"quantity": "have_qty"})
    )

    req = recipe_requirements_df.dropna(subset=["category_id"]).copy()
    req["req_qty"] = req["req_qty"].astype(float)
    req_totals = req.groupby("recipe_id", as_index=False)["req_qty"].sum().rename(columns={"req_qty": "total_req"})

    merged = order_cat.merge(req, on="category_id", how="inner")
    if merged.empty:
        return pd.DataFrame(columns=["user_id", "recipe_id", "strength"])

    merged["covered"] = np.minimum(merged["have_qty"].astype(float), merged["req_qty"])
    coverage = merged.groupby(["order_id", "recipe_id"], as_index=False)["covered"].sum()
    coverage = coverage.merge(req_totals, on="recipe_id", how="left")
    coverage["ratio"] = coverage["covered"] / coverage["total_req"].replace(0, np.nan)
    coverage["ratio"] = coverage["ratio"].fillna(0.0).clip(0.0, 1.0)

    # Make fully-coverable dishes stronger positives.
    coverage["strength"] = coverage["ratio"] + (coverage["ratio"] >= 0.999).astype(float)
    coverage = coverage[coverage["strength"] > 0]

    orders_id_map = orders_df[["order_id", "user_id"]].drop_duplicates()
    out = coverage.merge(orders_id_map, on="order_id", how="inner")
    return out.groupby(["user_id", "recipe_id"], as_index=False)["strength"].sum()


def _load_recipe_features(engine: Engine, recipe_table: str, ingredients_table: str) -> pd.DataFrame:
    # recipes/recipe_ingredients convention
    if recipe_table == "recipes":
        q = text(
            """
            SELECT
                r.id AS recipe_id,
                r.name AS recipe_name,
                COALESCE(SUM(COALESCE(p.calories_kcal, 0) * COALESCE(ri.quantity, 0)), 0) AS kcal,
                COALESCE(SUM(COALESCE(p.protein_g, 0) * COALESCE(ri.quantity, 0)), 0) AS protein_g,
                COALESCE(SUM(COALESCE(p.fat_g, 0) * COALESCE(ri.quantity, 0)), 0) AS fat_g,
                COALESCE(SUM(COALESCE(p.carbs_g, 0) * COALESCE(ri.quantity, 0)), 0) AS carbs_g
            FROM recipes r
            LEFT JOIN recipe_ingredients ri ON ri.recipe_id = r.id
            LEFT JOIN products p ON p.id = ri.product_id
            GROUP BY r.id, r.name
            """
        )
        return pd.read_sql(q, engine)

    # dishes/dish_products convention
    q = text(
        """
        SELECT
            d.id AS recipe_id,
            d.name AS recipe_name,
            COALESCE(SUM(COALESCE(p.calories_kcal, 0) * COALESCE(dp.quantity, 0)), 0) AS kcal,
            COALESCE(SUM(COALESCE(p.protein_g, 0) * COALESCE(dp.quantity, 0)), 0) AS protein_g,
            COALESCE(SUM(COALESCE(p.fat_g, 0) * COALESCE(dp.quantity, 0)), 0) AS fat_g,
            COALESCE(SUM(COALESCE(p.carbs_g, 0) * COALESCE(dp.quantity, 0)), 0) AS carbs_g
        FROM dishes d
        LEFT JOIN dish_products dp ON dp.dish_id = d.id
        LEFT JOIN products p ON p.id = dp.product_id
        GROUP BY d.id, d.name
        """
    )
    return pd.read_sql(q, engine)


def _build_interactions(orders_df: pd.DataFrame) -> pd.DataFrame:
    # Direct user->recipe orders schema.
    interactions = (
        orders_df.groupby(["user_id", "recipe_id"], as_index=False)
        .size()
        .rename(columns={"size": "strength"})
    )
    interactions["strength"] = interactions["strength"].astype(float)
    return interactions


def train_test_split_leave_one_out(
    interactions_df: pd.DataFrame, random_state: int = 42
) -> Tuple[pd.DataFrame, pd.DataFrame]:
    rng = np.random.default_rng(random_state)
    test_rows = []
    train_rows = []

    for user_id, grp in interactions_df.groupby("user_id"):
        grp = grp.sample(frac=1, random_state=int(rng.integers(0, 1_000_000)))
        if len(grp) > 1:
            test_rows.append(grp.iloc[[0]])
            train_rows.append(grp.iloc[1:])
        else:
            train_rows.append(grp)

    train_df = pd.concat(train_rows, ignore_index=True) if train_rows else interactions_df.iloc[0:0].copy()
    test_df = pd.concat(test_rows, ignore_index=True) if test_rows else interactions_df.iloc[0:0].copy()
    return train_df, test_df


def _matrix_from_interactions(
    interactions_df: pd.DataFrame,
) -> Tuple[csr_matrix, Dict[int, int], Dict[int, int], Dict[int, int], Dict[int, int]]:
    users = sorted(interactions_df["user_id"].unique().tolist())
    recipes = sorted(interactions_df["recipe_id"].unique().tolist())
    user_to_index = {u: i for i, u in enumerate(users)}
    recipe_to_index = {r: j for j, r in enumerate(recipes)}
    index_to_user = {i: u for u, i in user_to_index.items()}
    index_to_recipe = {j: r for r, j in recipe_to_index.items()}

    rows = interactions_df["user_id"].map(user_to_index).to_numpy()
    cols = interactions_df["recipe_id"].map(recipe_to_index).to_numpy()
    vals = interactions_df["strength"].astype(float).to_numpy()
    mat = csr_matrix((vals, (rows, cols)), shape=(len(users), len(recipes)))
    return mat, user_to_index, index_to_user, recipe_to_index, index_to_recipe


def train_recommender(db_url: str, config: Optional[RecommenderConfig] = None) -> RecommenderArtifacts:
    cfg = config or RecommenderConfig()
    engine = get_engine(db_url)
    ensure_orders_table(engine)
    recipe_table, ingredients_table = _detect_recipe_tables(engine)
    orders_df = _load_orders(engine)
    if {"recipe_id"}.issubset(set(orders_df.columns)):
        interactions = _build_interactions(orders_df)
    else:
        order_items_df = _load_order_items_with_category(engine)
        req_df = _load_recipe_requirements_by_category(engine, recipe_table)
        interactions = _build_interactions_from_order_baskets(orders_df, order_items_df, req_df)

    train_df, test_df = train_test_split_leave_one_out(interactions, random_state=cfg.random_state)
    if train_df.empty:
        raise RuntimeError("Not enough order history to train collaborative model")

    mat, u2i, i2u, r2i, i2r = _matrix_from_interactions(train_df)
    max_components = max(1, min(cfg.svd_components, min(mat.shape) - 1))
    model = TruncatedSVD(n_components=max_components, random_state=cfg.random_state)
    user_factors = model.fit_transform(mat)
    recipe_factors = model.components_.T
    recipe_features = _load_recipe_features(engine, recipe_table, ingredients_table)
    user_profiles = _load_user_profiles(engine)

    return RecommenderArtifacts(
        config=cfg,
        train_df=train_df,
        test_df=test_df,
        interactions_train=mat,
        user_to_index=u2i,
        index_to_user=i2u,
        recipe_to_index=r2i,
        index_to_recipe=i2r,
        model=model,
        user_factors=user_factors,
        recipe_factors=recipe_factors,
        recipe_features=recipe_features,
        user_profiles=user_profiles,
        recipe_table=recipe_table,
    )


def _load_user_profiles(engine: Engine) -> pd.DataFrame:
    insp = inspect(engine)
    cols = {c["name"] for c in insp.get_columns("users")}
    select_cols = ["id AS user_id"]
    if "weight_kg" in cols:
        select_cols.append("weight_kg")
    if "goal" in cols:
        select_cols.append("goal")
    if "target_daily_kcal" in cols:
        select_cols.append("target_daily_kcal")
    elif "target_calories_kcal" in cols:
        select_cols.append("target_calories_kcal AS target_daily_kcal")

    q = text(f"SELECT {', '.join(select_cols)} FROM users")
    return pd.read_sql(q, engine)


def _predict_scores(artifacts: RecommenderArtifacts, user_id: int) -> Dict[int, float]:
    if user_id not in artifacts.user_to_index:
        return {}
    uidx = artifacts.user_to_index[user_id]
    scores = artifacts.user_factors[uidx].dot(artifacts.recipe_factors.T)
    return {artifacts.index_to_recipe[i]: float(scores[i]) for i in range(len(scores))}


def _user_seen_recipes(train_df: pd.DataFrame, user_id: int) -> set[int]:
    return set(train_df.loc[train_df["user_id"] == user_id, "recipe_id"].tolist())


def _content_based_fallback(
    artifacts: RecommenderArtifacts,
    k: int,
    weight_kg: Optional[float] = None,
    goal: Optional[str] = None,
    target_daily_kcal: Optional[float] = None,
) -> List[Dict]:
    recipes = artifacts.recipe_features.copy()
    if recipes.empty:
        return []

    if target_daily_kcal is None:
        base = 2200.0
        if weight_kg is not None and weight_kg > 0:
            base = weight_kg * 30.0
        if goal == "lose":
            base -= 300.0
        elif goal == "gain":
            base += 300.0
        target_daily_kcal = max(1200.0, base)

    meal_target = target_daily_kcal / 3.0
    target_protein_g = (meal_target * 0.30) / 4.0
    target_fat_g = (meal_target * 0.30) / 9.0
    target_carbs_g = (meal_target * 0.40) / 4.0

    recipes["kcal_fit"] = 1 - np.minimum(1.0, np.abs(recipes["kcal"] - meal_target) / np.maximum(meal_target, 1.0))
    recipes["protein_fit"] = 1 - np.minimum(
        1.0, np.abs(recipes["protein_g"] - target_protein_g) / np.maximum(target_protein_g, 1.0)
    )
    recipes["fat_fit"] = 1 - np.minimum(1.0, np.abs(recipes["fat_g"] - target_fat_g) / np.maximum(target_fat_g, 1.0))
    recipes["carbs_fit"] = 1 - np.minimum(
        1.0, np.abs(recipes["carbs_g"] - target_carbs_g) / np.maximum(target_carbs_g, 1.0)
    )
    recipes["score"] = 0.5 * recipes["kcal_fit"] + 0.2 * recipes["protein_fit"] + 0.15 * recipes["fat_fit"] + 0.15 * recipes["carbs_fit"]
    top = recipes.sort_values("score", ascending=False).head(k)
    return top[["recipe_id", "recipe_name", "score", "kcal", "protein_g", "fat_g", "carbs_g"]].to_dict("records")


def _content_based_fallback_from_features(
    recipes: pd.DataFrame,
    k: int,
    weight_kg: Optional[float] = None,
    goal: Optional[str] = None,
    target_daily_kcal: Optional[float] = None,
) -> List[Dict]:
    if recipes.empty:
        return []

    rows = recipes.copy()
    if target_daily_kcal is None:
        base = 2200.0
        if weight_kg is not None and weight_kg > 0:
            base = weight_kg * 30.0
        if goal == "lose":
            base -= 300.0
        elif goal == "gain":
            base += 300.0
        target_daily_kcal = max(1200.0, base)

    meal_target = target_daily_kcal / 3.0
    target_protein_g = (meal_target * 0.30) / 4.0
    target_fat_g = (meal_target * 0.30) / 9.0
    target_carbs_g = (meal_target * 0.40) / 4.0

    rows["kcal_fit"] = 1 - np.minimum(1.0, np.abs(rows["kcal"] - meal_target) / np.maximum(meal_target, 1.0))
    rows["protein_fit"] = 1 - np.minimum(
        1.0, np.abs(rows["protein_g"] - target_protein_g) / np.maximum(target_protein_g, 1.0)
    )
    rows["fat_fit"] = 1 - np.minimum(1.0, np.abs(rows["fat_g"] - target_fat_g) / np.maximum(target_fat_g, 1.0))
    rows["carbs_fit"] = 1 - np.minimum(
        1.0, np.abs(rows["carbs_g"] - target_carbs_g) / np.maximum(target_carbs_g, 1.0)
    )
    rows["score"] = 0.5 * rows["kcal_fit"] + 0.2 * rows["protein_fit"] + 0.15 * rows["fat_fit"] + 0.15 * rows["carbs_fit"]
    top = rows.sort_values("score", ascending=False).head(k)
    return top[["recipe_id", "recipe_name", "score", "kcal", "protein_g", "fat_g", "carbs_g"]].to_dict("records")


def _read_user_fallback_profile(
    user_profiles: pd.DataFrame,
    user_id: int,
    weight_kg: Optional[float],
    goal: Optional[str],
    target_daily_kcal: Optional[float],
) -> Tuple[Optional[float], Optional[str], Optional[float]]:
    if user_profiles is None or user_profiles.empty:
        return weight_kg, goal, target_daily_kcal
    row = user_profiles.loc[user_profiles["user_id"] == user_id]
    if row.empty:
        return weight_kg, goal, target_daily_kcal

    if weight_kg is None and "weight_kg" in row.columns:
        v = row.iloc[0].get("weight_kg")
        weight_kg = float(v) if pd.notna(v) else weight_kg
    if goal is None and "goal" in row.columns:
        v = row.iloc[0].get("goal")
        goal = str(v) if pd.notna(v) else goal
    if target_daily_kcal is None and "target_daily_kcal" in row.columns:
        v = row.iloc[0].get("target_daily_kcal")
        target_daily_kcal = float(v) if pd.notna(v) else target_daily_kcal
    return weight_kg, goal, target_daily_kcal


def get_top_recommendations(
    artifacts: RecommenderArtifacts,
    user_id: int,
    k: int = 5,
    weight_kg: Optional[float] = None,
    goal: Optional[str] = None,
    target_daily_kcal: Optional[float] = None,
) -> List[Dict]:
    if target_daily_kcal is None and artifacts.user_profiles is not None and not artifacts.user_profiles.empty:
        row = artifacts.user_profiles.loc[artifacts.user_profiles["user_id"] == user_id]
        if not row.empty:
            if weight_kg is None and "weight_kg" in row.columns:
                v = row.iloc[0].get("weight_kg")
                weight_kg = float(v) if pd.notna(v) else weight_kg
            if goal is None and "goal" in row.columns:
                v = row.iloc[0].get("goal")
                goal = str(v) if pd.notna(v) else goal
            if "target_daily_kcal" in row.columns and target_daily_kcal is None:
                v = row.iloc[0].get("target_daily_kcal")
                target_daily_kcal = float(v) if pd.notna(v) else target_daily_kcal

    cf_scores = _predict_scores(artifacts, user_id)
    if not cf_scores:
        return _content_based_fallback(artifacts, k, weight_kg, goal, target_daily_kcal)

    seen = _user_seen_recipes(artifacts.train_df, user_id)
    ranked = sorted(cf_scores.items(), key=lambda x: x[1], reverse=True)
    ranked = [(rid, score) for rid, score in ranked if rid not in seen][:k]

    names = artifacts.recipe_features[["recipe_id", "recipe_name"]].drop_duplicates()
    name_map = dict(zip(names["recipe_id"], names["recipe_name"]))
    return [{"recipe_id": rid, "recipe_name": name_map.get(rid, f"recipe_{rid}"), "score": float(score)} for rid, score in ranked]


def precision_at_k(artifacts: RecommenderArtifacts, k: int = 5) -> float:
    if artifacts.test_df.empty:
        return 0.0
    hits = 0
    total = 0
    for row in artifacts.test_df.itertuples(index=False):
        user_id = int(row.user_id)
        truth_recipe = int(row.recipe_id)
        recs = get_top_recommendations(artifacts, user_id=user_id, k=k)
        rec_ids = [int(x["recipe_id"]) for x in recs]
        hits += 1 if truth_recipe in rec_ids else 0
        total += 1
    return float(hits / total) if total > 0 else 0.0


def fit_and_evaluate(db_url: str, k: int = 5, config: Optional[RecommenderConfig] = None) -> Dict[str, float]:
    artifacts = train_recommender(db_url, config=config)
    return {"precision_at_5": precision_at_k(artifacts, k=k)}


def _cli() -> None:
    parser = argparse.ArgumentParser(description="Train and run recipe recommender")
    parser.add_argument("--db-url", required=True, help="PostgreSQL SQLAlchemy URL")
    parser.add_argument("--user-id", type=int, required=True, help="User ID to recommend for")
    parser.add_argument("--k", type=int, default=5, help="Top-K recommendations")
    parser.add_argument("--weight-kg", type=float, default=None, help="Fallback: user weight")
    parser.add_argument("--goal", type=str, default=None, help="Fallback: lose|maintain|gain")
    parser.add_argument("--target-daily-kcal", type=float, default=None, help="Fallback: target kcal/day")
    parser.add_argument("--components", type=int, default=24, help="SVD components")
    args = parser.parse_args()

    cfg = RecommenderConfig(svd_components=args.components, top_k_default=args.k)
    try:
        artifacts = train_recommender(args.db_url, config=cfg)
        recs = get_top_recommendations(
            artifacts,
            user_id=args.user_id,
            k=args.k,
            weight_kg=args.weight_kg,
            goal=args.goal,
            target_daily_kcal=args.target_daily_kcal,
        )
        report = {
            "precision_at_5": precision_at_k(artifacts, k=5),
            "recommendations": recs,
        }
    except RuntimeError as err:
        # If there is not enough collaborative history yet, still return recommendations via content fallback.
        if "Not enough order history" not in str(err):
            raise
        engine = get_engine(args.db_url)
        recipe_table, ingredients_table = _detect_recipe_tables(engine)
        recipe_features = _load_recipe_features(engine, recipe_table, ingredients_table)
        user_profiles = _load_user_profiles(engine)
        w, g, t = _read_user_fallback_profile(
            user_profiles=user_profiles,
            user_id=args.user_id,
            weight_kg=args.weight_kg,
            goal=args.goal,
            target_daily_kcal=args.target_daily_kcal,
        )
        recs = _content_based_fallback_from_features(recipe_features, k=args.k, weight_kg=w, goal=g, target_daily_kcal=t)
        report = {
            "precision_at_5": 0.0,
            "recommendations": recs,
        }
    print(json.dumps(report, ensure_ascii=False, indent=2))


if __name__ == "__main__":
    _cli()
