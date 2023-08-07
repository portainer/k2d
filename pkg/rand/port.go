package rand

import (
	"errors"
	"math/rand"
	"sync"
	"time"
)

// PortGenerator provides a safe way to generate a random port that hasn't been used yet.
type PortGenerator struct {
	r     *rand.Rand
	mutex sync.Mutex
}

// NewPortGenerator creates a new PortGenerator with a random number generator
// seeded with the current time.
func NewPortGenerator() *PortGenerator {
	src := rand.NewSource(time.Now().UnixNano())
	r := rand.New(src)

	return &PortGenerator{
		r: r,
	}
}

// GenerateRandomPort is a method of the PortGenerator struct that generates a random port number
// in the range of 30000 to 32767, inclusive. It takes a pointer to a map[int]struct{} as an argument.
// The map keeps track of ports that are already in use. The function ensures that the generated port
// number is not already in use by checking its presence in the map.
// If the generated port is not in use, it will be added to the map and returned.
// If all ports in the range are already in use, the function will return an error.
//
// The function is thread-safe, it uses a mutex to ensure that only one goroutine can access
// the random number generator and the map at a time.
func (p *PortGenerator) GenerateRandomPort(usedPorts *map[int]struct{}) (int, error) {
	minPort := 30000
	maxPort := 32767

	p.mutex.Lock()
	defer p.mutex.Unlock()

	for {
		// Generate a random port in the range [minPort, maxPort]
		port := minPort + p.r.Intn(maxPort-minPort+1)

		// Check if the port is already used, if not, return it.
		if _, used := (*usedPorts)[port]; !used {
			(*usedPorts)[port] = struct{}{}
			return port, nil
		}

		// If all ports are used, return an error.
		if len(*usedPorts) == maxPort-minPort+1 {
			return 0, errors.New("unable to generate a random port number. All ports are in use")
		}
	}
}
