from __future__ import annotations

from dataclasses import dataclass
from pathlib import Path
from typing import Dict, Optional

import numpy as np
import pandas as pd
from sklearn.decomposition import TruncatedSVD
from sklearn.preprocessing import MinMaxScaler
from sqlalchemy import create_engine, text


@dataclass
class RecommenderConfig:
    svd_components: int = 24
    cf_weight: float = 0.65
    content_weight: float = 0.35
    favorite_boost: float = 2.0
    order_boost: float = 1.0
    top_k_default: int = 10


def log_user_event_to_csv(
    file_path: str,
    user_id: int,
    event_type: str,
    dish_id: Optional[int] = None,
    product_id: Optional[int] = None,
    value: float = 1.0,
    timestamp: Optional[pd.Timestamp] = None,
) -> None:
    event_time = pd.Timestamp.utcnow() if timestamp is None else pd.Timestamp(timestamp)
    row = pd.DataFrame(
        [
            {
                "event_time": event_time,
                "user_id": user_id,
                "event_type": event_type,
                "dish_id": dish_id,
                "product_id": product_id,
                "value": value,
            }
        ]
    )
    target = Path(file_path)
    target.parent.mkdir(parents=True, exist_ok=True)
    exists = target.exists()
    row.to_csv(target, mode="a", index=False, header=not exists)


def build_interactions_from_event_log(
    events_df: pd.DataFrame,
    dish_products_df: pd.DataFrame,
    config: RecommenderConfig,
) -> pd.DataFrame:
    if events_df.empty:
        return pd.DataFrame(columns=["user_id", "dish_id", "score"])

    dish_events = events_df.dropna(subset=["dish_id"]).copy()
    dish_events["score"] = dish_events["value"].astype(float)
    dish_events = dish_events[["user_id", "dish_id", "score"]]

    product_events = events_df.dropna(subset=["product_id"]).copy()
    if not product_events.empty:
        product_events = product_events.merge(
            dish_products_df[["dish_id", "product_id"]], on="product_id", how="inner"
        )
        product_events = (
            product_events.groupby(["user_id", "dish_id"], as_index=False)["value"].sum()
        )
        product_events["score"] = product_events["value"] * (config.favorite_boost * 0.5)
        product_events = product_events[["user_id", "dish_id", "score"]]
    else:
        product_events = pd.DataFrame(columns=["user_id", "dish_id", "score"])

    interactions = pd.concat([dish_events, product_events], ignore_index=True)
    return interactions.groupby(["user_id", "dish_id"], as_index=False)["score"].sum()


def get_engine(db_url: str):
    return create_engine(db_url, future=True)


def fetch_users(engine) -> pd.DataFrame:
    q = text(
        """
        SELECT id AS user_id, name, role, COALESCE(gender, '') AS gender
        FROM users
        """
    )
    return pd.read_sql(q, engine)


def fetch_orders(engine) -> pd.DataFrame:
    q = text(
        """
        SELECT id AS order_id, cashier_id AS user_id, created_at, total_amount
        FROM orders
        """
    )
    return pd.read_sql(q, engine)


def fetch_order_items(engine) -> pd.DataFrame:
    q = text(
        """
        SELECT id AS order_item_id, order_id, product_id, quantity, price_per_unit
        FROM order_items
        """
    )
    return pd.read_sql(q, engine)


def fetch_products(engine) -> pd.DataFrame:
    q = text(
        """
        SELECT
            id AS product_id,
            name AS product_name,
            category_id,
            manufacturer_id,
            COALESCE(calories_kcal, 0) AS calories_kcal,
            COALESCE(protein_g, 0) AS protein_g,
            COALESCE(fat_g, 0) AS fat_g,
            COALESCE(carbs_g, 0) AS carbs_g
        FROM products
        """
    )
    return pd.read_sql(q, engine)


def fetch_dishes(engine) -> pd.DataFrame:
    q = text("SELECT id AS dish_id, name AS dish_name FROM dishes")
    return pd.read_sql(q, engine)


def fetch_dish_products(engine) -> pd.DataFrame:
    q = text(
        """
        SELECT id, dish_id, product_id, quantity
        FROM dish_products
        """
    )
    return pd.read_sql(q, engine)


def fetch_favourite_dishes(engine) -> pd.DataFrame:
    q = text("SELECT user_id, dish_id FROM favourite_dishes")
    return pd.read_sql(q, engine)


def fetch_favourite_products(engine) -> pd.DataFrame:
    q = text("SELECT user_id, product_id FROM favourite_products")
    return pd.read_sql(q, engine)


def build_dish_nutrition(
    dishes_df: pd.DataFrame,
    dish_products_df: pd.DataFrame,
    products_df: pd.DataFrame,
) -> pd.DataFrame:
    merged = dish_products_df.merge(products_df, on="product_id", how="left")
    for col in ["calories_kcal", "protein_g", "fat_g", "carbs_g"]:
        merged[col] = merged[col].fillna(0) * merged["quantity"].fillna(0)

    agg = (
        merged.groupby("dish_id", as_index=False)[
            ["calories_kcal", "protein_g", "fat_g", "carbs_g"]
        ]
        .sum()
        .rename(
            columns={
                "calories_kcal": "dish_calories_kcal",
                "protein_g": "dish_protein_g",
                "fat_g": "dish_fat_g",
                "carbs_g": "dish_carbs_g",
            }
        )
    )
    return dishes_df.merge(agg, on="dish_id", how="left").fillna(0)


def build_user_dish_interactions(
    users_df: pd.DataFrame,
    orders_df: pd.DataFrame,
    order_items_df: pd.DataFrame,
    dishes_df: pd.DataFrame,
    dish_products_df: pd.DataFrame,
    favourite_dishes_df: pd.DataFrame,
    favourite_products_df: pd.DataFrame,
    config: RecommenderConfig,
) -> pd.DataFrame:
    # Order-driven implicit signal: if user ordered products that are part of dish.
    user_products = (
        orders_df[["order_id", "user_id"]]
        .merge(order_items_df[["order_id", "product_id", "quantity"]], on="order_id", how="inner")
        .groupby(["user_id", "product_id"], as_index=False)["quantity"]
        .sum()
    )

    dish_recipe = dish_products_df[["dish_id", "product_id", "quantity"]].rename(
        columns={"quantity": "recipe_qty"}
    )
    cross = user_products.merge(dish_recipe, on="product_id", how="inner")
    # Boost by overlap quantity, normalized at dish level.
    cross["order_signal"] = np.minimum(cross["quantity"], cross["recipe_qty"])
    order_signal = cross.groupby(["user_id", "dish_id"], as_index=False)["order_signal"].sum()
    order_signal["score"] = order_signal["order_signal"] * config.order_boost
    order_signal = order_signal[["user_id", "dish_id", "score"]]

    # Favorite dish signal.
    fav_dish_signal = favourite_dishes_df.copy()
    if not fav_dish_signal.empty:
        fav_dish_signal["score"] = config.favorite_boost
        fav_dish_signal = fav_dish_signal[["user_id", "dish_id", "score"]]
    else:
        fav_dish_signal = pd.DataFrame(columns=["user_id", "dish_id", "score"])

    # Favorite product -> dish signal.
    fav_prod_signal = favourite_products_df.merge(
        dish_products_df[["dish_id", "product_id"]], on="product_id", how="inner"
    )
    if not fav_prod_signal.empty:
        fav_prod_signal = (
            fav_prod_signal.groupby(["user_id", "dish_id"], as_index=False)
            .size()
            .rename(columns={"size": "score"})
        )
        fav_prod_signal["score"] = fav_prod_signal["score"] * (config.favorite_boost * 0.5)
        fav_prod_signal = fav_prod_signal[["user_id", "dish_id", "score"]]
    else:
        fav_prod_signal = pd.DataFrame(columns=["user_id", "dish_id", "score"])

    interactions = pd.concat([order_signal, fav_dish_signal, fav_prod_signal], ignore_index=True)
    if interactions.empty:
        # Build empty skeleton for cold-start.
        return (
            users_df[["user_id"]]
            .assign(key=1)
            .merge(dishes_df[["dish_id"]].assign(key=1), on="key")
            .drop(columns=["key"])
            .assign(score=0.0)
        )

    interactions = interactions.groupby(["user_id", "dish_id"], as_index=False)["score"].sum()
    return interactions


class CollaborativeFilteringModel:
    def __init__(self, n_components: int = 24):
        self.n_components = n_components
        self.svd: Optional[TruncatedSVD] = None
        self.user_index: Dict[int, int] = {}
        self.dish_index: Dict[int, int] = {}
        self.index_user: Dict[int, int] = {}
        self.index_dish: Dict[int, int] = {}
        self.user_factors: Optional[np.ndarray] = None
        self.dish_factors: Optional[np.ndarray] = None

    def fit(self, interactions_df: pd.DataFrame) -> "CollaborativeFilteringModel":
        users = sorted(interactions_df["user_id"].unique().tolist())
        dishes = sorted(interactions_df["dish_id"].unique().tolist())

        self.user_index = {uid: i for i, uid in enumerate(users)}
        self.dish_index = {did: j for j, did in enumerate(dishes)}
        self.index_user = {i: uid for uid, i in self.user_index.items()}
        self.index_dish = {j: did for did, j in self.dish_index.items()}

        mat = np.zeros((len(users), len(dishes)), dtype=np.float32)
        for row in interactions_df.itertuples(index=False):
            ui = self.user_index[row.user_id]
            di = self.dish_index[row.dish_id]
            mat[ui, di] = float(row.score)

        n_components = max(2, min(self.n_components, min(mat.shape) - 1))
        self.svd = TruncatedSVD(n_components=n_components, random_state=42)
        self.user_factors = self.svd.fit_transform(mat)
        self.dish_factors = self.svd.components_.T
        return self

    def predict_user_scores(self, user_id: int) -> Dict[int, float]:
        if (
            self.svd is None
            or self.user_factors is None
            or self.dish_factors is None
            or user_id not in self.user_index
        ):
            return {}
        uidx = self.user_index[user_id]
        scores = self.user_factors[uidx].dot(self.dish_factors.T)
        return {self.index_dish[j]: float(scores[j]) for j in range(scores.shape[0])}


class HybridDishRecommender:
    def __init__(self, config: Optional[RecommenderConfig] = None):
        self.config = config or RecommenderConfig()
        self.cf_model = CollaborativeFilteringModel(self.config.svd_components)
        self.users_df: Optional[pd.DataFrame] = None
        self.dish_features_df: Optional[pd.DataFrame] = None
        self.interactions_df: Optional[pd.DataFrame] = None
        self.scaler = MinMaxScaler()

    def fit(
        self,
        users_df: pd.DataFrame,
        dish_features_df: pd.DataFrame,
        interactions_df: pd.DataFrame,
    ) -> "HybridDishRecommender":
        self.users_df = users_df.copy()
        self.dish_features_df = dish_features_df.copy()
        self.interactions_df = interactions_df.copy()
        self.cf_model.fit(interactions_df)
        return self

    def _get_user_gender(self, user_id: int) -> str:
        if self.users_df is None:
            return ""
        row = self.users_df[self.users_df["user_id"] == user_id]
        if row.empty:
            return ""
        return str(row.iloc[0].get("gender", "") or "")

    def _target_calories_for_meal(self, user_id: int) -> float:
        gender = self._get_user_gender(user_id)
        daily = 2000.0 if gender == "female" else 2500.0
        return daily / 3.0

    def _content_score(self, user_id: int) -> Dict[int, float]:
        if self.dish_features_df is None:
            return {}
        df = self.dish_features_df.copy()
        target_kcal = self._target_calories_for_meal(user_id)

        # Soft target for macronutrients from meal calories.
        target_protein = (target_kcal * 0.25) / 4.0
        target_fat = (target_kcal * 0.30) / 9.0
        target_carbs = (target_kcal * 0.45) / 4.0

        df["kcal_fit"] = 1 - np.minimum(
            1.0, np.abs(df["dish_calories_kcal"] - target_kcal) / np.maximum(target_kcal, 1.0)
        )
        df["protein_fit"] = 1 - np.minimum(
            1.0, np.abs(df["dish_protein_g"] - target_protein) / np.maximum(target_protein, 1.0)
        )
        df["fat_fit"] = 1 - np.minimum(
            1.0, np.abs(df["dish_fat_g"] - target_fat) / np.maximum(target_fat, 1.0)
        )
        df["carbs_fit"] = 1 - np.minimum(
            1.0, np.abs(df["dish_carbs_g"] - target_carbs) / np.maximum(target_carbs, 1.0)
        )

        df["content_score"] = (
            0.45 * df["kcal_fit"]
            + 0.20 * df["protein_fit"]
            + 0.15 * df["fat_fit"]
            + 0.20 * df["carbs_fit"]
        )
        return dict(zip(df["dish_id"], df["content_score"].astype(float)))

    def recommend(self, user_id: int, top_k: Optional[int] = None) -> pd.DataFrame:
        if self.dish_features_df is None:
            raise RuntimeError("Model is not fitted")

        top_k = top_k or self.config.top_k_default

        cf_scores = self.cf_model.predict_user_scores(user_id)
        content_scores = self._content_score(user_id)

        all_dishes = self.dish_features_df[["dish_id", "dish_name"]].copy()
        all_dishes["cf_score"] = all_dishes["dish_id"].map(cf_scores).fillna(0.0)
        all_dishes["content_score"] = all_dishes["dish_id"].map(content_scores).fillna(0.0)

        score_cols = all_dishes[["cf_score", "content_score"]].to_numpy()
        norm = self.scaler.fit_transform(score_cols)
        all_dishes["cf_norm"] = norm[:, 0]
        all_dishes["content_norm"] = norm[:, 1]
        all_dishes["final_score"] = (
            self.config.cf_weight * all_dishes["cf_norm"]
            + self.config.content_weight * all_dishes["content_norm"]
        )

        # Already strongly interacted dishes can be filtered if desired.
        if self.interactions_df is not None:
            seen = set(
                self.interactions_df[self.interactions_df["user_id"] == user_id]["dish_id"].tolist()
            )
            result = all_dishes[~all_dishes["dish_id"].isin(seen)].copy()
        else:
            result = all_dishes

        result = result.sort_values("final_score", ascending=False).head(top_k)
        return result.reset_index(drop=True)


def prepare_training_data(db_url: str, config: Optional[RecommenderConfig] = None):
    cfg = config or RecommenderConfig()
    engine = get_engine(db_url)

    users_df = fetch_users(engine)
    orders_df = fetch_orders(engine)
    order_items_df = fetch_order_items(engine)
    products_df = fetch_products(engine)
    dishes_df = fetch_dishes(engine)
    dish_products_df = fetch_dish_products(engine)
    favourite_dishes_df = fetch_favourite_dishes(engine)
    favourite_products_df = fetch_favourite_products(engine)

    dish_features_df = build_dish_nutrition(dishes_df, dish_products_df, products_df)
    interactions_df = build_user_dish_interactions(
        users_df=users_df,
        orders_df=orders_df,
        order_items_df=order_items_df,
        dishes_df=dishes_df,
        dish_products_df=dish_products_df,
        favourite_dishes_df=favourite_dishes_df,
        favourite_products_df=favourite_products_df,
        config=cfg,
    )
    return users_df, dish_features_df, interactions_df


def train_hybrid_recommender(
    db_url: str, config: Optional[RecommenderConfig] = None
) -> HybridDishRecommender:
    cfg = config or RecommenderConfig()
    users_df, dish_features_df, interactions_df = prepare_training_data(db_url, cfg)
    model = HybridDishRecommender(cfg)
    model.fit(users_df, dish_features_df, interactions_df)
    return model


def recommend_dishes_for_user(
    db_url: str,
    user_id: int,
    top_k: int = 10,
    config: Optional[RecommenderConfig] = None,
) -> pd.DataFrame:
    model = train_hybrid_recommender(db_url, config)
    return model.recommend(user_id=user_id, top_k=top_k)

