package main

import (
	"context"
	"encoding/json"
	"fmt"

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

func Handle(ctx context.Context, event json.RawMessage) (any, error) {

	// return map[string]any{
	// 	"result": bufCobra.String(),
	// 	"stdout": out,
	// }, nil
	return nil, nil
}

func main() {
	lambda.Start(Handle)
}
