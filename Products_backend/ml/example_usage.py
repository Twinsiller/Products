import os

from dish_recommender import RecommenderConfig, recommend_dishes_for_user


def main():
    db_url = os.getenv("DATABASE_URL")
    if not db_url:
        raise RuntimeError("Set DATABASE_URL env first")

    user_id = int(os.getenv("REC_USER_ID", "1"))
    top_k = int(os.getenv("REC_TOP_K", "10"))

    cfg = RecommenderConfig(
        svd_components=24,
        cf_weight=0.65,
        content_weight=0.35,
        favorite_boost=2.0,
        order_boost=1.0,
        top_k_default=top_k,
    )
    recs = recommend_dishes_for_user(db_url=db_url, user_id=user_id, top_k=top_k, config=cfg)
    print(recs[["dish_id", "dish_name", "final_score", "cf_score", "content_score"]].to_string(index=False))


if __name__ == "__main__":
    main()
