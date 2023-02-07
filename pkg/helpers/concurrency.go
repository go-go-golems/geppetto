package helpers

func MergeChannels[A any](channels ...<-chan A) <-chan A {
	out := make(chan A)
	for _, ch := range channels {
		go func(ch <-chan A) {
			for v := range ch {
				out <- v
			}
		}(ch)
	}
	return out
}
