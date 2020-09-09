package google

import (
	"context"
	"github.com/CastyLab/grpc.server/config"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"io/ioutil"
	"log"
	"time"
)

var (
	err error
	jsonConfig []byte
	oauthClient *oauth2.Config
	scopes = []string{
		"profile",
		"email",
		"openid",
	}
)

func init() {
	jsonConfig, err = ioutil.ReadFile(config.Map.Secrets.Oauth.Google)
	if err != nil {
		log.Fatal(err)
	}
	oauthClient, err = google.ConfigFromJSON(jsonConfig, scopes...)
	if err != nil {
		log.Fatal(err)
	}
}

func Authenticate(code string) (*oauth2.Token, error) {
	mCtx, _ := context.WithTimeout(context.Background(), 10 * time.Second)
	return oauthClient.Exchange(mCtx, code, oauth2.AccessTypeOffline)
}