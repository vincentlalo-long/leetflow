package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"
)

const (
	graphqlURL = "https://leetcode.com/graphql"
	problemsURL = "https://leetcode.com/api/problems/all/"
	userAgent   = "Mozilla/5.0"
)

type client struct {
	http *http.Client
}

var Client = &client{http: &http.Client{Timeout: 15 * time.Second}}

func doPost(url string, payload map[string]interface{}, result interface{}) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	req, err := http.NewRequest("POST", url, strings.NewReader(string(body)))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Referer", "https://leetcode.com")
	req.Header.Set("User-Agent", userAgent)

	resp, err := Client.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, result)
}

func doGet(url string, result interface{}) error {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", userAgent)

	resp, err := Client.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, result)
}

type CodeSnippet struct {
	Lang     string `json:"lang"`
	LangSlug string `json:"langSlug"`
	Code     string `json:"code"`
}

type ProblemDetail struct {
	QuestionID         string        `json:"questionId"`
	QuestionFrontendID string        `json:"questionFrontendId"`
	Title              string        `json:"title"`
	Difficulty         string        `json:"difficulty"`
	Content            string        `json:"content"`
	Hints              []string      `json:"hints"`
	SimilarQuestions   string        `json:"similarQuestions"`
	TopicTags          []TopicTag    `json:"topicTags"`
	CodeSnippets       []CodeSnippet `json:"codeSnippets"`
}

func (p *ProblemDetail) GetCodeSnippet(langKey string) string {
	targetSlugs := []string{langKey}
	switch langKey {
	case "python":
		targetSlugs = []string{"python3", "python"}
	case "go":
		targetSlugs = []string{"golang", "go"}
	case "csharp":
		targetSlugs = []string{"csharp", "cs"}
	case "cpp":
		targetSlugs = []string{"cpp", "c++"}
	}

	for _, snippet := range p.CodeSnippets {
		for _, s := range targetSlugs {
			if strings.EqualFold(snippet.LangSlug, s) {
				return snippet.Code
			}
		}
	}
	return ""
}

type TopicTag struct {
	Name string `json:"name"`
}

type dailyResp struct {
	Data struct {
		ActiveDailyCodingChallengeQuestion struct {
			Date     string `json:"date"`
			UserStatus string `json:"userStatus"`
			Link     string `json:"link"`
			Question struct {
				QuestionID         string     `json:"questionId"`
				QuestionFrontendID string     `json:"questionFrontendId"`
				Title              string     `json:"title"`
				TitleSlug          string     `json:"titleSlug"`
				Difficulty         string     `json:"difficulty"`
				TopicTags          []TopicTag `json:"topicTags"`
			} `json:"question"`
		} `json:"activeDailyCodingChallengeQuestion"`
	} `json:"data"`
}

type problemDetailResp struct {
	Data struct {
		Question *ProblemDetail `json:"question"`
	} `json:"data"`
}

type ProblemInfo struct {
	Title string `json:"title"`
	Slug  string `json:"slug"`
}

type allProblemsResp struct {
	StatStatusPairs []struct {
		Stat struct {
			FrontendQuestionID int    `json:"frontend_question_id"`
			QuestionTitle      string `json:"question__title"`
			QuestionTitleSlug  string `json:"question__title_slug"`
		} `json:"stat"`
		Difficulty struct {
			Level int `json:"level"`
		} `json:"difficulty"`
		PaidOnly bool `json:"paid_only"`
	} `json:"stat_status_pairs"`
}

type UserProfileResp struct {
	Data struct {
		MatchedUser *struct {
			Username    string `json:"username"`
			SubmitStats *struct {
				AcSubmissionNum []struct {
					Difficulty string `json:"difficulty"`
					Count      int    `json:"count"`
				} `json:"acSubmissionNum"`
			} `json:"submitStats"`
			Profile *struct {
				Ranking    float64 `json:"ranking"`
				Reputation int     `json:"reputation"`
			} `json:"profile"`
		} `json:"matchedUser"`
	} `json:"data"`
}

type ContestInfo struct {
	Title     string `json:"title"`
	TitleSlug string `json:"titleSlug"`
	StartTime int64  `json:"startTime"`
	Duration  int64  `json:"duration"`
}

type contestsResp struct {
	Data struct {
		TopTwoContests []ContestInfo `json:"topTwoContests"`
	} `json:"data"`
}

func GetProblemDetails(slug string) (*ProblemDetail, error) {
	query := `query getQuestionDetail($titleSlug: String!) {
		question(titleSlug: $titleSlug) {
			questionId, questionFrontendId, title, difficulty, content, hints, similarQuestions,
			topicTags { name },
			codeSnippets { lang, langSlug, code }
		}
	}`
	payload := map[string]interface{}{
		"query": query,
		"variables": map[string]string{"titleSlug": slug},
	}
	var resp problemDetailResp
	if err := doPost(graphqlURL, payload, &resp); err != nil {
		return nil, err
	}
	if resp.Data.Question == nil {
		return nil, fmt.Errorf("problem not found")
	}
	return resp.Data.Question, nil
}

func GetDailyChallenge() (*ProblemDetail, string, string, error) {
	query := `query questionOfToday {
		activeDailyCodingChallengeQuestion {
			date, userStatus, link,
			question { questionId, questionFrontendId, title, titleSlug, difficulty, topicTags { name } }
		}
	}`
	payload := map[string]interface{}{"query": query}
	var resp dailyResp
	if err := doPost(graphqlURL, payload, &resp); err != nil {
		return nil, "", "", err
	}
	q := resp.Data.ActiveDailyCodingChallengeQuestion.Question
	detail := &ProblemDetail{
		QuestionID:         q.QuestionID,
		QuestionFrontendID: q.QuestionFrontendID,
		Title:              q.Title,
		Difficulty:         q.Difficulty,
		TopicTags:          q.TopicTags,
	}
	return detail, resp.Data.ActiveDailyCodingChallengeQuestion.Link, resp.Data.ActiveDailyCodingChallengeQuestion.Date, nil
}

func Slugify(text string) string {
	reg := regexp.MustCompile(`[^a-z0-9]+`)
	slug := reg.ReplaceAllString(strings.ToLower(text), "-")
	return strings.Trim(slug, "-")
}

func GetProblemByID(frontendID string) (*ProblemInfo, error) {
	var resp allProblemsResp
	if err := doGet(problemsURL, &resp); err != nil {
		return nil, err
	}
	cleanID := strings.TrimLeft(frontendID, "0")
	for _, p := range resp.StatStatusPairs {
		apiID := fmt.Sprintf("%d", p.Stat.FrontendQuestionID)
		if apiID == cleanID {
			return &ProblemInfo{
				Title: p.Stat.QuestionTitle,
				Slug:  p.Stat.QuestionTitleSlug,
			}, nil
		}
	}
	return nil, fmt.Errorf("problem %s not found", frontendID)
}

func GetAllProblems() (*allProblemsResp, error) {
	var resp allProblemsResp
	if err := doGet(problemsURL, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func GetUserProfile(username string) (*UserProfileResp, error) {
	query := `query getUserProfile($username: String!) {
		matchedUser(username: $username) {
			username
			submitStats: submitStatsGlobal {
				acSubmissionNum { difficulty, count }
			}
			profile { ranking, reputation, starRating }
		}
	}`
	payload := map[string]interface{}{
		"query": query,
		"variables": map[string]string{"username": username},
	}
	var resp UserProfileResp
	if err := doPost(graphqlURL, payload, &resp); err != nil {
		return nil, err
	}
	if resp.Data.MatchedUser == nil {
		return nil, fmt.Errorf("user %s not found", username)
	}
	return &resp, nil
}

func GetUpcomingContests() ([]ContestInfo, error) {
	query := `query { topTwoContests { title, titleSlug, startTime, duration } }`
	payload := map[string]interface{}{"query": query}
	var resp contestsResp
	if err := doPost(graphqlURL, payload, &resp); err != nil {
		return nil, err
	}
	return resp.Data.TopTwoContests, nil
}

type SubmissionCheckResult struct {
	State             string   `json:"state"`
	StatusCode        int      `json:"status_code"`
	StatusMsg         string   `json:"status_msg"`
	StatusRuntime     string   `json:"status_runtime"`
	Memory            int64    `json:"memory"`
	StatusMemory      string   `json:"status_memory"`
	RuntimePercentile float64  `json:"runtime_percentile"`
	MemoryPercentile  float64  `json:"memory_percentile"`
	TotalCorrect      int      `json:"total_correct"`
	TotalTestcases    int      `json:"total_testcases"`
	FullCompileError  string   `json:"full_compile_error"`
	CompileError      string   `json:"compile_error"`
	FullRuntimeError  string   `json:"full_runtime_error"`
	RuntimeError      string   `json:"runtime_error"`
	CodeOutput        []string `json:"code_output"`
	ExpectedOutput    []string `json:"expected_output"`
	StdOutput         []string `json:"std_output"`
	CompareResult     string   `json:"compare_result"`
}

func doPostWithAuth(url string, session, csrf string, payload map[string]interface{}, result interface{}) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	req, err := http.NewRequest("POST", url, strings.NewReader(string(body)))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Referer", url)
	req.Header.Set("User-Agent", userAgent)
	if csrf != "" {
		req.Header.Set("X-CSRFToken", csrf)
	}
	var cookies []string
	if session != "" {
		cookies = append(cookies, fmt.Sprintf("LEETCODE_SESSION=%s", session))
	}
	if csrf != "" {
		cookies = append(cookies, fmt.Sprintf("csrftoken=%s", csrf))
	}
	if len(cookies) > 0 {
		req.Header.Set("Cookie", strings.Join(cookies, "; "))
	}

	resp, err := Client.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode >= 400 {
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(data))
	}
	return json.Unmarshal(data, result)
}

func doGetWithAuth(url string, session, csrf string, result interface{}) error {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", userAgent)
	var cookies []string
	if session != "" {
		cookies = append(cookies, fmt.Sprintf("LEETCODE_SESSION=%s", session))
	}
	if csrf != "" {
		cookies = append(cookies, fmt.Sprintf("csrftoken=%s", csrf))
	}
	if len(cookies) > 0 {
		req.Header.Set("Cookie", strings.Join(cookies, "; "))
	}

	resp, err := Client.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode >= 400 {
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(data))
	}
	return json.Unmarshal(data, result)
}

type submitResponse struct {
	SubmissionID json.RawMessage `json:"submission_id"`
	Error        string          `json:"error"`
}

type interpretResponse struct {
	InterpretID  string          `json:"interpret_id"`
	SubmissionID json.RawMessage `json:"submission_id"`
	Error        string          `json:"error"`
}

func SubmitSolution(session, csrf, slug, questionID, langKey, code string) (string, error) {
	url := fmt.Sprintf("https://leetcode.com/problems/%s/submit/", slug)
	payload := map[string]interface{}{
		"lang":        langKey,
		"question_id": questionID,
		"typed_code":  code,
	}
	var resp submitResponse
	if err := doPostWithAuth(url, session, csrf, payload, &resp); err != nil {
		return "", err
	}
	if resp.Error != "" {
		return "", fmt.Errorf("submit error: %s", resp.Error)
	}
	id := strings.Trim(string(resp.SubmissionID), `"`)
	if id == "" || id == "null" {
		return "", fmt.Errorf("invalid submission response: %s", string(resp.SubmissionID))
	}
	return id, nil
}

func InterpretSolution(session, csrf, slug, questionID, langKey, code, dataInput string) (string, error) {
	url := fmt.Sprintf("https://leetcode.com/problems/%s/interpret_solution/", slug)
	payload := map[string]interface{}{
		"lang":        langKey,
		"question_id": questionID,
		"typed_code":  code,
		"data_input":  dataInput,
	}
	var resp interpretResponse
	if err := doPostWithAuth(url, session, csrf, payload, &resp); err != nil {
		return "", err
	}
	if resp.Error != "" {
		return "", fmt.Errorf("test run error: %s", resp.Error)
	}
	if resp.InterpretID != "" {
		return resp.InterpretID, nil
	}
	id := strings.Trim(string(resp.SubmissionID), `"`)
	if id != "" && id != "null" {
		return id, nil
	}
	return "", fmt.Errorf("no interpret_id returned")
}

func CheckSubmissionStatus(session, csrf, id string) (*SubmissionCheckResult, error) {
	url := fmt.Sprintf("https://leetcode.com/submissions/detail/%s/check/", id)
	var resp SubmissionCheckResult
	if err := doGetWithAuth(url, session, csrf, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

type EditorData struct {
	QuestionID         string `json:"questionId"`
	QuestionFrontendID string `json:"questionFrontendId"`
	Title              string `json:"title"`
	TitleSlug          string `json:"titleSlug"`
	ExampleTestcases   string `json:"exampleTestcases"`
	SampleTestCase     string `json:"sampleTestCase"`
}

type editorDataResp struct {
	Data struct {
		Question *EditorData `json:"question"`
	} `json:"data"`
}

func GetProblemTestcases(slug string) (string, error) {
	query := `query questionData($titleSlug: String!) {
		question(titleSlug: $titleSlug) {
			questionId, questionFrontendId, title, titleSlug, exampleTestcases, sampleTestCase
		}
	}`
	payload := map[string]interface{}{
		"query": query,
		"variables": map[string]string{"titleSlug": slug},
	}
	var resp editorDataResp
	if err := doPost(graphqlURL, payload, &resp); err != nil {
		return "", err
	}
	if resp.Data.Question == nil {
		return "", fmt.Errorf("problem not found")
	}
	q := resp.Data.Question
	if q.ExampleTestcases != "" {
		return q.ExampleTestcases, nil
	}
	return q.SampleTestCase, nil
}

