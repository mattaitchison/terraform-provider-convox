package helpers

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/convox/logger"
	"github.com/stvp/rollbar"
)

var regexpEmail = regexp.MustCompile(`^[a-z0-9._%+\-]+@[a-z0-9.\-]+\.[a-z]{2,4}$`)

func init() {
	rollbar.Token = os.Getenv("ROLLBAR_TOKEN")
	rollbar.Environment = os.Getenv("CLIENT_ID")
}

func Error(log *logger.Logger, err error) {
	if log != nil {
		log.ErrorBacktrace(err)
	}

	if rollbar.Token != "" {
		extraData := map[string]string{
			"AWS_REGION": os.Getenv("AWS_REGION"),
			"CLIENT_ID":  os.Getenv("CLIENT_ID"),
			"RACK":       os.Getenv("RACK"),
			"RELEASE":    os.Getenv("RELEASE"),
			"VPC":        os.Getenv("VPC"),
		}
		extraField := &rollbar.Field{"env", extraData}
		rollbar.Error(rollbar.ERR, err, extraField)
	}
}

func TrackEvent(event string, params map[string]interface{}) {
}

// Convenience function to track success in a controller handler
// See also httperr.TrackErrorf and httperr.TrackServer
func TrackSuccess(event string, params map[string]interface{}) {
	params["state"] = "success"

	TrackEvent(event, params)
}

func TrackError(event string, err error, params map[string]interface{}) {
	params["error"] = fmt.Sprintf("%v", err)
	params["state"] = "error"

	TrackEvent(event, params)
}

func RackId() string {
	if stackId := os.Getenv("STACK_ID"); stackId != "" {
		parts := strings.Split(stackId, "/")
		return parts[len(parts)-1]
	}

	return os.Getenv("CLIENT_ID")
}
