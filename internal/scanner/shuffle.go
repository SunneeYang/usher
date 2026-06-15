package scanner

import (
	"math/rand"
	"time"
)

func shuffleFiles(files []string, seed int64) {
	if seed == 0 {
		seed = time.Now().UnixNano()
	}
	r := rand.New(rand.NewSource(seed))
	r.Shuffle(len(files), func(i, j int) {
		files[i], files[j] = files[j], files[i]
	})
}
