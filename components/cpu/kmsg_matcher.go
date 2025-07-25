package cpu

import (
	"regexp"
)

const (
	// e.g.,
	// INFO: task kcompactd1:1177 blocked for more than 120 seconds.
	// INFO: task jfsmount:136986 blocked for more than 120 seconds.
	// task jfsmount:136986 blocked for more than 120 seconds.
	// "echo 0 > /proc/sys/kernel/hung_task_timeout_secs" disables this message.
	// task:jfsmount        state:D stack:    0 pid: 9831 ppid:  9614 flags:0x00000004
	// ref. https://github.com/torvalds/linux/blob/68763b29e0a6441f57f9ee652bbf8e7bc59183e5/kernel/hung_task.c#L144-L145
	//
	// The number after the colon (e.g., 136986 in "jfsmount:136986") is the Process ID (PID) of the blocked task.
	eventBlockedTooLong   = "cpu_blocked_too_long"
	regexBlockedTooLong   = `(?:INFO: )?task ([^:]+:[\d]+).+blocked for more than \d+ seconds`
	messageBlockedTooLong = "CPU task blocked for more than 120 seconds"

	// e.g.,
	// [Sun Jan  5 18:28:55 2025] watchdog: BUG: soft lockup - CPU#18 stuck for 27s! [python3:2254956]
	// [Sun Jan  5 20:25:34 2025] watchdog: BUG: soft lockup - CPU#6 stuck for 48s! [python3:2257218]
	// [Sun Jan  5 18:33:00 2025] watchdog: BUG: soft lockup - CPU#0 stuck for 25s! [pt_data_pin:2273422]
	// [Sun Jan  5 19:42:34 2025] watchdog: BUG: soft lockup - CPU#4 stuck for 23s! [pt_autograd_0:2289563]
	// [Sun Jan  5 18:37:06 2025] watchdog: BUG: soft lockup - CPU#0 stuck for 27s! [cuda-EvtHandlr:2255424]
	//
	// The number after the colon (e.g., 2257218 in "[python3:2257218]") is the Process ID (PID) of the process causing the soft lockup.
	eventSoftLockup   = "cpu_soft_lockup"
	regexSoftLockup   = `soft lockup - CPU#\d+ stuck for \d+s! \[([^:]+:[\d]+)\]`
	messageSoftLockup = "CPU soft lockup detected, not releasing for a period of time"
)

var (
	compiledBlockedTooLong = regexp.MustCompile(regexBlockedTooLong)
	compiledSoftLockup     = regexp.MustCompile(regexSoftLockup)
)

// HasBlockedTooLong returns the task name with PID and true if the line indicates that a task is hung too long.
func HasBlockedTooLong(line string) (string, bool) {
	if match := compiledBlockedTooLong.FindStringSubmatch(line); match != nil {
		return match[1], true
	}
	return "", false
}

// HasSoftLockup returns the task name with PID and true if the line indicates a CPU soft lockup.
func HasSoftLockup(line string) (string, bool) {
	if match := compiledSoftLockup.FindStringSubmatch(line); match != nil {
		return match[1], true
	}
	return "", false
}

func Match(line string) (eventName string, message string) {
	// Check for blocked too long
	if processInfo, ok := HasBlockedTooLong(line); ok {
		return eventBlockedTooLong, messageBlockedTooLong + " (" + processInfo + ")"
	}

	// Check for soft lockup
	if processInfo, ok := HasSoftLockup(line); ok {
		return eventSoftLockup, messageSoftLockup + " (" + processInfo + ")"
	}

	return "", ""
}

type match struct {
	check     func(string) (string, bool)
	eventName string
	regex     string
	message   string
}

func getMatches() []match {
	return []match{
		{check: HasBlockedTooLong, eventName: eventBlockedTooLong, regex: regexBlockedTooLong, message: messageBlockedTooLong},
		{check: HasSoftLockup, eventName: eventSoftLockup, regex: regexSoftLockup, message: messageSoftLockup},
	}
}
