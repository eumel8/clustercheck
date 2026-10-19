package promcheck

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

// QueryPrometheus runs an instant query against the Prometheus API and returns
// the value of the first result as string ("0" if the result is empty).
// TLS certificate verification is skipped.
func QueryPrometheus(prometheus, query, username, password string, debug bool) (string, error) {
	value := "0"
	params := url.Values{}
	params.Add("query", query)
	u := fmt.Sprintf("%s/api/v1/query?%s", prometheus, params.Encode())

	if debug {
		fmt.Printf("\n[DEBUG] Prometheus API Request:\n")
		fmt.Printf("  URL: %s\n", u)
		fmt.Printf("  Query: %s\n", query)
		if username != "" {
			fmt.Printf("  Auth: Basic (username: %s)\n", username)
		}
	}

	req, err := http.NewRequest("GET", u, nil)
	if err != nil {
		return value, err
	}
	req.SetBasicAuth(username, password)

	client := &http.Client{
		Timeout: 10 * time.Second,
		Transport: &http.Transport{
			Proxy:           http.ProxyFromEnvironment,
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}
	resp, err := client.Do(req)
	if err != nil {
		if debug {
			fmt.Printf("[DEBUG] Prometheus API Error: %v\n\n", err)
		}
		return value, err
	}
	defer resp.Body.Close()

	if debug {
		fmt.Printf("  Status Code: %d\n", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return value, err
	}
	if debug {
		fmt.Printf("[DEBUG] Prometheus API Response:\n  Body: %s\n\n", string(body))
	}

	var result struct {
		Data struct {
			Result []struct {
				Value [2]interface{} `json:"value"`
			} `json:"result"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return value, err
	}
	if len(result.Data.Result) > 0 {
		s, ok := result.Data.Result[0].Value[1].(string)
		if !ok {
			return value, fmt.Errorf("unexpected value type %T in Prometheus response", result.Data.Result[0].Value[1])
		}
		value = s
	}
	return value, nil
}
