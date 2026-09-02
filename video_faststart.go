package main

import "os/exec"

// Takes a filepath to a video
// Returns a new filepath to a file with "fast start" encoding
func processVideoForFastStart(filePath string) (string, error) {
	outputFilePath := filePath + ".processing"

	// ffmpeg -i <input_file> -c copy -movflags faststart -f mp4 <output_file>
	//
	// What it does:
	// Copies the audio/video streams from the input file into a new MP4 file
	// without re-encoding, while rearranging the file structure so metadata
	// (the moov atom) is placed at the front for faster streaming playback.
	//
	// Arguments breakdown:
	// -i <input_path>      : Specifies the input file path to read from.
	// -c copy              : Stream copy mode. Copies audio and video streams directly
	//                        without re-encoding, making the operation extremely fast.
	// -movflags faststart  : Moves the 'moov' atom (video metadata/index) to the
	//                        beginning of the MP4 file so playback can start immediately
	//                        before downloading the entire video.
	// -f mp4               : Explicitly forces the output container format to MP4.
	// <output_path>        : The destination file path for the processed output;
	// 						  output is written directly to this file path
	// 						  hence no need to create a bytes buffer to direct output to.
	cmd := exec.Command("ffmpeg", "-i", filePath, "-c", "copy", "-movflags", "faststart", "-f", "mp4", outputFilePath)
	err := cmd.Run()
	if err != nil {
		return "", err
	}

	return outputFilePath, nil
}
