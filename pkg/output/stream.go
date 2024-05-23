// Copyright 2019 The Kanister Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package output

import (
	"bufio"
	"bytes"
	"context"
	"io"

	"github.com/kanisterio/kanister/pkg/field"
	"github.com/kanisterio/kanister/pkg/log"
)

type scanState struct {
	outputBuf       []byte
	readingOutput   bool
	separatorSuffix []byte
}

const bufferSize64k = 64 * 1024

func splitLines(ctx context.Context, r io.ReadCloser, f func(context.Context, []byte) error) error {
	// Call r.Close() if the context is canceled or if s.Scan() == false.
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	go func() {
		<-ctx.Done()
		_ = r.Close()
	}()

	state := InitState()

	reader := bufio.NewReaderSize(r, bufferSize64k)

	// Run a simple state machine loop
	for {
		line, isPrefix, err := reader.ReadLine()
		if err == io.EOF {
			// Terminal state
			return nil
		}
		if err != nil {
			return err
		}
		// Skip empty lines
		if len(line) == 0 {
			continue
		}
		switch {
		case state.readingOutput:
			if state, err = handleOutput(state, line, isPrefix, ctx, f); err != nil {
				return err
			}
		case len(state.separatorSuffix) > 0:
			if state, err = handleSeparatorSuffix(state, line, isPrefix, ctx, f); err != nil {
				return err
			}
		default:
			if state, err = handleLog(line, isPrefix, ctx, f); err != nil {
				return err
			}
		}
	}
}

func InitState() scanState {
	return scanState{
		outputBuf:       []byte(nil),
		readingOutput:   false,
		separatorSuffix: []byte(nil),
	}
}

func ReadPhaseOutputState(outputBuf []byte) scanState {
	return scanState{
		outputBuf:       outputBuf,
		readingOutput:   true,
		separatorSuffix: []byte(nil),
	}
}

func CheckSeparatorSuffixState(separatorSuffix []byte) scanState {
	return scanState{
		outputBuf:       []byte(nil),
		readingOutput:   false,
		separatorSuffix: separatorSuffix,
	}
}

func handleOutput(
	state scanState,
	line []byte,
	isPrefix bool,
	ctx context.Context,
	f func(context.Context, []byte) error,
) (scanState, error) {
	if isPrefix {
		// Accumulate phase output
		return ReadPhaseOutputState(append(state.outputBuf, line...)), nil
	}
	// Reached the end of the line while reading phase output
	if err := f(ctx, append(state.outputBuf, line...)); err != nil {
		return state, err
	}
	// Transition out of readingOutput state
	return InitState(), nil
}

func handleSeparatorSuffix(
	state scanState,
	line []byte,
	isPrefix bool,
	ctx context.Context,
	f func(context.Context, []byte) error,
) (scanState, error) {
	if bytes.Index(line, state.separatorSuffix) == 0 {
		return captureOutputContent(line, isPrefix, len(state.separatorSuffix), ctx, f)
	}
	// Read log like normal
	return handleLog(line, isPrefix, ctx, f)
}

func handleLog(
	line []byte,
	isPrefix bool,
	ctx context.Context,
	f func(context.Context, []byte) error,
) (scanState, error) {
	indexOfPOString := bytes.Index(line, []byte(PhaseOpString))
	if indexOfPOString == -1 {
		// Log plain output, there is no phase output here
		logOutput(ctx, line)

		// There is a corner case possible when PhaseOpString is split between chunks
		splitSeparator, separatorSuffix := checkSplitSeparator(line)
		if splitSeparator != -1 {
			// Transition to separatorSuffix state to check next line
			return CheckSeparatorSuffixState(separatorSuffix), nil
		}

		return InitState(), nil
	}
	// Log everything before separator as plain output
	prefix := line[0:indexOfPOString]
	if len(prefix) > 0 {
		logOutput(ctx, prefix)
	}

	return captureOutputContent(line, isPrefix, indexOfPOString+len(PhaseOpString), ctx, f)
}

func captureOutputContent(
	line []byte,
	isPrefix bool,
	startIndex int,
	ctx context.Context,
	f func(context.Context, []byte) error,
) (scanState, error) {
	outputContent := line[startIndex:]
	if !isPrefix {
		if err := f(ctx, outputContent); err != nil {
			return InitState(), err
		}
		return InitState(), nil
	}
	return ReadPhaseOutputState(append([]byte(nil), outputContent...)), nil
}

func checkSplitSeparator(line []byte) (splitSeparator int, separatorSuffix []byte) {
	lineLength := len(line)
	phaseOpBytes := []byte(PhaseOpString)
	for i := 1; i < len(phaseOpBytes) && i <= lineLength; i++ {
		if bytes.Equal(line[lineLength-i:], phaseOpBytes[0:i]) {
			return lineLength - i, phaseOpBytes[i:]
		}
	}
	return -1, nil
}

func logOutput(ctx context.Context, out []byte) {
	log.WithContext(ctx).Print("", field.M{"Pod_Out": string(out)})
}

// State machine
// init state: ignore and log output, until we reach ###Phase-output###
// Read output state: accumulate output in buffer
// Transitions:
// init state -> output state: on reaching ###Phase-output###: create and start accumulating output buffer
// output state -> output state: DONT DO THAT YET
// output state -> init state : on reaching \n: parse output json from output buffer and clean the buffer

func LogAndParse(ctx context.Context, r io.ReadCloser) (map[string]interface{}, error) {
	out := make(map[string]interface{})
	err := splitLines(ctx, r, func(ctx context.Context, outputContent []byte) error {
		outputContent = bytes.TrimSpace(outputContent)
		if len(outputContent) != 0 {
			log.WithContext(ctx).Print("", field.M{"Pod_Out": string(outputContent)})
			op, err := UnmarshalOutput(outputContent)
			if err != nil {
				return err
			}
			if op != nil {
				out[op.Key] = op.Value
			}
		}
		return nil
	})
	return out, err
}
