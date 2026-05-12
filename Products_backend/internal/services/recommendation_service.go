package services

import (
	"Products_backend/internal/models"

	"gorm.io/gorm"
)

// RecommendationService строит простые персональные рекомендации
// на основе избранных товаров и блюд пользователя.
type RecommendationService struct {
	DB *gorm.DB
}

// ProductRecommendation представляет рекомендованный товар с простым скором.
type ProductRecommendation struct {
	Product      models.Product                 `json:"product"`
	Score        float64                        `json:"score"`
	CBScore      float64                        `json:"cb_score,omitempty"`
	CFScore      float64                        `json:"cf_score,omitempty"`
	MealScore    float64                        `json:"meal_score,omitempty"`
	RecencyScore float64                        `json:"recency_score,omitempty"`
	Reason       string                         `json:"reason,omitempty"`
	LinkedDishes []ProductRecommendationDishRef `json:"linked_dishes,omitempty"`
}

type ProductRecommendationDishRef struct {
	DishID                     int64   `json:"dish_id"`
	DishName                   string  `json:"dish_name"`
	DishScore                  float64 `json:"dish_score"`
	MissingIngredientsEstimate int     `json:"missing_ingredients_estimate"`
}

// DishRecommendation представляет рекомендованное блюдо с простым скором.
type DishRecommendation struct {
	Dish  models.Dish `json:"dish"`
	Score int         `json:"score"`
}

// RecommendProducts возвращает топ-N товаров, похожих на избранные пользователя.
func (s *RecommendationService) RecommendProducts(userID int64, limit int) ([]ProductRecommendation, error) {
	if limit <= 0 {
		limit = 10
	}

	mlRecs, err := s.RunProductRecommender(userID, limit)
	if err == nil && len(mlRecs) > 0 {
		return mlRecs, nil
	}

	// 1. Собираем избранные товары пользователя.
	var favProducts []models.FavouriteProduct
	if err := s.DB.Where("user_id = ?", userID).Find(&favProducts).Error; err != nil {
		return nil, err
	}
	if len(favProducts) == 0 {
		return []ProductRecommendation{}, nil
	}

	// 2. Профиль по категориям и производителям.
	categoryScore := make(map[int64]int)
	manufacturerScore := make(map[int64]int)

	var favProductIDs []int64
	for _, fp := range favProducts {
		favProductIDs = append(favProductIDs, fp.ProductID)
	}

	var products []models.Product
	if err := s.DB.Where("id IN ?", favProductIDs).Find(&products).Error; err != nil {
		return nil, err
	}

	for _, p := range products {
		if p.CategoryID != nil {
			categoryScore[*p.CategoryID] += 2
		}
		if p.ManufacturerID != nil {
			manufacturerScore[*p.ManufacturerID] += 1
		}
	}

	// 3. Кандидаты: все товары, которых ещё нет в избранном.
	var candidates []models.Product
	if err := s.DB.
		Where("id NOT IN ?", favProductIDs).
		Find(&candidates).Error; err != nil {
		return nil, err
	}

	// 4. Подсчёт скора.
	var result []ProductRecommendation
	for _, c := range candidates {
		score := 0
		if c.CategoryID != nil {
			score += categoryScore[*c.CategoryID]
		}
		if c.ManufacturerID != nil {
			score += manufacturerScore[*c.ManufacturerID]
		}

		if score > 0 {
			result = append(result, ProductRecommendation{
				Product: c,
				Score:   float64(score),
				Reason:  "Рекомендация по совпадению с вашими избранными товарами (категория и производитель).",
			})
		}
	}

	// 5. Сортируем по убыванию score.
	for i := 0; i < len(result); i++ {
		for j := i + 1; j < len(result); j++ {
			if result[j].Score > result[i].Score {
				result[i], result[j] = result[j], result[i]
			}
		}
	}

	if len(result) > limit {
		result = result[:limit]
	}

	return result, nil
}

// RecommendDishes возвращает топ-N блюд, используя пересечение с любимыми товарами
// и блюдами пользователя.
func (s *RecommendationService) RecommendDishes(userID int64, limit int) ([]DishRecommendation, error) {
	if limit <= 0 {
		limit = 10
	}

	// 1. Избранные блюда.
	var favDishes []models.FavouriteDish
	if err := s.DB.Where("user_id = ?", userID).Find(&favDishes).Error; err != nil {
		return nil, err
	}
	var favDishIDs []int64
	for _, fd := range favDishes {
		favDishIDs = append(favDishIDs, fd.DishID)
	}

	// 2. Избранные товары.
	var favProducts []models.FavouriteProduct
	if err := s.DB.Where("user_id = ?", userID).Find(&favProducts).Error; err != nil {
		return nil, err
	}
	var favProductIDs []int64
	for _, fp := range favProducts {
		favProductIDs = append(favProductIDs, fp.ProductID)
	}

	// Если вообще нет сигналов — нечего рекомендовать.
	if len(favDishIDs) == 0 && len(favProductIDs) == 0 {
		return []DishRecommendation{}, nil
	}

	// 3. Кандидаты: все блюда, которых нет в избранном.
	var dishes []models.Dish
	if len(favDishIDs) > 0 {
		if err := s.DB.Where("id NOT IN ?", favDishIDs).Find(&dishes).Error; err != nil {
			return nil, err
		}
	} else {
		if err := s.DB.Find(&dishes).Error; err != nil {
			return nil, err
		}
	}

	// 4. Подтягиваем связку блюдо–товар.
	var dishProducts []models.DishProduct
	if err := s.DB.Find(&dishProducts).Error; err != nil {
		return nil, err
	}
	// Индекс: dishID -> список productID.
	dishToProducts := make(map[int64][]int64)
	for _, dp := range dishProducts {
		dishToProducts[dp.DishID] = append(dishToProducts[dp.DishID], dp.ProductID)
	}

	favProductsSet := make(map[int64]struct{})
	for _, id := range favProductIDs {
		favProductsSet[id] = struct{}{}
	}

	// 5. Считаем score для блюд.
	var result []DishRecommendation
	for _, d := range dishes {
		score := 0

		// Бонус, если блюдо уже в избранных (на случай, если мы всё-таки включим их).
		for _, id := range favDishIDs {
			if id == d.ID {
				score += 2
				break
			}
		}

		// Бонус за товары из избранного в составе блюда.
		for _, pid := range dishToProducts[d.ID] {
			if _, ok := favProductsSet[pid]; ok {
				score += 1
			}
		}

		if score > 0 {
			result = append(result, DishRecommendation{
				Dish:  d,
				Score: score,
			})
		}
	}

	// 6. Сортировка по score.
	for i := 0; i < len(result); i++ {
		for j := i + 1; j < len(result); j++ {
			if result[j].Score > result[i].Score {
				result[i], result[j] = result[j], result[i]
			}
		}
	}

	if len(result) > limit {
		result = result[:limit]
	}

	return result, nil
}
