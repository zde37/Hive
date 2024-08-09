package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
	mocked "github.com/zde37/Hive/internal/mocks"
	"go.uber.org/mock/gomock"
)

func TestHealth(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockHandler := mocked.NewMockHandler(ctrl)

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/health", nil)

	mockHandler.EXPECT().Health(gomock.Any(), gomock.Any()).DoAndReturn(func(w http.ResponseWriter, r *http.Request) error {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, err := io.WriteString(w, "Hello world")
		return err
	})

	err := mockHandler.Health(w, r)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, w.Code)

	expectedContentType := "text/plain; charset=utf-8"
	require.Equal(t, expectedContentType, w.Header().Get("Content-Type"))
	expectedBody := "Hello world"
	require.Equal(t, expectedBody, w.Body.String())
}

func TestGetNodeInfo(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockHandler := mocked.NewMockHandler(ctrl)

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/info/testpeerid", nil)

	expectedNodeInfo := map[string]interface{}{
		"ID":           "testpeerid",
		"Addresses":    []interface{}{"addr1", "addr2"},
		"AgentVersion": "v1.0.0",
		"Protocols":    []interface{}{"proto1", "proto2"},
		"PublicKey":    "pubkey",
	}

	mockHandler.EXPECT().GetNodeInfo(gomock.Any(), gomock.Any()).DoAndReturn(func(w http.ResponseWriter, r *http.Request) error {
		w.Header().Set("Content-Type", "application/json")
		return json.NewEncoder(w).Encode(expectedNodeInfo)
	})

	err := mockHandler.GetNodeInfo(w, r)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, "application/json", w.Header().Get("Content-Type"))

	var responseNodeInfo map[string]interface{}
	err = json.NewDecoder(w.Body).Decode(&responseNodeInfo)
	require.NoError(t, err)
	require.Equal(t, expectedNodeInfo, responseNodeInfo)
}

func TestPingNode(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockHandler := mocked.NewMockHandler(ctrl)
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/ping/testpeerid", nil)

	expectedPingResponse := struct {
		Success bool   `json:"success"`
		Text    string `json:"text"`
		Time    string `json:"time"`
	}{
		Success: true,
		Text:    "Ping successful",
		Time:    "100ms",
	}

	mockHandler.EXPECT().PingNode(gomock.Any(), gomock.Any()).DoAndReturn(func(w http.ResponseWriter, r *http.Request) error {
		w.Header().Set("Content-Type", "application/json")
		return json.NewEncoder(w).Encode(expectedPingResponse)
	})

	err := mockHandler.PingNode(w, r)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, "application/json", w.Header().Get("Content-Type"))

	var responsePingInfo struct {
		Success bool   `json:"success"`
		Text    string `json:"text"`
		Time    string `json:"time"`
	}
	err = json.NewDecoder(w.Body).Decode(&responsePingInfo)
	require.NoError(t, err)
	require.Equal(t, expectedPingResponse, responsePingInfo)
}

func TestPingNodes(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockHandler := mocked.NewMockHandler(ctrl)
	tests := []struct {
		name           string
		peerID         string
		setupMock      func(mockHandler *mocked.MockHandler)
		expectedStatus int
		expectedResp   interface{}
	}{
		{
			name:   "Empty peerID",
			peerID: "",
			setupMock: func(mockHandler *mocked.MockHandler) {
				mockHandler.EXPECT().PingNode(gomock.Any(), gomock.Any()).Return(
					NewErrorStatus(fmt.Errorf("peerid is required"), http.StatusBadRequest, 0),
				)
			},
			expectedStatus: http.StatusBadRequest,
			expectedResp: ErrorResponse{
				Error: "peerid is required",
			},
		},
		{
			name:   "IPFS Ping Error",
			peerID: "testpeerid",
			setupMock: func(mockHandler *mocked.MockHandler) {
				mockHandler.EXPECT().PingNode(gomock.Any(), gomock.Any()).Return(
					NewErrorStatus(errors.New("an error occurred"), http.StatusInternalServerError, 1),
				)
			},
			expectedStatus: http.StatusInternalServerError,
			expectedResp: ErrorResponse{
				Error: "an error occurred",
			},
		},
		{
			name:   "Successful Ping",
			peerID: "testpeerid",
			setupMock: func(mockHandler *mocked.MockHandler) {
				mockHandler.EXPECT().PingNode(gomock.Any(), gomock.Any()).DoAndReturn(func(w http.ResponseWriter, r *http.Request) error {
					response := struct {
						Success bool   `json:"success"`
						Text    string `json:"text"`
						Time    string `json:"time"`
					}{
						Success: true,
						Text:    "Ping successful",
						Time:    "50ms",
					}
					return json.NewEncoder(w).Encode(response)
				})
			},
			expectedStatus: http.StatusOK,
			expectedResp: struct {
				Success bool   `json:"success"`
				Text    string `json:"text"`
				Time    string `json:"time"`
			}{
				Success: true,
				Text:    "Ping successful",
				Time:    "50ms",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMock(mockHandler)

			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, "/ping/"+tt.peerID, nil)
			// r = r.WithContext(context.WithValue(r.Context(), "peerid", tt.peerID))

			err := mockHandler.PingNode(w, r)
			if tt.expectedStatus != http.StatusOK {
				require.Error(t, err)
				errorStatus, ok := err.(ErrorStatus)
				require.True(t, ok)
				require.Equal(t, tt.expectedStatus, errorStatus.statusCode)

				errorResp, ok := tt.expectedResp.(ErrorResponse)
				require.True(t, ok)
				require.Equal(t, errorResp.Error, errorStatus.error.Error())
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.expectedStatus, w.Code)

				var response struct {
					Success bool   `json:"success"`
					Text    string `json:"text"`
					Time    string `json:"time"`
				}
				err = json.NewDecoder(w.Body).Decode(&response)
				require.NoError(t, err)

				expectedResp, ok := tt.expectedResp.(struct {
					Success bool   `json:"success"`
					Text    string `json:"text"`
					Time    string `json:"time"`
				})
				require.True(t, ok)
				require.Equal(t, expectedResp, response)
			}
		})
	}
}
