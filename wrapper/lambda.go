package wrapper

import (
	"context"
	"encoding/json"
	"os"

	"github.com/spf13/cobra"
)

type CobraLambdaEvent struct {
	Args []string `json:"args"`
}

type CobraLambdaFunc func(ctx context.Context, event json.RawMessage) (any, error)

func NewCobrLambdaHandler(cmd *cobra.Command) CobraLambdaFunc {
	return func(ctx context.Context, eventJSON json.RawMessage) (any, error) {
		lambda := &CobraLambda{
			cmd:            cmd,
			ctx:            ctx,
			originalStdout: os.Stdout,
			originalStderr: os.Stderr,
		}

		event, err := UnmarshalEvent(eventJSON)

		if err != nil {
			return nil, err
		}

		return lambda.ExecuteContext(ctx, event.Args)
	}
}

func UnmarshalEvent(eventJSON json.RawMessage) (*CobraLambdaEvent, error) {
	event := &CobraLambdaEvent{}

	err := json.Unmarshal(eventJSON, event)

	if err != nil {
		return nil, err
	}

	return event, nil
}
