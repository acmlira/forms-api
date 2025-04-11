package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"forms/internal/logger"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"slices"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/labstack/echo/v4"
)

type CreateFormRequest struct {
	Question string `json:"question" validate:"required"`
	Answer   string `json:"answer" validate:"required"`
}

// CreateForm godoc
// @Summary      Create a new form
// @Description  Creates a new form with status "draft"
// @Tags         forms
// @Accept       json
// @Produce      json
// @Param        form  body  CreateFormRequest  true  "Form to be created"
// @Success      201   {object}  map[string]string
// @Failure      400   {object}  map[string]string
// @Failure      500   {object}  map[string]string
// @Router       /v1/forms [post]
func CreateForm(db *pgxpool.Pool) echo.HandlerFunc {
	return func(c echo.Context) error {
		var req CreateFormRequest
		if err := c.Bind(&req); err != nil || strings.TrimSpace(req.Question) == "" {
			return c.JSON(http.StatusBadRequest, echo.Map{"error": "invalid request body"})
		}

		id := uuid.New().String()
		now := time.Now()

		urgency := measureUrgency(context.Background(), req.Answer)

		_, err := db.Exec(context.Background(), `
			INSERT INTO forms (id, question, answer, status, created_at, updated_at, urgency )
			VALUES ($1, $2, $3, 'draft', $4, $5, $6)
		`, id, req.Question, req.Answer, now, now, &urgency)
		if err != nil {
			logger.Error("failed to insert form", err)
			return c.JSON(http.StatusInternalServerError, echo.Map{"error": "failed to create form"})
		}

		return c.JSON(http.StatusCreated, echo.Map{"form_id": id})
	}
}

// ListForms godoc
// @Summary      List forms with filters and sorting
// @Description  Returns paginated list of forms filtered by any field (question, answer, urgency, status) and sorted
// @Tags         forms
// @Accept       json
// @Produce      json
// @Param        page     query  int     false  "Page number"     default(1)
// @Param        limit    query  int     false  "Items per page"  default(10)
// @Param        question query  string  false  "Filter by question"
// @Param        answer   query  string  false  "Filter by answer"
// @Param        urgency  query  string  false  "Filter by urgency"
// @Param        status   query  string  false  "Filter by status"
// @Param        sort     query  string  false  "Sort by field name"
// @Param        direction query string  false  "Sort direction: asc or desc"
// @Success      200  {object}  map[string]interface{}
// @Failure      500  {object}  map[string]string
// @Router       /v1/forms [get]
func ListForms(db *pgxpool.Pool) echo.HandlerFunc {
	return func(c echo.Context) error {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		page, _ := strconv.Atoi(c.QueryParam("page"))
		limit, _ := strconv.Atoi(c.QueryParam("limit"))
		if page < 1 {
			page = 1
		}
		if limit < 1 || limit > 100 {
			limit = 10
		}
		offset := (page - 1) * limit

		filters := []string{}
		args := []interface{}{}
		i := 1

		for _, field := range []string{"question", "answer", "urgency", "status"} {
			value := c.QueryParam(field)
			if value != "" {
				filters = append(filters, field+" ILIKE $"+strconv.Itoa(i))
				args = append(args, "%"+value+"%")
				i++
			}
		}

		where := ""
		if len(filters) > 0 {
			where = "WHERE " + strings.Join(filters, " AND ")
		}

		sort := c.QueryParam("sort")
		direction := strings.ToUpper(c.QueryParam("direction"))
		if direction != "ASC" && direction != "DESC" {
			direction = "DESC"
		}
		order := "ORDER BY CASE urgency WHEN 'high' THEN 1 WHEN 'medium' THEN 2 WHEN 'low' THEN 3 ELSE 4 END "
		if sort != "" {
			if sort == "urgency" {
				order += direction
			}
			if sort == "question" || sort == "answer" || sort == "status" || sort == "created_at" {
				order = "ORDER BY " + sort + " " + direction
			}
		}

		query := `
			SELECT id, question, answer, urgency, status, created_at, updated_at
			FROM forms
			` + where + `
			` + order + `
			LIMIT $` + strconv.Itoa(i) + ` OFFSET $` + strconv.Itoa(i+1)
		args = append(args, limit, offset)

		rows, err := db.Query(ctx, query, args...)
		if err != nil {
			logger.Error("query error", nil)
			return c.JSON(http.StatusInternalServerError, echo.Map{"error": "query failed"})
		}
		defer rows.Close()

		var results []map[string]interface{}
		for rows.Next() {
			var id, question, status string
			var answer, urgency *string
			var createdAt, updatedAt time.Time
			if err := rows.Scan(&id, &question, &answer, &urgency, &status, &createdAt, &updatedAt); err != nil {
				logger.Error("scan error", nil)
				return c.JSON(http.StatusInternalServerError, echo.Map{"error": "scan error"})
			}
			results = append(results, echo.Map{
				"id":         id,
				"question":   question,
				"answer":     answer,
				"urgency":    urgency,
				"status":     status,
				"created_at": createdAt,
				"updated_at": updatedAt,
			})
		}

		return c.JSON(http.StatusOK, echo.Map{
			"page":    page,
			"limit":   limit,
			"results": results,
		})
	}
}

type UpdateFormRequest struct {
	Answer *string `json:"answer"`
	Status string  `json:"status"`
}

// UpdateForm godoc
// @Summary      Update a form's answer and status
// @Description  Update the answer field and change status based on business rules
// @Tags         forms
// @Accept       json
// @Produce      json
// @Param        id    path   string              true  "Form ID"
// @Param        body  body   UpdateFormRequest   true  "Update payload"
// @Success      200   {object}  map[string]string
// @Failure      400   {object}  map[string]string
// @Failure      500   {object}  map[string]string
// @Router       /v1/forms/{id} [put]
func UpdateForm(db *pgxpool.Pool) echo.HandlerFunc {
	return func(c echo.Context) error {
		id := c.Param("id")
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		var req UpdateFormRequest
		if err := c.Bind(&req); err != nil {
			return c.JSON(http.StatusBadRequest, echo.Map{"error": "invalid payload"})
		}

		validTransitions := map[string][]string{
			"draft":    {"closed", "filled"},
			"filled":   {"closed", "reviewed"},
			"reviewed": {"closed"},
			"closed":   {"draft"},
		}

		var currentStatus string
		err := db.QueryRow(ctx, "SELECT status FROM forms WHERE id = $1", id).Scan(&currentStatus)
		if err != nil {
			return c.JSON(http.StatusBadRequest, echo.Map{"error": "form not found"})
		}

		allowed := slices.Contains(validTransitions[currentStatus], req.Status)
		if !allowed {
			return c.JSON(http.StatusBadRequest, echo.Map{"error": "invalid status transition"})
		}

		var urgency *string
		if req.Answer != nil {
			urg := measureUrgency(c.Request().Context(), *req.Answer)
			urgency = &urg
		}

		_, err = db.Exec(ctx, `
			UPDATE forms
			SET answer = $1, urgency = $2, status = $3, updated_at = NOW()
			WHERE id = $4
		`, req.Answer, urgency, req.Status, id)
		if err != nil {
			logger.Error("update error", err)
			return c.JSON(http.StatusInternalServerError, echo.Map{"error": "failed to update form"})
		}

		return c.JSON(http.StatusOK, echo.Map{"message": "form updated"})
	}
}

func measureUrgency(ctx context.Context, text string) string {
	urgency := "medium"

	host := os.Getenv("EMOTION_API_HOST")
	if host == "" {
		logger.Fatal("EMOTION_API_HOST not set", nil)
	}

	payload := strings.NewReader(`{ "text": "` + text + `"}`)

	url := fmt.Sprintf("%s/v1/analysis", host)
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, url, payload)
	if err != nil {
		return urgency
	}
	request.Header.Set("Content-Type", "application/json")

	client := &http.Client{}

	response, err := client.Do(request)
	if err != nil {
		return urgency
	}

	if response.StatusCode != 200 {
		return urgency
	}

	var output output
	err = json.NewDecoder(response.Body).Decode(&output)
	if err != nil {
		return urgency
	}

	score := 0.0

	for _, res := range output.FullResult {
		switch res.Label {
		case "joy", "surprise":
			score -= res.Score * 1.5
		case "neutral":
			score += res.Score * 0.2
		case "sadness", "anger", "disgust", "fear":
			score += res.Score * 1.2
		}
	}

	switch {
	case score >= 0.6:
		urgency = "high"
	case score >= 0.2:
		urgency = "medium"
	default:
		urgency = "low"
	}

	return urgency
}

// urgencyMap := map[string]string{
// 	"anger":    "high",
// 	"disgust":  "high",
// 	"fear":     "high",
// 	"neutral":  "medium",
// 	"surprise": "medium",
// 	"joy":      "low",
// 	"sadness":  "low",
// }

// return urgencyMap[output.Emotion]

type output struct {
	Emotion    string `json:"emotion"`
	FullResult []struct {
		Label string  `json:"label"`
		Score float64 `json:"score"`
	} `json:"full_result"`
	Score float64 `json:"score"`
}
