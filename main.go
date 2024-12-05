package main

import (
	"context"
	"errors"
	"fmt"
	"os"

	lambda "github.com/JayJamieson/go-lambda-invoke"
)

func main() {
	ctx := context.Background()

	if len(os.Args[1:]) == 0 {
		fmt.Println(HelpMessage)
		os.Exit(2)
	}

	client, err := lambda.NewDefaultClient(ctx)

	if err != nil {
		fmt.Printf("%v", err)
		os.Exit(1)
	}

	funcName, ok, err := parseFuncName(os.Args[1:])

	if err != nil && errors.Is(err, ErrHelp) {
		fmt.Print(HelpMessage)
		os.Exit(0)
	}

	if !ok {
		fmt.Printf("%v\n", err)
		os.Exit(1)
	}

	var output = &ExecutionOutput{}

	err = lambda.InvokeSync(ctx, client, &lambda.InvokeInput{
		Name:      funcName,
		Qualifier: "$LATEST",
		Payload:   os.Args[3:],
	}, &output)

	if err != nil {
		fmt.Printf("%v\n", err)
		os.Exit(1)
	}

	fmt.Println(output.Stdout)
}
