package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"

	admissionv1 "k8s.io/api/admission/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
)

var (
	universalDeserializer = serializer.NewCodecFactory(runtime.NewScheme()).UniversalDeserializer()
)

type AdmissionReviewHandler struct {
	logger *slog.Logger
}

func (h *AdmissionReviewHandler) handleRequest(r *http.Request) (*admissionv1.AdmissionReview, error) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading body: %w", err)
	}
	defer r.Body.Close()

	var admissionReviewReq admissionv1.AdmissionReview
	if _, _, err := universalDeserializer.Decode(body, nil, &admissionReviewReq); err != nil {
		return nil, fmt.Errorf("could not deserialize request: %w", err)
	}

	if admissionReviewReq.Request == nil {
		return nil, errors.New("malformed admission review: request is nil")
	}

	return &admissionReviewReq, nil
}

func (h *AdmissionReviewHandler) handlePod(admissionReviewReq *admissionv1.AdmissionReview) (*admissionv1.AdmissionResponse, error) {
	var pod corev1.Pod
	if err := json.Unmarshal(admissionReviewReq.Request.Object.Raw, &pod); err != nil {
		return nil, fmt.Errorf("could not unmarshal pod object: %w", err)
	}

	if value, exists := pod.Labels["required-label"]; !exists || value == "" {
		h.logger.Info("Rejecting pod: missing required label")
		return &admissionv1.AdmissionResponse{
			Allowed: false,
			Result: &metav1.Status{
				Message: "Pod is missing the required-label",
			},
		}, nil
	}

	h.logger.Info("Allowing request: required label present")
	return &admissionv1.AdmissionResponse{
		Allowed: true,
		Result: &metav1.Status{
			Message: "Validation OK",
		},
	}, nil
}

func (h *AdmissionReviewHandler) handleOtherResources() *admissionv1.AdmissionResponse {
	h.logger.Info("Resource is not a Pod, allowing by default")
	return &admissionv1.AdmissionResponse{
		Allowed: true,
		Result: &metav1.Status{
			Message: "Validation OK",
		},
	}
}

func (h *AdmissionReviewHandler) handleAdmissionReview(w http.ResponseWriter, r *http.Request) {
	admissionReviewReq, err := h.handleRequest(r)
	if err != nil {
		h.logger.Error("Error handling request", "error", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	var admissionResponse *admissionv1.AdmissionResponse
	switch admissionReviewReq.Request.Kind.Kind {
	case "Pod":
		admissionResponse, err = h.handlePod(admissionReviewReq)
		if err != nil {
			h.logger.Error("Error handling Pod", "error", err)
			admissionResponse = &admissionv1.AdmissionResponse{
				Allowed: false,
				Result: &metav1.Status{
					Message: fmt.Sprintf("Error handling Pod: %v", err),
				},
			}
		}
	default:
		admissionResponse = h.handleOtherResources()
	}

	admissionReviewResponse := admissionv1.AdmissionReview{
		TypeMeta: metav1.TypeMeta{
			Kind:       "AdmissionReview",
			APIVersion: "admission.k8s.io/v1",
		},
		Response: admissionResponse,
	}

	admissionReviewResponse.Response.UID = admissionReviewReq.Request.UID

	respBytes, err := json.Marshal(admissionReviewResponse)
	if err != nil {
		h.logger.Error("Error marshalling response", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if _, err := w.Write(respBytes); err != nil {
		h.logger.Error("Error writing response", "error", err)
	}
}
func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	slog.SetDefault(logger)

	h := &AdmissionReviewHandler{logger: logger}

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		logger.Info("Received request on root endpoint")
		fmt.Fprintf(w, "Welcome to the Validating Webhook Server!")
	})
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})
	mux.HandleFunc("/readyz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})
	mux.HandleFunc("/validate", h.handleAdmissionReview)

	server := &http.Server{
		Addr:    ":8443",
		Handler: mux,
	}

	logger.Info("Starting HTTPS server", "port", "8443")
	if err := server.ListenAndServeTLS("/etc/webhook/tls/tls.crt", "/etc/webhook/tls/tls.key"); err != nil {
		logger.Error("Failed to start server", "error", err)
		os.Exit(1)
	}
}
