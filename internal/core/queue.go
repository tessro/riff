package core

// Queue represents a playback queue.
type Queue struct {
	Tracks       []Track `json:"tracks"`
	CurrentIndex int     `json:"current_index"`
}

// Current returns the currently playing track, or nil if the queue is empty.
func (q *Queue) Current() *Track {
	if q == nil || len(q.Tracks) == 0 || q.CurrentIndex < 0 || q.CurrentIndex >= len(q.Tracks) {
		return nil
	}
	return &q.Tracks[q.CurrentIndex]
}

// Upcoming returns tracks after the current position.
func (q *Queue) Upcoming() []Track {
	if q == nil || len(q.Tracks) == 0 || q.CurrentIndex < 0 || q.CurrentIndex >= len(q.Tracks)-1 {
		return nil
	}
	return q.Tracks[q.CurrentIndex+1:]
}

// Len returns the total number of tracks in the queue.
func (q *Queue) Len() int {
	if q == nil {
		return 0
	}
	return len(q.Tracks)
}

// IsEmpty returns true if the queue has no tracks.
func (q *Queue) IsEmpty() bool {
	return q.Len() == 0
}
