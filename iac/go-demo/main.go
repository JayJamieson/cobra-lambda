package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/spf13/cobra"
)

var Stdout = os.Stdout // keep backup of the real stdout
func main() {
	lambda.Start(Handle)
}

func Handle(ctx context.Context, event json.RawMessage) (any, error) {
	defer func() {
		// reset stdout back to normal
		os.Stdout = Stdout
	}()
	var rootCmd = &cobra.Command{
		Use:   "hugo",
		Short: "Hugo is a very fast static site generator",
		Long: `A Fast and Flexible Static Site Generator built with
					 love by spf13 and friends in Go.
					 Complete documentation is available at http://hugo.spf13.com`,
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("Hello from cobra command handler!")
			fmt.Printf("Arguments passed to me%v\n", args)
		},
	}

	// hijack to stdout to send in response
	r, w, err := os.Pipe()

	if err != nil {
		return nil, err
	}

	os.Stdout = w

	outC := make(chan string)

	go func() {
		// copy the output in a separate goroutine to prevent blocking
		buf := &bytes.Buffer{}
		io.Copy(buf, r)
		outC <- buf.String()
	}()

	data := make([]string, 0, 10)
	err = json.Unmarshal(event, &data)

	if err != nil {
		return nil, err
	}

	bufCobra := &bytes.Buffer{}

	rootCmd.SetOut(bufCobra)
	rootCmd.SetArgs(data)

	fmt.Println("Before executing the handler")

	rootCmd.SetContext(ctx)
	err = rootCmd.Execute()

	if err != nil {
		return nil, err
	}

	fmt.Println("After executing the handler")
	// close Pipe as no longer needed
	w.Close()
	out := <-outC

	return map[string]any{
		"result": bufCobra.String(),
		"stdout": out,
	}, nil
}
