package main

import (
	"errors"
	"fmt"
)

var ErrHelp = errors.New("flag: help requested")

var HelpMessage = `Cobra Lambda
Usage of cobra-lambda:
	With arguements:

	cl
	cobra-lambda --name [function name] -arg1 123 -arg2 foo --arg3

	Without arguments:
	cl
	cobra-lambda --name [function name]

Arguments after --name will be forwarded to remote cli named [function name]
`

func parseFuncName(args []string) (string, bool, error) {
	if len(args) == 0 {
		return "", false, nil
	}
	s := args[0]
	if len(s) < 2 || s[0] != '-' {
		return "", false, nil
	}

	numMinuses := 1
	if s[1] == '-' {
		numMinuses++
		if len(s) == 2 { // "--" terminates the flags
			args = args[1:]
			return "", false, nil
		}
	}

	name := s[numMinuses:]
	if len(name) == 0 || name[0] == '-' || name[0] == '=' {
		return "", false, fmt.Errorf("bad flag syntax: %s", s)
	}

	// it's a flag. does it have an argument?
	args = args[1:]
	hasValue := false
	value := ""
	for i := 1; i < len(name); i++ { // equals cannot be first
		if name[i] == '=' {
			value = name[i+1:]
			hasValue = true
			name = name[0:i]
			break
		}
	}

	if name == "help" || name == "h" { // special case for nice help message.
		return "", false, ErrHelp
	}

	// It must have a value, which might be the next argument.
	if !hasValue && len(args) > 0 {
		// value is the next arg
		hasValue = true
		value = args[0]
	}

	if name != "name" {
		return "", false, fmt.Errorf("flag provided but not valid: -%s", name)
	}

	if !hasValue {
		return "", false, fmt.Errorf("flag needs an argument: -%s", name)
	}

	return value, true, nil
}
