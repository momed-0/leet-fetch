package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"
)

var (
	username = os.Getenv("LEETCODE_USERNAME")
	session  = os.Getenv("LEETCODE_SESSION")
	connStr  = os.Getenv("DATABASE_URL")
)

func main() {

	conn, err := pgx.Connect(context.Background(), connStr)
	if err != nil {
		fmt.Println("❌ Failed to connect to Supabase:", err)
		os.Exit(1)
	}
	defer conn.Close(context.Background())
	fmt.Println("✅ Connected to Supabase")

	submissions := getTodayAcceptedSubmissions()
	fmt.Printf("\n✅ Solved today: %d problems\n\n", len(submissions))

	for _, sub := range submissions {
		fmt.Println("🔹", sub.Title)

		description := getProblemDescription(sub.TitleSlug)
		code := getSubmissionCodeByID(sub.ID)

		err := insertSubmissionToDB(conn, sub, code, description)
		if err != nil {
			fmt.Println("❌ Error inserting:", err)
		} else {
			fmt.Println("✅ Inserted to DB")
		}
		time.Sleep(1 * time.Second) // be kind to LeetCode
	}
}

type Submission struct {
	ID        string `json:"id"`
	Title     string `json:"title"`
	TitleSlug string `json:"titleSlug"`
	Timestamp string `json:"timestamp"`
}

func getTodayAcceptedSubmissions() []Submission {
	query := `
	query recentAcSubmissions($username: String!, $limit: Int!) {
		recentAcSubmissionList(username: $username, limit: $limit) {
			id
			title
			titleSlug
			timestamp
		}
	}`

	variables := map[string]interface{}{
		"username": username,
		"limit":    50,
	}

	body := map[string]interface{}{
		"query":     query,
		"variables": variables,
	}

	resp := graphqlRequest(body)
	defer resp.Body.Close()

	type RespData struct {
		Data struct {
			RecentSubmissionList []Submission `json:"recentAcSubmissionList"`
		} `json:"data"`
	}

	var data RespData
	json.NewDecoder(resp.Body).Decode(&data)

	today := time.Now().UTC().Format("2006-01-02")
	var todayAccepted []Submission
	for _, s := range data.Data.RecentSubmissionList {
		tsInt, _ := stringToInt64(s.Timestamp)
		t := time.Unix(tsInt, 0).UTC().Format("2006-01-02")
		if t == today {
			todayAccepted = append(todayAccepted, s)
		}
	}

	return todayAccepted
}
func getSubmissionCodeByID(id string) string {
	query := `
	query submissionDetails($submissionId: Int!) {
		submissionDetails(submissionId: $submissionId) {
			code
		}
	}`

	variables := map[string]interface{}{
		"submissionId": id,
	}

	body := map[string]interface{}{
		"query":     query,
		"variables": variables,
	}

	resp := graphqlRequest(body)
	defer resp.Body.Close()

	type RespData struct {
		Data struct {
			SubmissionDetails struct {
				Code string `json:"code"`
			} `json:"submissionDetails"`
		} `json:"data"`
	}

	var data RespData
	json.NewDecoder(resp.Body).Decode(&data)

	return data.Data.SubmissionDetails.Code
}

func getProblemDescription(slug string) string {
	query := `
	query questionContent($titleSlug: String!) {
		question(titleSlug: $titleSlug) {
			content
		}
	}`

	variables := map[string]interface{}{
		"titleSlug": slug,
	}

	body := map[string]interface{}{
		"query":     query,
		"variables": variables,
	}

	resp := graphqlRequest(body)
	defer resp.Body.Close()

	type RespData struct {
		Data struct {
			Question struct {
				Content string `json:"content"`
			} `json:"question"`
		} `json:"data"`
	}

	var data RespData
	json.NewDecoder(resp.Body).Decode(&data)

	return data.Data.Question.Content
}

func graphqlRequest(body map[string]interface{}) *http.Response {
	jsonBody, _ := json.Marshal(body)
	req, _ := http.NewRequest("POST", "https://leetcode.com/graphql", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Cookie", "LEETCODE_SESSION="+session)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("❌ Failed to connect to LeetCode:", err)
		os.Exit(1)
	}
	return resp
}

func stringToInt64(s string) (int64, error) {
	return strconv.ParseInt(s, 10, 64)
}

func insertSubmissionToDB(conn *pgx.Conn, sub Submission, code string, description string) error {
	timestampInt, _ := strconv.ParseInt(sub.Timestamp, 10, 64)
	timestamp := time.Unix(timestampInt, 0)

	_, err := conn.Exec(context.Background(), `
	INSERT INTO leetcode_questions (slug, title, description)
		VALUES ($1, $2, $3)
		ON CONFLICT (slug) DO UPDATE
		SET description = EXCLUDED.description
	`, sub.TitleSlug, sub.Title, description)
	if err != nil {
		return err
	}
	_, err = conn.Exec(context.Background(), `
		INSERT INTO leetcode_submissions (
			submission_id, question_slug, title, submitted_at, language, status, code, description
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, sub.ID, sub.TitleSlug, sub.Title, timestamp, "C++", "Accepted", code, description)

	return err
}
