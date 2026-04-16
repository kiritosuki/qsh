package cmd

import (
	"fmt"
	"os"

	"github.com/kiritosuki/qsh/internal/ai"
	"github.com/kiritosuki/qsh/internal/clipboard"
	"github.com/kiritosuki/qsh/internal/output"
	"github.com/spf13/cobra"
)

var askCmd = &cobra.Command{
	Use:   "ask [query]",
	Short: "Ask anything about shell commands",
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		query := args[0]

		apiKey := os.Getenv("OPENAI_API_KEY")
		if apiKey == "" {
			fmt.Println("❌ Please set OPENAI_API_KEY")
			return
		}

		resp, err := ai.Query(apiKey, query)
		if err != nil {
			fmt.Println("Error:", err)
			return
		}

		output.Render(resp)

		if resp.Command != "" {
			clipboard.Copy(resp.Command)
			fmt.Println("\n📋 Command copied to clipboard")
		}
	},
}

func init() {
	rootCmd.AddCommand(askCmd)
}
