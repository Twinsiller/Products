package services

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"

	"Products_backend/internal/models"
)

func buildPythonDBURL() string {
	if v := os.Getenv("RECOMMENDER_DATABASE_URL"); v != "" {
		return v
	}

	host := os.Getenv("POSTGRES_HOST")
	if host == "" {
		host = "localhost"
	}
	port := os.Getenv("POSTGRES_PORT")
	if port == "" {
		port = "5432"
	}
	user := os.Getenv("POSTGRES_USER")
	if user == "" {
		user = "postgres"
	}
	password := os.Getenv("POSTGRES_PASSWORD")
	if password == "" {
		password = "postgres"
	}
	dbname := os.Getenv("POSTGRES_DB")
	if dbname == "" {
		dbname = "postgres"
	}

	return fmt.Sprintf(
		"postgresql+psycopg2://%s:%s@%s:%s/%s",
		url.QueryEscape(user),
		url.QueryEscape(password),
		host,
		port,
		dbname,
	)
}

func (s *RecommendationService) RunFinalRecipeRecommender(userID int64, limit int) (map[string]interface{}, error) {
	if limit <= 0 {
		limit = 5
	}

	cwd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("getwd: %w", err)
	}
	scriptPath := filepath.Join(cwd, "ml", "recommender.py")
	if _, err := os.Stat(scriptPath); err != nil {
		return nil, fmt.Errorf("recommender script not found at %s", scriptPath)
	}

	dbURL := buildPythonDBURL()
	baseArgs := []string{
		scriptPath,
		"--db-url", dbURL,
		"--user-id", strconv.FormatInt(userID, 10),
		"--k", strconv.Itoa(limit),
	}

	type candidate struct {
		bin  string
		args []string
	}
	candidates := []candidate{}
	if custom := os.Getenv("PYTHON_BIN"); custom != "" {
		candidates = append(candidates, candidate{bin: custom, args: baseArgs})
	}
	candidates = append(candidates, candidate{bin: "python", args: baseArgs})
	candidates = append(candidates, candidate{bin: "python3", args: baseArgs})
	if runtime.GOOS == "windows" {
		candidates = append(candidates, candidate{bin: "py", args: append([]string{"-3"}, baseArgs...)})
	}

	var lastErr error
	for _, c := range candidates {
		cmd := exec.Command(c.bin, c.args...)
		cmd.Dir = cwd
		// Force UTF-8 output from Python on Windows to avoid mojibake in JSON payload.
		cmd.Env = append(os.Environ(),
			"PYTHONUTF8=1",
			"PYTHONIOENCODING=utf-8",
		)
		stdout, err := cmd.Output()
		if err != nil {
			if ee, ok := err.(*exec.ExitError); ok {
				lastErr = fmt.Errorf("%s failed: %s", c.bin, string(ee.Stderr))
			} else {
				lastErr = err
			}
			continue
		}

		var out map[string]interface{}
		if err := json.Unmarshal(stdout, &out); err != nil {
			lastErr = fmt.Errorf("failed to parse recommender output: %w; output=%s", err, string(stdout))
			continue
		}
		return out, nil
	}

	if lastErr == nil {
		lastErr = fmt.Errorf("failed to run python recommender")
	}
	return nil, lastErr
}

type foodRecommendationItem struct {
	ProductID  int64   `json:"product_id"`
	FinalScore float64 `json:"final_score"`
	CBScore    float64 `json:"cb_score"`
	CFScore    float64 `json:"cf_score"`
	MealScore  float64 `json:"meal_score"`
	Recency    float64 `json:"recency_score"`
	Reason     string  `json:"reason"`
	Linked     []struct {
		DishID                     int64   `json:"dish_id"`
		DishName                   string  `json:"dish_name"`
		DishScore                  float64 `json:"dish_score"`
		MissingIngredientsEstimate int     `json:"missing_ingredients_estimate"`
	} `json:"linked_dishes"`
}

type foodRecommendationResponse struct {
	Recommendations []foodRecommendationItem `json:"recommendations"`
}

// RunProductRecommender запускает Python-рекомендер товаров и возвращает товары со скорами.
func (s *RecommendationService) RunProductRecommender(userID int64, limit int) ([]ProductRecommendation, error) {
	if limit <= 0 {
		limit = 10
	}

	cwd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("getwd: %w", err)
	}
	scriptPath := filepath.Join(cwd, "ml", "food_recommender.py")
	if _, err := os.Stat(scriptPath); err != nil {
		return nil, fmt.Errorf("product recommender script not found at %s", scriptPath)
	}

	dbURL := buildPythonDBURL()
	baseArgs := []string{
		scriptPath,
		"--db-url", dbURL,
		"--user-id", strconv.FormatInt(userID, 10),
		"--k", strconv.Itoa(limit),
	}

	type candidate struct {
		bin  string
		args []string
	}
	candidates := []candidate{}
	if custom := os.Getenv("PYTHON_BIN"); custom != "" {
		candidates = append(candidates, candidate{bin: custom, args: baseArgs})
	}
	candidates = append(candidates, candidate{bin: "python", args: baseArgs})
	candidates = append(candidates, candidate{bin: "python3", args: baseArgs})
	if runtime.GOOS == "windows" {
		candidates = append(candidates, candidate{bin: "py", args: append([]string{"-3"}, baseArgs...)})
	}

	var parsed foodRecommendationResponse
	var lastErr error
	for _, c := range candidates {
		cmd := exec.Command(c.bin, c.args...)
		cmd.Dir = cwd
		cmd.Env = append(os.Environ(), "PYTHONUTF8=1", "PYTHONIOENCODING=utf-8")
		stdout, err := cmd.Output()
		if err != nil {
			if ee, ok := err.(*exec.ExitError); ok {
				lastErr = fmt.Errorf("%s failed: %s", c.bin, string(ee.Stderr))
			} else {
				lastErr = err
			}
			continue
		}
		if err := json.Unmarshal(stdout, &parsed); err != nil {
			lastErr = fmt.Errorf("failed to parse food recommender output: %w; output=%s", err, string(stdout))
			continue
		}
		break
	}
	if len(parsed.Recommendations) == 0 {
		if lastErr == nil {
			lastErr = fmt.Errorf("food recommender returned no recommendations")
		}
		return nil, lastErr
	}

	byID := make(map[int64]float64, len(parsed.Recommendations))
	byItem := make(map[int64]foodRecommendationItem, len(parsed.Recommendations))
	ids := make([]int64, 0, len(parsed.Recommendations))
	for _, r := range parsed.Recommendations {
		if r.ProductID <= 0 {
			continue
		}
		if _, ok := byID[r.ProductID]; !ok {
			ids = append(ids, r.ProductID)
		}
		byID[r.ProductID] = r.FinalScore
		byItem[r.ProductID] = r
	}
	if len(ids) == 0 {
		return []ProductRecommendation{}, nil
	}

	var products []models.Product
	if err := s.DB.Where("id IN ?", ids).Find(&products).Error; err != nil {
		return nil, err
	}

	out := make([]ProductRecommendation, 0, len(products))
	for _, p := range products {
		srcScore := byID[p.ID]
		src := byItem[p.ID]
		linked := make([]ProductRecommendationDishRef, 0, len(src.Linked))
		for _, ld := range src.Linked {
			linked = append(linked, ProductRecommendationDishRef{
				DishID:                     ld.DishID,
				DishName:                   ld.DishName,
				DishScore:                  ld.DishScore,
				MissingIngredientsEstimate: ld.MissingIngredientsEstimate,
			})
		}
		out = append(out, ProductRecommendation{
			Product:      p,
			Score:        srcScore,
			CBScore:      src.CBScore,
			CFScore:      src.CFScore,
			MealScore:    src.MealScore,
			RecencyScore: src.Recency,
			Reason:       src.Reason,
			LinkedDishes: linked,
		})
	}

	sort.Slice(out, func(i, j int) bool {
		return out[i].Score > out[j].Score
	})
	if len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}
