package api

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/PlakarKorp/plakar/cmd/plakar/utils"
)

type TokenResponse struct {
	Token string `json:"token"`
}

type LoginRequestGithub struct {
	Redirect string `json:"redirect"`
}

type LoginRequestEmail struct {
	Email    string `json:"email"`
	Redirect string `json:"redirect"`
}

func servicesLoginGithub(w http.ResponseWriter, r *http.Request) error {
	var req LoginRequestGithub

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return fmt.Errorf("failed to decode request body: %w", err)
	}

	parameters := make(map[string]string)
	parameters["redirect"] = req.Redirect

	lf, err := utils.NewLoginFlow(lrepository.AppContext(), lrepository.Configuration().RepositoryID)
	if err != nil {
		return fmt.Errorf("failed to create login flow: %w", err)
	}

	redirectURL, err := lf.RunUI("github", parameters)
	if err != nil {
		return fmt.Errorf("failed to run login flow: %w", err)
	}

	ret := struct {
		URL string `json:"URL"`
	}{
		URL: redirectURL,
	}

	return json.NewEncoder(w).Encode(ret)
}

func servicesLoginEmail(w http.ResponseWriter, r *http.Request) error {
	var req LoginRequestEmail

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return fmt.Errorf("failed to decode request body: %w", err)
	}

	parameters := make(map[string]string)
	parameters["email"] = req.Email
	parameters["redirect"] = req.Redirect

	lf, err := utils.NewLoginFlow(lrepository.AppContext(), lrepository.Configuration().RepositoryID)
	if err != nil {
		return fmt.Errorf("failed to create login flow: %w", err)
	}

	redirectURL, err := lf.RunUI("email", parameters)
	if err != nil {
		return fmt.Errorf("failed to run login flow: %w", err)
	}

	ret := struct {
		URL string `json:"URL"`
	}{
		URL: redirectURL,
	}
	return json.NewEncoder(w).Encode(ret)
}

func servicesLogout(w http.ResponseWriter, r *http.Request) error {
	configuration := lrepository.Configuration()
	if cache, err := lrepository.AppContext().GetCache().Repository(configuration.RepositoryID); err != nil {
		return err
	} else if exists := cache.HasAuthToken(); !exists {
		return nil
	} else {
		return cache.DeleteAuthToken()
	}
}
