package http

import (
	"fmt"

	"github.com/jaops-space/grafana-yamcs-jaops/api/yamcs/api"
	"github.com/jaops-space/grafana-yamcs-jaops/pkg/utils/exception"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

// ProtoRequest is a helper function for sending requests with a given HTTP method,
// marshaling the body, and unmarshaling the response to the provided proto.Message.
func (httpManager *HTTPManager) ProtoRequest(method, path string, body proto.Message, unmarshalTo proto.Message) error {
	// Construct the URL by combining the base API root with the provided path
	url := fmt.Sprintf("%s%s", httpManager.APIRoot, path)

	// Marshal the request body based on the format (Protobuf or JSON)
	marshalledBody, err := marshalMessage(body, httpManager.UsingProtobuf)
	if err != nil {
		return err
	}

	// Send the request and capture the response
	response, err := httpManager.SendRequest(method, url, marshalledBody)
	if err != nil && response != nil {
		exc := &api.ExceptionMessage{}
		err := unmarshalResponse(response, exc, httpManager.UsingProtobuf)
		if err != nil {
			return exception.Wrap("Error unmarshalling error after HTTP error", "HTTP_API_ERROR", err)
		}
		return exception.Wrap(fmt.Sprintf("Error in %s call to \"%s\", type: %s, message: %s\n", method, path, exc.GetType(), exc.GetMsg()), "HTTP_API_ERROR", err)
	} else if err != nil {
		return err
	}

	// Unmarshal the response based on the format (Protobuf or JSON)
	return unmarshalResponse(response, unmarshalTo, httpManager.UsingProtobuf)
}

// marshalMessage marshals a given proto message into either Protobuf or JSON format.
func marshalMessage(body proto.Message, useProtobuf bool) ([]byte, error) {
	if useProtobuf {
		return proto.Marshal(body)
	}
	return protojson.Marshal(body)
}

// unmarshalResponse unmarshals the response into the provided proto.Message based on the format.
func unmarshalResponse(response []byte, unmarshalTo proto.Message, useProtobuf bool) error {
	if useProtobuf {
		return proto.Unmarshal(response, unmarshalTo)
	}
	return protojson.Unmarshal(response, unmarshalTo)
}

/*
Login authenticates using the provided account credentials and returns the authentication tokens.

Parameters:
- account: Account credentials containing the username and password.

Returns:
- AuthCredentials containing access tokens and refresh token.
- Error in case of failure.
*/
func (httpManager *HTTPManager) Login(account *Credentials) error {

	err := httpManager.Credentials.Login(httpManager)
	if err != nil {
		return err
	}
	httpManager.RefreshStop = make(chan struct{})
	httpManager.StartAutoRefresh()
	return nil
}

// GetProto sends a GET request with the given path and unmarshals the response into the provided proto.Message.
func (httpManager *HTTPManager) GetProto(path string, unmarshalTo proto.Message) error {
	return httpManager.ProtoRequest("GET", path, nil, unmarshalTo)
}

// PutProto sends a PUT request with the given path, body, and unmarshals the response into the provided proto.Message.
func (httpManager *HTTPManager) PutProto(path string, body proto.Message, unmarshalTo proto.Message) error {
	return httpManager.ProtoRequest("PUT", path, body, unmarshalTo)
}

// PostProto sends a POST request with the given path, body, and unmarshals the response into the provided proto.Message.
func (httpManager *HTTPManager) PostProto(path string, body proto.Message, unmarshalTo proto.Message) error {
	return httpManager.ProtoRequest("POST", path, body, unmarshalTo)
}

// DeleteProto sends a DELETE request with the given path, body, and unmarshals the response into the provided proto.Message.
func (httpManager *HTTPManager) DeleteProto(path string, body proto.Message, unmarshalTo proto.Message) error {
	return httpManager.ProtoRequest("DELETE", path, body, unmarshalTo)
}
