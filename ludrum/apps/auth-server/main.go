package main

import (
	"log"
	config "ludrum/configs"
	"ludrum/internal/ingestion/auth"
)

func main() {
	cfg:= config.LoadConfig()
	//generate URL
	client := auth.CreateClient(cfg.AppID, cfg.SecretID, cfg.RedirectURL)
	loginURL:= client.GetLoginURL()
	log.Println(loginURL)

	//generate token
	token, err:= auth.GenerateAccessToken(cfg.AuthCode, client)
	if err != nil {
		log.Println("Error generating token: ", err)
	}

	AccessToken:= token["access_token"]
	RefreshToken:= token["refresh_token"]

	log.Println("Access Token: ", AccessToken)
	log.Println("Refresh Token:	", RefreshToken)
}