package main

import (
	"encoding/json"
	"fmt"
	"net/rpc"
	"os"
	"time"

	"github.com/JayJamieson/cobra-lambda/wrapper"
	"github.com/aws/aws-lambda-go/lambda/messages"
)

func main() {
	client, err := rpc.Dial("tcp", fmt.Sprintf("localhost:%d", 8001))

	if err != nil {
		fmt.Printf("%v\n", err)
		os.Exit(1)
	}

	argsEvent := wrapper.CobraLambdaEvent{Args: os.Args[1:]}
	payload, err := json.Marshal(argsEvent)

	if err != nil {
		fmt.Printf("%v\n", err)
		os.Exit(1)
	}

	args := messages.InvokeRequest{
		Payload: payload,
		Deadline: messages.InvokeRequest_Timestamp{
			Seconds: time.Now().Unix() + 10,
		},
	}

	invokeResponse := &messages.InvokeResponse{}
	if err := client.Call("Function.Invoke", args, &invokeResponse); err != nil {
		fmt.Printf("%v\n", err)
		os.Exit(1)
	}

	output := &wrapper.CobraLambdaOutput{}

	if err := json.Unmarshal(invokeResponse.Payload, &output); err != nil {
		fmt.Printf("%v\n", err)
		os.Exit(1)
	}

	fmt.Print(output.Stdout)
}
