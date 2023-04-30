package chain

import (
	"errors"
	"net/http"
	"net/url"

	"github.com/djhworld/theunwrapper/unwrap"
	"github.com/rs/zerolog/log"
)

var (
	ErrNoUnwrapperFound = errors.New("no unwrapper found")
)

func New(r *http.Request, unwrappers map[string]*unwrap.Unwrapper) (*ChainedUnwrapper, error) {
	var (
		host  string
		start *unwrap.Unwrapper
	)

	host = r.Header.Get("X-Forwarded-Host")
	if host == "" {
		host = r.Host
	}

	if host != "" {
		start = unwrappers[host]
		if start == nil {
			return nil, ErrNoUnwrapperFound
		} else {
			log.Info().Msgf("using unwrapper for: %s", start.Host())
		}
	} else {
		return nil, ErrNoUnwrapperFound
	}

	return &ChainedUnwrapper{
		ur:         r.URL,
		unwrapper:  start,
		chain:      []ChainEntry{},
		unwrappers: unwrappers,
		visitList:  make(map[string]struct{}),
	}, nil
}

// ChainEntry describes the transition from moving to one URL to the next, given
// the unwrapper that was used
type ChainEntry struct {
	From  url.URL
	To    url.URL
	Using unwrap.Unwrapper
}

type ChainedUnwrapper struct {
	ur         *url.URL
	unwrapper  *unwrap.Unwrapper
	chain      []ChainEntry
	visitList  map[string]struct{}
	unwrappers map[string]*unwrap.Unwrapper
	err        error
}

// Err returns the last error set
func (c *ChainedUnwrapper) Err() error {
	return c.err
}

// Err returns the currently set URL
func (c *ChainedUnwrapper) Last() *url.URL {
	return c.ur
}

// Err returns a slice of ChainEntry structs that describe the
// hops visited before finding the final URL
func (c *ChainedUnwrapper) Visited() []ChainEntry {
	return c.chain
}

// Next will visit the next endpoint in the chain.
// Returns false when the end of the chain is reached or if there is an error
func (c *ChainedUnwrapper) Next() bool {
	// try to ensure we don't visit the same URL twice
	if _, ok := c.visitList[c.ur.String()]; ok {
		log.Error().Msg("cycle detected!")
		c.err = errors.New("cycle detetected")
		return false
	}

	endpoint, result, err := c.unwrapper.Do(c.ur.Path[1:])
	if err != nil {
		c.err = err
		return false
	}
	c.visitList[endpoint.String()] = struct{}{}

	if result != nil {
		c.chain = append(c.chain, ChainEntry{From: *endpoint, To: *result, Using: *c.unwrapper})
		if r, ok := c.unwrappers[result.Host]; ok {
			c.unwrapper = r
			c.ur = result
			return true
		}
	} else {
		log.Error().Msg("failed to lookup!")
		c.unwrapper = nil
		return false
	}

	log.Debug().Msgf("finished, found: %s", result)

	c.unwrapper = nil
	c.ur = result
	return false
}
