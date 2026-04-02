package config

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	AppID        string
	SecretID     string
	RedirectURL  string
	AuthCode     string
	AccessToken  string
	RefreshToken string
}

func LoadConfig() *Config {
	err := godotenv.Load()
	if err != nil && !os.IsNotExist(err) {
		fmt.Println("error Fetching keys from config file: ", err)
	}

	return &Config{
		AppID: 			os.Getenv("APP_ID"),
		SecretID: 		os.Getenv("SECRET_ID"),
		RedirectURL: 	os.Getenv("REDIRECT_URL"),
		AuthCode: 		os.Getenv("AUTH_CODE"),
		AccessToken: 	os.Getenv("ACCESS_TOKEN"),
		RefreshToken: 	os.Getenv("REFRESH_TOKEN"),
	}
}
