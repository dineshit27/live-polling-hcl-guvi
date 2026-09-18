package handlers

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

type ErrorResponse struct {
	Error   string            `json:"error"`
	Message string            `json:"message,omitempty"`
	Details map[string]string `json:"details,omitempty"`
}

type SuccessResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
	Data    any    `json:"data,omitempty"`
}

func RespondError(c *gin.Context, statusCode int, errTitle string, message string) {
	c.AbortWithStatusJSON(statusCode, ErrorResponse{
		Error:   errTitle,
		Message: message,
	})
}

// RespondValidationError formats validation failures into field-level error descriptions.
// Note: This function is strictly a presentation-layer response formatter. It does NOT automatically
// validate input. Each HTTP handler and service must explicitly bind and validate input parameters
// (e.g. via c.ShouldBindJSON or domain validators) before invoking repository or database operations.
func RespondValidationError(c *gin.Context, err error) {
	details := make(map[string]string)

	if validationErrors, ok := err.(validator.ValidationErrors); ok {
		for _, fieldErr := range validationErrors {
			fieldName := fieldErr.Field()
			tag := fieldErr.Tag()
			param := fieldErr.Param()

			switch tag {
			case "required":
				details[fieldName] = fmt.Sprintf("%s is required", fieldName)
			case "email":
				details[fieldName] = fmt.Sprintf("%s must be a valid email address", fieldName)
			case "min":
				details[fieldName] = fmt.Sprintf("%s must be at least %s characters or items", fieldName, param)
			case "max":
				details[fieldName] = fmt.Sprintf("%s must not exceed %s characters or items", fieldName, param)
			case "oneof":
				details[fieldName] = fmt.Sprintf("%s must be one of: %s", fieldName, param)
			default:
				details[fieldName] = fmt.Sprintf("%s failed validation rule: %s", fieldName, tag)
			}
		}
	} else {
		details["payload"] = "Malformed or invalid JSON payload"
	}

	c.AbortWithStatusJSON(http.StatusBadRequest, ErrorResponse{
		Error:   "Validation Failed",
		Message: "One or more input fields failed validation",
		Details: details,
	})
}

func RespondSuccess(c *gin.Context, statusCode int, data any, message string) {
	c.JSON(statusCode, SuccessResponse{
		Success: true,
		Message: message,
		Data:    data,
	})
}
