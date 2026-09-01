package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"math"
	"os/exec"
)

// Args: video filepath as string
// Returns: aspect ratio of the video at the filepath as a string, error
// If error, return empty string and error
func getVideoAspectRatio(filepath string) (string, error) {
	cmd := exec.Command("ffprobe", "-v", "error", "-print_format", "json", "-show_streams", filepath) // configure the command (does not run yet)
	cmdOutput := bytes.Buffer{}                                                                       // create bytes buffer for command output
	cmd.Stdout = &cmdOutput                                                                           // direct the command's stdout into the buffer
	err := cmd.Run()                                                                                  // execute the command and wait for it to finish
	if err != nil {
		return "", err
	}

	type ffprobeOutput struct {
		Streams []struct {
			Width  int `json:"width"`
			Height int `json:"height"`
		} `json:"streams"`
	}

	decoder := json.NewDecoder(&cmdOutput)
	ffprobeOut := ffprobeOutput{}
	err = decoder.Decode(&ffprobeOut)
	if err != nil {
		return "", err
	}

	// Ensure streams exist to avoid index panic in if/else block below
	if len(ffprobeOut.Streams) == 0 {
		return "", errors.New("no streams in video file")
	}

	stream := ffprobeOut.Streams[0]
	if stream.Height <= 0 || stream.Width <= 0 {
		return "", errors.New("invalid video file dimensions")
	}
	actualRatio := float64(stream.Width) / float64(stream.Height)
	portraitTargetRatio := 9.0 / 16.0
	landscapeTargetRatio := 16.0 / 9.0

	if getRelativeAspectRatioError(actualRatio, portraitTargetRatio) <= 0.01 { // 1% relative tolerance
		return "9:16", nil
	} else if getRelativeAspectRatioError(actualRatio, landscapeTargetRatio) <= 0.01 {
		return "16:9", nil
	} else {
		return "other", nil
	}
}

// Takes actual aspect ratio of a video and compares it to a target aspect ratio
// Returns a float64 relative error value
func getRelativeAspectRatioError(actualRatio, targetRatio float64) float64 {
	return math.Abs(actualRatio-targetRatio) / targetRatio
}
