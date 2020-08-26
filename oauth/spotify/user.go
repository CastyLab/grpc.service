package spotify

import (
	"encoding/json"
	"errors"
	"golang.org/x/oauth2"
	"io/ioutil"
	"log"
	"net/http"
)

type ExplicitContent struct {
	FilterEnabled  bool  `json:"filter_enabled"`
	FilterLocked   bool  `json:"filter_locked"`
}

type User struct {
	Id              string            `json:"id"`
	DisplayName     string            `json:"display_name"`
	Email           string            `json:"email"`
	Country         string            `json:"country"`
	Href            string            `json:"href"`
	Images          []string          `json:"images"`
	Product         string            `json:"product"`
	Type            string            `json:"type"`
	Uri             string            `json:"uri"`
	ExplicitContent []ExplicitContent `json:"explicit_content"`

	ExternalUrls struct{
		Spotify string `json:"spotify"`
	} `json:"external_urls"`

	Followers struct{
		Href   string  `json:"href"`
		Total  int     `json:"total"`
	}
}

func (u *User) GetUserId() string {
	return u.Id
}

func (u *User) GetAvatar() string {
	return ""
}

func (u *User) GetEmailAddress() string {
	return u.Email
}

func (u *User) GetFullname() string {
	return u.DisplayName
}

func GetUserByToken(token *oauth2.Token) (*User, error) {

	request, err := http.NewRequest(
		"GET",
		"https://api.spotify.com/v1/me",
		nil,
	)

	if request != nil {

		request.Header.Add("Authorization", "Bearer " + token.AccessToken)

		resp, err := http.DefaultClient.Do(request)
		if err != nil {
			log.Println(err)
		}

		defer resp.Body.Close()

		body, _ := ioutil.ReadAll(resp.Body)

		var user *User
		jsonErr := json.Unmarshal(body, &user)
		if jsonErr != nil {
			log.Println(jsonErr)
		}

		if resp.StatusCode == http.StatusOK {
			return user, nil
		}

		return user, errors.New(string(body))
	}
	return nil, err
}