package main

import (
	"encoding/base64"
	"encoding/json"
	"os"
)

type Token struct {
	AccountTag   string `json:"a"`
	TunnelID     string `json:"t"`
	TunnelSecret string `json:"s"`
}

type Credentials struct {
	AccountTag   string `json:"AccountTag"`
	TunnelID     string `json:"TunnelID"`
	TunnelSecret string `json:"TunnelSecret"`
}

func parseB64EncodedToken(b64EncodedToken string) (Token, error) {
	token := Token{}
	tokenJson, err := base64.StdEncoding.DecodeString(b64EncodedToken)
	if err != nil {
		return token, err
	}
	err = json.Unmarshal([]byte(tokenJson), &token)
	if err != nil {
		return token, err
	}

	return token, nil
}

func writeCredentialsFile(path string, token Token) error {
	contents, err := json.Marshal((Credentials)(token))
	if err != nil {
		return nil
	}
	return os.WriteFile(path, contents, 0644)
}
