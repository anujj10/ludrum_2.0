package auth

import (
	"encoding/json"

	fyersgosdk "github.com/FyersDev/fyers-go-sdk"
)

type FyersClient struct {
	Client 		*fyersgosdk.Client
	FyersModel	*fyersgosdk.FyersModel
}

func CreateClient(appID, secretID, redirectURL string) *fyersgosdk.Client{
	client:= fyersgosdk.SetClientData(appID, secretID, redirectURL)
	return client
}

func LoginURL(client *fyersgosdk.Client) string {
	url:= client.GetLoginURL()
	return url
}

func GenerateAccessToken(authcode string, client *fyersgosdk.Client) (map[string]interface{}, error) {
	token, err:= client.GenerateAccessToken(authcode, client)
	if err != nil {
		return nil, err
	}

	var tokenParse map[string]interface{}

	json.Unmarshal([]byte(token), &tokenParse)

	return tokenParse, nil
}

