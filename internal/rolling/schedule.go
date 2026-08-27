package rolling

import "time"

// NextRolloverAfter 计算固定间隔滚动的下一个触发点。
func NextRolloverAfter(now time.Time, interval time.Duration, modulate bool) time.Time {
	if interval <= 0 {
		return time.Time{}
	}
	if !modulate {
		return now.Add(interval)
	}
	if interval == 24*time.Hour {
		year, month, day := now.Date()
		return time.Date(year, month, day+1, 0, 0, 0, 0, now.Location())
	}
	if interval == time.Hour {
		return now.Truncate(time.Hour).Add(time.Hour)
	}
	if interval == time.Minute {
		return now.Truncate(time.Minute).Add(time.Minute)
	}
	truncated := now.Truncate(interval)
	if truncated.Equal(now) {
		return now.Add(interval)
	}
	return truncated.Add(interval)
}

// NextCronRolloverAfter 计算 cron 调度的下一个触发点。
func NextCronRolloverAfter(now time.Time, cron *CronSchedule) time.Time {
	next, ok := cron.Next(now)
	if !ok {
		return time.Time{}
	}
	return next
}
