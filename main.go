package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"
)

var (
	username = os.Getenv("LEETCODE_USERNAME")
	session  = os.Getenv("LEETCODE_SESSION")
)

func main() {
	submissions := getTodayAcceptedSubmissions()
	fmt.Printf("\n✅ Solved today: %d problems\n\n", len(submissions))

	for _, sub := range submissions {
		fmt.Println("🔹", sub.Title)

		description := getProblemDescription(sub.TitleSlug)
		code := getSubmissionCodeByID(sub.ID)

		err := insertSubmissionToSupabase(sub, code, description)
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

func insertSubmissionToSupabase(sub Submission, code string, description string) error {
	supabaseUrl := os.Getenv("SUPABASE_URL")
	supabaseKey := os.Getenv("SUPABASE_ANON_KEY")

	client := &http.Client{}
	timestampInt, _ := strconv.ParseInt(sub.Timestamp, 10, 64)
	timestamp := time.Unix(timestampInt, 0).Format(time.RFC3339)

	// UPSERT leetcode_questions
	questionPayload := map[string]interface{}{
		"slug":        sub.TitleSlug,
		"title":       sub.Title,
		"description": description,
	}

	qPayloadBytes, _ := json.Marshal(questionPayload)
	qReq, _ := http.NewRequest("POST", supabaseUrl+"/rest/v1/leetcode_questions", bytes.NewBuffer(qPayloadBytes))
	qReq.Header.Set("apikey", supabaseKey)
	qReq.Header.Set("Authorization", "Bearer "+supabaseKey)
	qReq.Header.Set("Content-Type", "application/json")
	qReq.Header.Set("Prefer", "resolution=merge-duplicates") // enables UPSERT

	qRes, err := client.Do(qReq)
	if err != nil || qRes.StatusCode >= 300 {
		return fmt.Errorf("question upsert failed: %v", err)
	}
	defer qRes.Body.Close()

	// 2. Insert leetcode_submissions
	subPayload := map[string]interface{}{
		"submission_id": sub.ID,
		"question_slug": sub.TitleSlug,
		"submitted_at":  timestamp,
		"code":          code,
	}

	sPayloadBytes, _ := json.Marshal(subPayload)
	sReq, _ := http.NewRequest("POST", supabaseUrl+"/rest/v1/leetcode_submissions", bytes.NewBuffer(sPayloadBytes))
	sReq.Header.Set("apikey", supabaseKey)
	sReq.Header.Set("Authorization", "Bearer "+supabaseKey)
	sReq.Header.Set("Content-Type", "application/json")

	sRes, err := client.Do(sReq)
	if err != nil || sRes.StatusCode >= 300 {
		return fmt.Errorf("submission insert failed: %v", err)
	}
	defer sRes.Body.Close()

	return nil
}

