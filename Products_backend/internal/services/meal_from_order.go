package services

import (
	"errors"
	"math"
	"regexp"
	"strings"

	"Products_backend/internal/models"
)

// ErrOrderAccess — заказ не принадлежит пользователю.
var ErrOrderAccess = errors.New("order access denied")

// Среднесуточная норма калорий (упрощённо, без возраста/активности).
const (
	dailyCaloriesMale   = 2500.0
	dailyCaloriesFemale = 2000.0
	mealsPerDay         = 3.0
)

type categoryNutritionAgg struct {
	Count int
	Kcal  float64
	Prot  float64
	Fat   float64
	Carbs float64
}

// MissingIngredient — не хватает категории продукта для блюда; подсказки по конкретным товарам этой категории.
type MissingIngredient struct {
	NeededIngredient string `json:"needed_ingredient"`

	// Поля ниже оставлены для совместимости с текущим frontend.
	NeededCategoryID   int64            `json:"needed_category_id"`
	NeededCategoryName string           `json:"needed_category_name"`
	QtyShort           int              `json:"qty_short"`
	Suggestions        []models.Product `json:"suggestions"`
}

// OrderMealRecommendation — блюдо относительно заказа пользователя.
type OrderMealRecommendation struct {
	Dish           models.Dish         `json:"dish"`
	Score          float64             `json:"score"`
	Makeable       bool                `json:"makeable"`
	MatchedItems   int                 `json:"matched_items"`
	RequiredItems  int                 `json:"required_items"`
	MatchRatio     float64             `json:"match_ratio"`
	TotalKcal      float64             `json:"total_calories_kcal"`
	TotalProteinG  float64             `json:"total_protein_g"`
	TotalFatG      float64             `json:"total_fat_g"`
	TotalCarbsG    float64             `json:"total_carbs_g"`
	Missing        []MissingIngredient `json:"missing,omitempty"`
	MealTargetKcal float64             `json:"meal_target_kcal"`
}

// DishesFromOrder подбирает блюда по товарам из заказа, ранжирует по КБЖУ и близости к целевым калориям на приём пищи.
func (s *RecommendationService) DishesFromOrder(userID, orderID int64, limit int) ([]OrderMealRecommendation, error) {
	if limit <= 0 {
		limit = 20
	}

	var user models.User
	if err := s.DB.First(&user, userID).Error; err != nil {
		return nil, err
	}

	var order models.Order
	if err := s.DB.First(&order, orderID).Error; err != nil {
		return nil, err
	}
	if order.UserID != userID {
		return nil, ErrOrderAccess
	}

	var items []models.OrderItem
	if err := s.DB.Where("order_id = ?", orderID).Find(&items).Error; err != nil {
		return nil, err
	}
	avail := make(map[int64]int)
	for _, it := range items {
		avail[it.ProductID] += it.Quantity
	}

	mealTarget := dailyCaloriesMale / mealsPerDay
	if user.Gender == "female" {
		mealTarget = dailyCaloriesFemale / mealsPerDay
	}

	var dishes []models.Dish
	if err := s.DB.
		Preload("Products").
		Preload("Products.Product").
		Preload("CategoryRequirements").
		Preload("CategoryRequirements.Category").
		Find(&dishes).Error; err != nil {
		return nil, err
	}

	var out []OrderMealRecommendation
	for _, d := range dishes {
		rec := s.scoreDishByCategories(d, avail, mealTarget)
		out = append(out, rec)
	}
	return limitAndSortMeals(out, limit), nil
}

// DishesFromCart подбирает блюда по списку товаров из корзины.
func (s *RecommendationService) DishesFromCart(userID int64, avail map[int64]int, limit int) ([]OrderMealRecommendation, error) {
	if limit <= 0 {
		limit = 20
	}

	var user models.User
	if err := s.DB.First(&user, userID).Error; err != nil {
		return nil, err
	}

	mealTarget := dailyCaloriesMale / mealsPerDay
	if user.Gender == "female" {
		mealTarget = dailyCaloriesFemale / mealsPerDay
	}

	var dishes []models.Dish
	if err := s.DB.
		Preload("Products").
		Preload("Products.Product").
		Preload("CategoryRequirements").
		Preload("CategoryRequirements.Category").
		Find(&dishes).Error; err != nil {
		return nil, err
	}

	out := make([]OrderMealRecommendation, 0, len(dishes))
	for _, d := range dishes {
		out = append(out, s.scoreDishByCategories(d, avail, mealTarget))
	}
	return limitAndSortMeals(out, limit), nil
}

func limitAndSortMeals(out []OrderMealRecommendation, limit int) []OrderMealRecommendation {
	// Сначала блюда с максимальным совпадением с корзиной, затем собираемость и score.
	for i := 0; i < len(out); i++ {
		for j := i + 1; j < len(out); j++ {
			if out[j].MatchRatio > out[i].MatchRatio {
				out[i], out[j] = out[j], out[i]
				continue
			}
			if out[j].MatchRatio < out[i].MatchRatio {
				continue
			}
			if out[j].MatchedItems > out[i].MatchedItems {
				out[i], out[j] = out[j], out[i]
				continue
			}
			if out[j].MatchedItems < out[i].MatchedItems {
				continue
			}
			if out[j].Makeable && !out[i].Makeable {
				out[i], out[j] = out[j], out[i]
				continue
			}
			if out[j].Makeable == out[i].Makeable && out[j].Score > out[i].Score {
				out[i], out[j] = out[j], out[i]
			}
		}
	}

	if len(out) > limit {
		return out[:limit]
	}
	return out
}

func (s *RecommendationService) scoreDishByCategories(d models.Dish, avail map[int64]int, mealTargetKcal float64) OrderMealRecommendation {
	if len(d.Products) > 0 {
		return s.scoreDishByProducts(d, avail, mealTargetKcal)
	}

	var miss []MissingIngredient
	makeable := true
	matched := 0
	required := len(d.CategoryRequirements)

	var kcal, prot, fat, carbs float64
	ingredientAvail := make(map[string]int)
	ingredientNutrition := make(map[string]categoryNutritionAgg)

	if len(avail) > 0 {
		ids := make([]int64, 0, len(avail))
		for id := range avail {
			ids = append(ids, id)
		}
		var products []models.Product
		_ = s.DB.Where("id IN ?", ids).Find(&products).Error
		for _, p := range products {
			q := avail[p.ID]
			key := normalizeIngredientKey(p.Name)
			if key == "" {
				continue
			}
			ingredientAvail[key] += q
			prev := ingredientNutrition[key]
			prev.Count += q
			prev.Kcal += p.CaloriesKcal * float64(q)
			prev.Prot += p.ProteinG * float64(q)
			prev.Fat += p.FatG * float64(q)
			prev.Carbs += p.CarbsG * float64(q)
			ingredientNutrition[key] = prev
		}
	}

	for _, req := range d.CategoryRequirements {
		reqName := strings.TrimSpace(req.IngredientName)
		if reqName == "" {
			if req.Category != nil && req.Category.Name != "" {
				reqName = req.Category.Name
			} else if req.CategoryID != nil {
				reqName = "Категория"
			} else {
				reqName = "Ингредиент"
			}
		}
		reqKey := normalizeIngredientKey(reqName)
		q := req.Quantity
		have := ingredientAvail[reqKey]
		if have > 0 {
			matched++
		}
		if have < q {
			makeable = false
			short := q - have
			mi := MissingIngredient{
				NeededIngredient:   reqName,
				NeededCategoryName: reqName,
				QtyShort:           short,
			}
			if req.CategoryID != nil {
				mi.NeededCategoryID = *req.CategoryID
			}
			var sugg []models.Product
			qb := s.DB.Model(&models.Product{})
			if req.CategoryID != nil {
				qb = qb.Where("category_id = ?", *req.CategoryID)
			}
			if reqName != "" {
				qb = qb.Where("LOWER(name) LIKE ?", "%"+strings.ToLower(reqName)+"%")
			}
			_ = qb.Order("default_price ASC").Limit(15).Find(&sugg).Error
			mi.Suggestions = sugg
			miss = append(miss, mi)
		}

		avg := ingredientAvgNutrition(s, req, ingredientNutrition[reqKey])
		mult := float64(q)
		kcal += avg.CaloriesKcal * mult
		prot += avg.ProteinG * mult
		fat += avg.FatG * mult
		carbs += avg.CarbsG * mult
	}

	score := nutritionScore(kcal, prot, fat, carbs, mealTargetKcal)
	matchRatio := 0.0
	if required > 0 {
		matchRatio = float64(matched) / float64(required)
	}

	return OrderMealRecommendation{
		Dish:           d,
		Score:          score,
		Makeable:       makeable,
		MatchedItems:   matched,
		RequiredItems:  required,
		MatchRatio:     matchRatio,
		TotalKcal:      kcal,
		TotalProteinG:  prot,
		TotalFatG:      fat,
		TotalCarbsG:    carbs,
		Missing:        miss,
		MealTargetKcal: mealTargetKcal,
	}
}

func (s *RecommendationService) scoreDishByProducts(d models.Dish, avail map[int64]int, mealTargetKcal float64) OrderMealRecommendation {
	var miss []MissingIngredient
	makeable := true
	matched := 0
	required := len(d.Products)
	var kcal, prot, fat, carbs float64

	for _, req := range d.Products {
		pid := req.ProductID
		need := req.Quantity
		have := avail[pid]
		if have > 0 {
			matched++
		}

		var p models.Product
		if req.Product != nil {
			p = *req.Product
		} else {
			_ = s.DB.First(&p, pid).Error
		}

		if have < need {
			makeable = false
			short := need - have
			name := p.Name
			if strings.TrimSpace(name) == "" {
				name = "Товар"
			}
			mi := MissingIngredient{
				NeededIngredient:   name,
				NeededCategoryID:   pid,
				NeededCategoryName: name,
				QtyShort:           short,
			}
			sugg := make([]models.Product, 0, 1)
			if p.ID > 0 {
				sugg = append(sugg, p)
			}
			mi.Suggestions = sugg
			miss = append(miss, mi)
		}

		kcal += p.CaloriesKcal * float64(need)
		prot += p.ProteinG * float64(need)
		fat += p.FatG * float64(need)
		carbs += p.CarbsG * float64(need)
	}

	score := nutritionScore(kcal, prot, fat, carbs, mealTargetKcal)
	matchRatio := 0.0
	if required > 0 {
		matchRatio = float64(matched) / float64(required)
	}
	return OrderMealRecommendation{
		Dish:           d,
		Score:          score,
		Makeable:       makeable,
		MatchedItems:   matched,
		RequiredItems:  required,
		MatchRatio:     matchRatio,
		TotalKcal:      kcal,
		TotalProteinG:  prot,
		TotalFatG:      fat,
		TotalCarbsG:    carbs,
		Missing:        miss,
		MealTargetKcal: mealTargetKcal,
	}
}

func ingredientAvgNutrition(s *RecommendationService, req models.DishCategoryRequirement, agg categoryNutritionAgg) models.Product {
	if agg.Count > 0 {
		cnt := float64(agg.Count)
		return models.Product{
			CaloriesKcal: agg.Kcal / cnt,
			ProteinG:     agg.Prot / cnt,
			FatG:         agg.Fat / cnt,
			CarbsG:       agg.Carbs / cnt,
		}
	}

	// Если в корзине нет подходящего товара, берём усреднение по каталогу для ингредиента.
	var rows []models.Product
	qb := s.DB.Model(&models.Product{})
	if req.CategoryID != nil {
		qb = qb.Where("category_id = ?", *req.CategoryID)
	}
	if strings.TrimSpace(req.IngredientName) != "" {
		qb = qb.Where("LOWER(name) LIKE ?", "%"+strings.ToLower(strings.TrimSpace(req.IngredientName))+"%")
	}
	if err := qb.Limit(30).Find(&rows).Error; err != nil || len(rows) == 0 {
		return models.Product{}
	}
	var kcal, prot, fat, carbs float64
	for _, p := range rows {
		kcal += p.CaloriesKcal
		prot += p.ProteinG
		fat += p.FatG
		carbs += p.CarbsG
	}
	cnt := float64(len(rows))
	return models.Product{
		CaloriesKcal: kcal / cnt,
		ProteinG:     prot / cnt,
		FatG:         fat / cnt,
		CarbsG:       carbs / cnt,
	}
}

var packageTokenRe = regexp.MustCompile(`(?i)\b\d+[.,]?\d*\s*(кг|г|гр|мг|мл|л|шт|уп|упаковк\w*|pack|pcs)\b`)
var extraSpaceRe = regexp.MustCompile(`\s+`)

func normalizeIngredientKey(name string) string {
	s := strings.ToLower(strings.TrimSpace(name))
	if s == "" {
		return ""
	}
	s = packageTokenRe.ReplaceAllString(s, " ")
	s = strings.NewReplacer("(", " ", ")", " ", ",", " ", ".", " ").Replace(s)
	s = extraSpaceRe.ReplaceAllString(strings.TrimSpace(s), " ")
	return s
}

// nutritionScore: выше — лучше баланс БЖУ и близость калорий к цели приёма пищи.
func nutritionScore(kcal, prot, fat, carbs, mealTarget float64) float64 {
	if kcal <= 0 {
		kcal = 1
	}
	pCal := prot * 4
	fCal := fat * 9
	cCal := carbs * 4
	sumMacro := pCal + fCal + cCal
	if sumMacro <= 0 {
		sumMacro = 1
	}
	// Целевые доли калорий: белки ~25%, жиры ~30%, углеводы ~45%
	pp := pCal / sumMacro
	ff := fCal / sumMacro
	cc := cCal / sumMacro
	balance := 1.0 - (math.Abs(pp-0.25) + math.Abs(ff-0.30) + math.Abs(cc-0.45))
	calFit := 1.0 - math.Min(1.0, math.Abs(kcal-mealTarget)/mealTarget)
	return balance*50 + calFit*50
}
