//go:build !darwin

package main

func runWithPlatformLifecycle(args []string) int {
	return run(args)
}
