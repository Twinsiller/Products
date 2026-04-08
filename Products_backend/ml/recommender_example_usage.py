import os

from recommender import RecommenderConfig, get_top_recommendations, precision_at_k, train_recommender


def main() -> None:
    db_url = os.getenv("DATABASE_URL")
    if not db_url:
        raise RuntimeError("Set DATABASE_URL env first")

    user_id = int(os.getenv("REC_USER_ID", "1"))
    top_k = int(os.getenv("REC_TOP_K", "5"))
    cfg = RecommenderConfig(svd_components=24, top_k_default=top_k)

    artifacts = train_recommender(db_url, config=cfg)
    recs = get_top_recommendations(artifacts, user_id=user_id, k=top_k)

    print(f"precision@5 = {precision_at_k(artifacts, k=5):.4f}")
    for i, rec in enumerate(recs, start=1):
        print(f"{i}. {rec['recipe_name']} (id={rec['recipe_id']}, score={rec['score']:.4f})")


if __name__ == "__main__":
    main()
