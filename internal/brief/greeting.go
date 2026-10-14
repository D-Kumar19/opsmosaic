package brief

import (
	"fmt"
	"time"
)

func greeting(name string, now time.Time, visit uint64) (string, string) {
	var title string
	var comments []string
	switch hour := now.Hour(); {
	case hour >= 5 && hour < 12:
		title = "☀️ Good morning, %s!"
		comments = []string{
			"☕ Need a coffee? Here's a little signal to start your day.",
			"🌱 Fresh day, fresh context. Let's learn one small thing.",
			"🥐 Breakfast for your brain, before the first deploy.",
		}
	case hour >= 12 && hour < 17:
		title = "🌤️ Good afternoon, %s!"
		comments = []string{
			"🔋 A tiny recharge for your DevOps brain.",
			"☕ Coffee break? Make it a curiosity break too.",
			"🧩 One small insight before the next big task.",
		}
	case hour >= 17 && hour < 22:
		title = "🌆 Good evening, %s!"
		comments = []string{
			"🍵 Wind down with a little tech, not another incident.",
			"✨ One last spark of curiosity for the day.",
			"🛋️ Your terminal brought something to read. No deploy required.",
		}
	default:
		title = "🦉 Hey night owl, %s!"
		comments = []string{
			"🌙 Still here? One small insight, then maybe some sleep.",
			"💤 Your clusters can reconcile. You deserve to recharge.",
			"🍵 Midnight curiosity? Keep it cozy, keep it read-only.",
		}
	}
	return fmt.Sprintf(title, cleanText(name)), comments[visit%uint64(len(comments))]
}
