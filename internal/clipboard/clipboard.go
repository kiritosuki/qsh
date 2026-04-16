package clipboard

import (
	"os/exec"
	"runtime"
)

func Copy(text string) {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "linux":
		cmd = exec.Command("xclip", "-selection", "clipboard")
	default:
		return
	}

	in, _ := cmd.StdinPipe()
	cmd.Start()
	in.Write([]byte(text))
	in.Close()
	cmd.Wait()
}
