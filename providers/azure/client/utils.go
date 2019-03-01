package client

import (
	"fmt"

	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/Azure/go-autorest/autorest/to"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"

	core "github.com/caicloud/loadbalancer-provider/core/provider"
)

// IsNotFound check the error is Resource Not Found
func IsNotFound(err error) bool {
	if e, ok := err.(autorest.DetailedError); ok {
		statusCode := e.StatusCode.(int)
		return 404 == statusCode
	}
	return false
}

const (
	// SecretTypeAzure azure secret type
	SecretTypeAzure corev1.SecretType = "azure"
)

func getAPISecret(provider corev1.SecretType, storeLister *core.StoreLister) (map[string][]byte, error) {
	secrets, err := storeLister.Secret.Secrets(v1.NamespaceSystem).List(labels.Everything())
	if err != nil {
		return nil, err
	}
	if len(secrets) == 0 {
		return nil, fmt.Errorf("azure: there is no specify type of secret:%v", provider)
	}
	for _, secret := range secrets {
		if secret.Type == provider {
			return secret.Data, nil
		}
	}
	return nil, fmt.Errorf("azure: get %s type secret failed", provider)
}

// ServiceError for azure service return error
type ServiceError struct {
	StatusCode int
	Code       string
	Message    string
	Target     string
}

func (e *ServiceError) Error() string {
	return fmt.Sprintf("statusCode %d Code %s Message %s", e.StatusCode, e.Code, e.Message)
}

// NewServiceError create a service error
func NewServiceError(code string, message string) *ServiceError {
	return &ServiceError{
		Code:    code,
		Message: message,
	}
}

// ParseServiceError parse azure error to ServiceError struct
// if the err is not service error, will return nil
func ParseServiceError(err error) *ServiceError {
	detailedError, ok := err.(autorest.DetailedError)
	if !ok {
		return nil
	}
	if detailedError.StatusCode == nil || detailedError.Original == nil {
		return nil
	}
	statusCode, ok := detailedError.StatusCode.(int)
	if !ok {
		return nil
	}
	var code, message string
	var target *string
	var serviceError *azure.ServiceError
	switch t := detailedError.Original.(type) {
	case *azure.ServiceError:
		serviceError = t
	case *azure.RequestError:
		if t.ServiceError == nil {
			return nil
		}
		serviceError = t.ServiceError

	default:
		return nil
	}
	code = serviceError.Code
	message = serviceError.Message
	target = serviceError.Target

	return &ServiceError{
		StatusCode: statusCode,
		Code:       code,
		Message:    message,
		Target:     to.String(target),
	}
}
