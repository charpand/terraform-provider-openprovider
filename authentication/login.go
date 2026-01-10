// Package authentication provides functionality for user authentication with the OpenProvider API.
package authentication

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
)

// HTTPClient defines the interface for making HTTP requests.
type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

// LoginRequest represents a request to authenticate a user.
type LoginRequest struct {
	IPAddress string `json:"ip"`
	Username  string `json:"username"`
	Password  string `json:"password"`
}

// LoginResponse represents a response from the authentication endpoint.
type LoginResponse struct {
	Code int `json:"code"`
	Data struct {
		Token      string `json:"token"`
		ResellerID int    `json:"reseller_id"`
	} `json:"data"`
}

// Login authenticates a user and returns a token.
func Login(c HTTPClient, baseURL, ipAddress, username, password string) (*string, error) {
	request := LoginRequest{
		IPAddress: ipAddress,
		Username:  username,
		Password:  password,
	}
	body, err := json.Marshal(request)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", fmt.Sprintf("%s/v1beta/auth/login", baseURL), bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	resp, err := c.Do(req)
	if err != nil {
		return nil, err
	}

	defer func() {
		err := resp.Body.Close()
		if err != nil {
			log.Fatal(err)
		}
	}()

	var results LoginResponse
	if err := json.NewDecoder(resp.Body).Decode(&results); err != nil {
		return nil, err
	}
	return &results.Data.Token, nil
}
