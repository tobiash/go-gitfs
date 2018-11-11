package gitfs

import (
	"gopkg.in/src-d/go-git.v4/plumbing"
	"gopkg.in/src-d/go-git.v4/plumbing/storer"
)

func resolveRef(s storer.Storer, refName plumbing.ReferenceName) (*plumbing.Reference, error) {
	ref, err := s.Reference(refName)
	if err != nil {
		return nil, err
	}
	for ref.Target() != "" {
		ref, err = s.Reference(ref.Target())
		if err != nil {
			return nil, err
		}
	}
	return ref, nil
}
