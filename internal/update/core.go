package update

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"syscall"

	"github.com/bestruirui/octopus/internal/utils/shutdown"
	"github.com/charmbracelet/log"
)

func UpdateCore() error {
	log.Infof("start update core")

	filename, err := getDownloadFilename()
	if err != nil {
		log.Warnf("update core failed: %v", err)
		return err
	}

	downloadUrl := updateUrl + "/" + filename
	log.Infof("download url: %s", downloadUrl)
	data, err := doRequestWithFallback(downloadUrl)
	if err != nil {
		log.Warnf("download failed: %v", err)
		return err
	}

	execPath, err := os.Executable()
	if err != nil {
		log.Warnf("get executable path failed: %v", err)
		return err
	}

	retired := ""
	if runtime.GOOS == "windows" {
		// Windows holds the image file of a running process open without
		// sharing write access, so extracting the new binary over it fails
		// with "Access is denied". Renaming the image aside is allowed.
		retired, err = retireExecutable(execPath)
		if err != nil {
			log.Warnf("retire old executable failed: %v", err)
			return err
		}
	}

	if err := unzip(data, filepath.Dir(execPath)); err != nil {
		log.Warnf("unzip failed: %v", err)
		// put the retired binary back so the running one stays usable
		if retired != "" {
			if rbErr := os.Rename(retired, execPath); rbErr != nil {
				log.Warnf("restore retired executable failed: %v", rbErr)
			}
		}
		return err
	}

	log.Infof("update core success")
	go restartExecutable(execPath)
	return nil
}

// retiredExecutableSuffix is appended to the running executable's path when a
// self-update moves it aside so the new binary can take its place.
const retiredExecutableSuffix = ".old"

// retireExecutable renames the running executable aside so an update archive
// can place the new binary at the original path. Windows opens the image file
// of a running process without sharing write or delete access, so extracting
// over it fails with "Access is denied", while renaming the image is allowed.
func retireExecutable(execPath string) (string, error) {
	retired := execPath + retiredExecutableSuffix
	if err := os.Remove(retired); err != nil && !os.IsNotExist(err) {
		// surface real errors (locked files will fail the rename below anyway)
		return "", fmt.Errorf("remove stale %s failed: %w", retired, err)
	}
	if err := os.Rename(execPath, retired); err != nil {
		return "", err
	}
	return retired, nil
}

// CleanupRetiredExecutable removes the leftover binary from a previous
// self-update. Best effort: if the previous instance has not fully exited yet
// the file is still locked and will be cleaned up on the following start.
func CleanupRetiredExecutable() {
	execPath, err := os.Executable()
	if err != nil {
		return
	}
	if err := os.Remove(execPath + retiredExecutableSuffix); err != nil && !os.IsNotExist(err) {
		log.Debugf("cleanup retired executable failed: %v", err)
	}
}

// getDownloadFilename 返回当前平台对应的发布归档名称。
func getDownloadFilename() (string, error) {
	arch := runtime.GOARCH
	goos := runtime.GOOS

	switch goos {
	case "windows":
		switch arch {
		case "amd64":
			return "octopus-windows-amd64.zip", nil
		}
	case "darwin":
		switch arch {
		case "amd64":
			return "octopus-darwin-amd64.zip", nil
		case "arm64":
			return "octopus-darwin-arm64.zip", nil
		}
	case "linux":
		switch arch {
		case "386":
			return "octopus-linux-386.zip", nil
		case "amd64":
			return "octopus-linux-amd64.zip", nil
		case "arm":
			return "octopus-linux-arm.zip", nil
		case "arm64":
			return "octopus-linux-arm64.zip", nil
		}
	case "android":
		switch arch {
		case "386":
			return "octopus-android-386.zip", nil
		case "amd64":
			return "octopus-android-amd64.zip", nil
		case "arm":
			return "octopus-android-arm.zip", nil
		case "arm64":
			return "octopus-android-arm64.zip", nil
		}
	}
	return "", fmt.Errorf("unsupported platform: %s/%s", goos, arch)
}

func restartExecutable(execPath string) {
	shutdown.Shutdown()

	log.Infof("restarting: %q %q", execPath, os.Args[1:])

	if runtime.GOOS == "windows" {
		cmd := exec.Command(execPath, os.Args[1:]...)
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Start(); err != nil {
			log.Errorf("restarting failed: %v", err)
		}
		os.Exit(0)
	}

	if err := syscall.Exec(execPath, os.Args, os.Environ()); err != nil {
		log.Errorf("restarting failed: %v", err)
	}
}
