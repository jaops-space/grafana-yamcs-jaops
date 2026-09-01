package http

import (
	"context"
	"fmt"

	"github.com/jaops-space/grafana-yamcs-jaops/api/yamcs/api"
	"github.com/jaops-space/grafana-yamcs-jaops/pkg/utils/exception"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

// ProtoRequest is a helper function for sending requests with a given HTTP method,
// marshaling the body, and unmarshaling the response to the provided proto.Message.
// query holds per-request query parameters (e.g. time ranges, filters,
// pagination tokens) that are merged with httpManager's persistent Query at
// request-build time; pass nil if the call needs no query parameters.
func (httpManager *HTTPManager) ProtoRequest(ctx context.Context, method, path string, query map[string]string, body proto.Message, unmarshalTo proto.Message) error {
	// Construct the URL by combining the base API root with the provided path
	url := fmt.Sprintf("%s%s", httpManager.APIRoot, path)

	// Marshal the request body based on the format (Protobuf or JSON)
	marshalledBody, err := marshalMessage(body, httpManager.UsingProtobuf)
	if err != nil {
		return err
	}

	// Send the request and capture the response
	response, err := httpManager.SendRequestWithQuery(ctx, method, url, marshalledBody, query)
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
	// Skip unmarshalling if unmarshalTo is nil (for operations that don't return data)
	if unmarshalTo != nil {
		return unmarshalResponse(response, unmarshalTo, httpManager.UsingProtobuf)
	}
	return nil
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

	err := httpManager.Credentials.Login(context.Background(), httpManager)
	if err != nil {
		return err
	}
	httpManager.StartAutoRefresh()
	return nil
}

// GetProto sends a GET request with the given path and unmarshals the response into the provided proto.Message.
func (httpManager *HTTPManager) GetProto(ctx context.Context, path string, unmarshalTo proto.Message) error {
	return httpManager.ProtoRequest(ctx, "GET", path, nil, nil, unmarshalTo)
}

// GetProtoWithQuery is identical to GetProto but additionally merges the
// given per-request query parameters with the manager's persistent Query.
func (httpManager *HTTPManager) GetProtoWithQuery(ctx context.Context, path string, query map[string]string, unmarshalTo proto.Message) error {
	return httpManager.ProtoRequest(ctx, "GET", path, query, nil, unmarshalTo)
}

// PutProto sends a PUT request with the given path, body, and unmarshals the response into the provided proto.Message.
func (httpManager *HTTPManager) PutProto(ctx context.Context, path string, body proto.Message, unmarshalTo proto.Message) error {
	return httpManager.ProtoRequest(ctx, "PUT", path, nil, body, unmarshalTo)
}

// PutProtoWithQuery is identical to PutProto but additionally merges the
// given per-request query parameters with the manager's persistent Query.
func (httpManager *HTTPManager) PutProtoWithQuery(ctx context.Context, path string, query map[string]string, body proto.Message, unmarshalTo proto.Message) error {
	return httpManager.ProtoRequest(ctx, "PUT", path, query, body, unmarshalTo)
}

// PostProto sends a POST request with the given path, body, and unmarshals the response into the provided proto.Message.
func (httpManager *HTTPManager) PostProto(ctx context.Context, path string, body proto.Message, unmarshalTo proto.Message) error {
	return httpManager.ProtoRequest(ctx, "POST", path, nil, body, unmarshalTo)
}

// PostProtoWithQuery is identical to PostProto but additionally merges the
// given per-request query parameters with the manager's persistent Query.
func (httpManager *HTTPManager) PostProtoWithQuery(ctx context.Context, path string, query map[string]string, body proto.Message, unmarshalTo proto.Message) error {
	return httpManager.ProtoRequest(ctx, "POST", path, query, body, unmarshalTo)
}

// PatchProto sends a PATCH request with the given path, body, and unmarshals the response into the provided proto.Message.
func (httpManager *HTTPManager) PatchProto(ctx context.Context, path string, body proto.Message, unmarshalTo proto.Message) error {
	return httpManager.ProtoRequest(ctx, "PATCH", path, nil, body, unmarshalTo)
}

// PatchProtoWithQuery is identical to PatchProto but additionally merges the
// given per-request query parameters with the manager's persistent Query.
func (httpManager *HTTPManager) PatchProtoWithQuery(ctx context.Context, path string, query map[string]string, body proto.Message, unmarshalTo proto.Message) error {
	return httpManager.ProtoRequest(ctx, "PATCH", path, query, body, unmarshalTo)
}

// DeleteProto sends a DELETE request with the given path, body, and unmarshals the response into the provided proto.Message.
func (httpManager *HTTPManager) DeleteProto(ctx context.Context, path string, body proto.Message, unmarshalTo proto.Message) error {
	return httpManager.ProtoRequest(ctx, "DELETE", path, nil, body, unmarshalTo)
}

// DeleteProtoWithQuery is identical to DeleteProto but additionally merges the
// given per-request query parameters with the manager's persistent Query.
func (httpManager *HTTPManager) DeleteProtoWithQuery(ctx context.Context, path string, query map[string]string, body proto.Message, unmarshalTo proto.Message) error {
	return httpManager.ProtoRequest(ctx, "DELETE", path, query, body, unmarshalTo)
}
