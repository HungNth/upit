//go:build !windows

package app

import "os"

func mutationTargetIsLink(_ string, info os.FileInfo) (bool, error) {
	return info.Mode()&os.ModeSymlink != 0, nil
}
