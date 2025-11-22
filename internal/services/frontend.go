package services

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

func PullFrontend(repoURL string) error {
	frontendDir := "./frontend"

	if _, err := os.Stat(frontendDir + "/.git"); err == nil {
		fmt.Println("Frontend repo exists, pulling latest...")
		cmd := exec.Command("git", "-C", frontendDir, "pull")
		output, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("git pull failed: %v - %s", err, output)
		}
		fmt.Println(string(output))
		return nil
	}

	fmt.Println("Cloning frontend repo...")
	if err := os.RemoveAll(frontendDir); err != nil {
		return err
	}

	if err := os.MkdirAll(frontendDir, 0755); err != nil {
		return err
	}

	repoURL = strings.TrimSpace(repoURL)
	cmd := exec.Command("git", "clone", repoURL, frontendDir)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git clone failed: %v - %s", err, output)
	}

	fmt.Println(string(output))
	return nil
}
