package main

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/JayJamieson/cobra-lambda/wrapper"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "cl",
	Short: "cobra-lambda is a utility to invoke Cobra CLI on Lambda",
	Long:  `A utility CLI to invoke Cobra CLI applications running on AWS Lambda`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Hello from cobra command handler running in AWS Lambda!")
		fmt.Printf("Arguments passed to me: %v\n", args)
	},
}

func Handler(ctx context.Context, event json.RawMessage) (any, error) {
	args, err := wrapper.UnmarshalEvent(event)

	if err != nil {
		return nil, err
	}

	w := wrapper.NewCobraLambdaCLI(ctx, rootCmd)
	result, err := w.Execute(args.Args)

	// TODO: implement err != nil checks before deserializing
	return map[string]any{
		"stdout": result.Stdout,
		"error":  err.Error(),
	}, nil
}

func main() {
	lambda.Start(Handler)
}
