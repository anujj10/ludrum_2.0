package fyers

import (
	"context"
	"fmt"

	"ludrum/internal/models"

	fyersgosdk "github.com/FyersDev/fyers-go-sdk"
	fyersws "github.com/FyersDev/fyers-go-sdk/websocket"
)

type MarketDataClient interface {
	FetchOptionChain(ctx context.Context, symbol, timestamp string, strikeCount int) (*models.OptionChainResponse, error)
	BuildAccessToken(appID, accessToken string) string
	NewDataSocket(accessToken string, subscription MarketSubscription, handlers SocketHandlers) *fyersws.FyersDataSocket
}

type SocketHandlers struct {
	OnConnect func()
	OnClose   func(message fyersws.DataClose)
	OnError   func(message fyersws.DataError)
	OnMessage func(message fyersws.DataResponse)
}

type APIClient struct {
	model *fyersgosdk.FyersModel
}

func NewAPIClient(appID, accessToken string) *APIClient {
	return &APIClient{
		model: fyersgosdk.NewFyersModel(appID, accessToken),
	}
}

func (c *APIClient) FetchOptionChain(ctx context.Context, symbol, timestamp string, strikeCount int) (*models.OptionChainResponse, error) {
	type result struct {
		raw string
		err error
	}

	ch := make(chan result, 1)
	go func() {
		raw, err := c.model.GetOptionChain(fyersgosdk.OptionChainRequest{
			Symbol:      symbol,
			Timestamp:   timestamp,
			StrikeCount: strikeCount,
		})
		ch <- result{raw: raw, err: err}
	}()

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case out := <-ch:
		if out.err != nil {
			return nil, out.err
		}

		payload, err := decodeOptionChain(out.raw)
		if err != nil {
			return nil, fmt.Errorf("decode fyers option chain: %w", err)
		}
		return payload, nil
	}
}

func (c *APIClient) BuildAccessToken(appID, accessToken string) string {
	return appID + ":" + accessToken
}

func (c *APIClient) NewDataSocket(accessToken string, subscription MarketSubscription, handlers SocketHandlers) *fyersws.FyersDataSocket {
	var socket *fyersws.FyersDataSocket
	onConnect := func() {
		if handlers.OnConnect != nil {
			handlers.OnConnect()
		}
		socket.Subscribe(subscription.Symbols, subscription.DataType)
	}

	socket = fyersws.NewFyersDataSocket(
		accessToken,
		"",
		false,
		true,
		true,
		50,
		onConnect,
		handlers.OnClose,
		handlers.OnError,
		handlers.OnMessage,
	)

	return socket
}

func decodeOptionChain(raw string) (*models.OptionChainResponse, error) {
	payload := &models.OptionChainResponse{}
	if err := jsonUnmarshal(raw, payload); err != nil {
		return nil, err
	}
	return payload, nil
}
