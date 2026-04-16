package output

import (
	"fmt"

	"github.com/kiritosuki/qsh/internal/ai"
)

type Resp struct {
	Command     string
	Explanation string
	Risk        string
}

func Render(r *ai.Response) {
	fmt.Println("\n🧠 Explanation:")
	fmt.Println(r.Explanation)

	if r.Command != "" {
		fmt.Println("\n💻 Command:")
		fmt.Println(r.Command)
	}

	if r.Risk == "high" {
		fmt.Println("\n⚠️  High risk command")
	}
}
