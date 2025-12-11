package main

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/JayJamieson/cobra-lambda/cli/flag"
	"github.com/JayJamieson/cobra-lambda/wrapper"
	lambda "github.com/JayJamieson/go-lambda-invoke"
)

var HelpMessage = `Cobra Lambda
Usage of cobra-lambda:
	With arguements:

	clctl
	cobra-lambda --name [function name] -arg1 123 -arg2 foo --arg3

	Without arguments:
	clctl
	cobra-lambda --name [function name]

Arguments after --name will be forwarded to remote cli named [function name]
`

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

	funcName, ok, err := flag.ParseFuncName(os.Args[1:])

	if err != nil && errors.Is(err, flag.ErrHelp) {
		fmt.Print(HelpMessage)
		os.Exit(0)
	}

	if !ok {
		fmt.Printf("%v\n", err)
		os.Exit(1)
	}

	output := &wrapper.CobraLambdaOutput{}

	err = lambda.InvokeSync(ctx, client, &lambda.InvokeInput{
		Name:      funcName,
		Qualifier: "$LATEST",
		Payload:   wrapper.CobraLambdaEvent{Args: os.Args[3:]},
	}, &output)

	if err != nil {
		fmt.Printf("%v\n", err)
		os.Exit(1)
	}

	fmt.Print(output.Stdout)
}
