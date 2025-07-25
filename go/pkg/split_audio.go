package split_audio

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

func RunSplitAudio() {
	// Check ffmpeg is installed
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		fmt.Println("FFmpeg is not installed. Please install it (e.g., `brew install ffmpeg`).")
		return
	}

	reader := bufio.NewReader(os.Stdin)

	fmt.Print("Enter the path to the MP3 file: ")
	audioPath, _ := reader.ReadString('\n')
	audioPath = strings.TrimSpace(audioPath)

	if _, err := os.Stat(audioPath); os.IsNotExist(err) {
		fmt.Println("Error: File does not exist.")
		return
	}

	durationSec, err := getAudioDuration(audioPath)
	if err != nil {
		fmt.Println("Error getting duration:", err)
		return
	}
	fmt.Printf("Audio duration: %.2f seconds\n", durationSec)

	fmt.Println("Choose splitting method:")
	fmt.Println("1) Split by duration points (in seconds or hh:mm:ss)")
	fmt.Println("2) Split into equal parts")
	fmt.Print("Enter choice: ")
	choiceStr, _ := reader.ReadString('\n')
	choice := strings.TrimSpace(choiceStr)

	filename := strings.TrimSuffix(filepath.Base(audioPath), filepath.Ext(audioPath))
	outputDir := filepath.Join(filepath.Dir(audioPath), "split_output")
	os.MkdirAll(outputDir, os.ModePerm)

	switch choice {
	case "1":
		fmt.Print("Enter duration points separated by space: ")
		line, _ := reader.ReadString('\n')
		inputs := strings.Fields(line)

		var durations []float64
		for _, val := range inputs {
			sec, err := parseTimeToSeconds(val)
			if err != nil || sec > durationSec {
				fmt.Printf("Invalid time format or exceeds duration: %s\n", val)
				return
			}
			durations = append(durations, sec)
		}

		start := 0.0
		part := 1
		for _, end := range durations {
			segDur := end - start
			outPath := filepath.Join(outputDir, fmt.Sprintf("%s_%d.mp3", filename, part))
			runFFmpeg(audioPath, outPath, start, segDur)
			start = end
			part++
		}
		// final segment
		outPath := filepath.Join(outputDir, fmt.Sprintf("%s_%d.mp3", filename, part))
		runFFmpeg(audioPath, outPath, start, -1)

	case "2":
		fmt.Print("Enter number of parts: ")
		partsStr, _ := reader.ReadString('\n')
		numParts, _ := strconv.Atoi(strings.TrimSpace(partsStr))
		if numParts < 1 {
			fmt.Println("Invalid number of parts.")
			return
		}

		partDuration := durationSec / float64(numParts)
		for i := 0; i < numParts; i++ {
			start := partDuration * float64(i)
			outPath := filepath.Join(outputDir, fmt.Sprintf("%s_%d.mp3", filename, i+1))
			if i == numParts-1 {
				runFFmpeg(audioPath, outPath, start, -1)
			} else {
				runFFmpeg(audioPath, outPath, start, partDuration)
			}
		}

	default:
		fmt.Println("Invalid option.")
	}
	fmt.Println("Splitting complete!")
}

// getAudioDuration uses ffmpeg to extract duration in seconds
func getAudioDuration(path string) (float64, error) {
	cmd := exec.Command("ffmpeg", "-i", path)
	stderr, _ := cmd.CombinedOutput()
	re := regexp.MustCompile(`Duration: (\d+):(\d+):(\d+\.?\d*)`)
	matches := re.FindStringSubmatch(string(stderr))
	if len(matches) != 4 {
		return 0, fmt.Errorf("duration not found")
	}
	h, _ := strconv.Atoi(matches[1])
	m, _ := strconv.Atoi(matches[2])
	s, _ := strconv.ParseFloat(matches[3], 64)
	return float64(h*3600+m*60) + s, nil
}

// parseTimeToSeconds parses seconds or hh:mm:ss format
func parseTimeToSeconds(input string) (float64, error) {
	if strings.Contains(input, ":") {
		parts := strings.Split(input, ":")
		if len(parts) != 3 {
			return 0, fmt.Errorf("invalid hh:mm:ss format")
		}
		h, _ := strconv.Atoi(parts[0])
		m, _ := strconv.Atoi(parts[1])
		s, _ := strconv.ParseFloat(parts[2], 64)
		return float64(h*3600+m*60) + s, nil
	}
	return strconv.ParseFloat(input, 64)
}

// runFFmpeg calls ffmpeg to extract segment
func runFFmpeg(input, output string, start, duration float64) {
	args := []string{"-i", input, "-ss", fmt.Sprintf("%.2f", start)}
	if duration > 0 {
		args = append(args, "-t", fmt.Sprintf("%.2f", duration))
	}
	args = append(args, "-c", "copy", output)
	cmd := exec.Command("ffmpeg", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Run()
	fmt.Println("Created:", output)
}
