package services

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
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
