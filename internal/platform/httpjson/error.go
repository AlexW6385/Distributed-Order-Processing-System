package httpjson

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func Error(c *gin.Context, err error) {
	code := http.StatusInternalServerError
	message := err.Error()
	if st, ok := status.FromError(err); ok {
		message = st.Message()
		switch st.Code() {
		case codes.InvalidArgument, codes.FailedPrecondition, codes.AlreadyExists:
			code = http.StatusBadRequest
		case codes.NotFound:
			code = http.StatusNotFound
		case codes.Unavailable:
			code = http.StatusServiceUnavailable
		}
	}
	c.JSON(code, gin.H{"error": message})
}
