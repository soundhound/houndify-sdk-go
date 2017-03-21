package houndify

type requestInfo map[string]interface{}

func createRequestInfo(clientID, requestID string, timeStamp int64, extraFields map[string]interface{}) (requestInfo, error) {
	reqInfo := make(requestInfo)

	if len(extraFields) > 0 {
		for key, val := range extraFields {
			if val != nil {
				reqInfo[key] = val
			}
		}
	}
	reqInfo["TimeStamp"] = timeStamp
	reqInfo["ClientID"] = clientID
	reqInfo["RequestID"] = requestID
	reqInfo["SDK"] = "Go"
	reqInfo["SDKVersion"] = "0.1"
	reqInfo["PartialTranscriptsDesired"] = true
	reqInfo["ObjectByteCountPrefix"] = true
	return reqInfo, nil
}
