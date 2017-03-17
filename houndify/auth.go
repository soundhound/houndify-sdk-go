package houndify

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"time"
)

func generateAuthValues(clientID, clientKey, userID, requestID string) (
	houndClientAuth, houndRequestAuth string, timeStamp int64, returnErr error) {

	timeStamp = time.Now().Unix()

	//base64 decode key
	decodedClientKey, err := base64.StdEncoding.DecodeString(unescapeBase64Url(clientKey))
	if err != nil {
		fmt.Println(err)
		returnErr = errors.New("failed to decode client key")
		return
	}
	//sign
	hmac := hmac.New(sha256.New, decodedClientKey)
	hmac.Write([]byte(userID + ";" + requestID + fmt.Sprintf("%d", timeStamp)))
	signature := escapeBase64Url(base64.StdEncoding.EncodeToString([]byte(hmac.Sum(nil))))

	houndClientAuth = fmt.Sprintf("%s;%d;%s", clientID, timeStamp, signature)
	houndRequestAuth = userID + ";" + requestID
	returnErr = nil
	return
}

func unescapeBase64Url(input string) string {
	return strings.Replace(strings.Replace(input, "-", "+", -1), "_", "/", -1)
}

func escapeBase64Url(input string) string {
	return strings.Replace(strings.Replace(input, "+", "-", -1), "/", "_", -1)
}
