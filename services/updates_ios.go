//go:build ios

package services

import (
	"context"
	"errors"
)

// iOS apps can't replace themselves: a sideloaded IPA is re-signed and
// installed by AltStore, SideStore or Sideloadly. Nova only tells the user a
// new release is out; the notification's Download button opens the release
// page (OpenReleasePage).

func (s *UpdateService) run(ctx context.Context) {
	if !s.beginRun() {
		return
	}
	defer s.endRun()
	if _, _, ok := s.checkLatest(ctx); ok {
		s.set(func(st *UpdateStatus) { st.State = "manual" })
	}
}

// Restart is never offered on iOS, since updates never reach "ready".
func (s *UpdateService) Restart(ctx context.Context) error {
	return errors.New("install the new version with the app you sideloaded Nova with")
}
