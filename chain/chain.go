package chain

import (
	"errors"
	"net/http"
	"net/url"

	"github.com/djhword/theunwrapper/unwrap"
	"github.com/rs/zerolog/log"
)

var (
	ErrNoUnwrapperFound = errors.New("no unwrapper found")
)

func New(r *http.Request, unwrappers map[string]*unwrap.Unwrapper) (*ChainedUnwrapper, error) {
	var start *unwrap.Unwrapper
	if host := r.Header.Get("X-Forwarded-Host"); host != "" {
		start = unwrappers[host]
		if start == nil {
			return nil, ErrNoUnwrapperFound
		} else {
			log.Info().Msgf("using resolver: %s", start.Host())
		}
	} else {
		//start = unwrappers["t.co"]
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

func (c *ChainedUnwrapper) Err() error {
	return c.err
}

func (c *ChainedUnwrapper) Last() *url.URL {
	return c.ur
}

func (c *ChainedUnwrapper) Visited() []ChainEntry {
	return c.chain
}

func (c *ChainedUnwrapper) Next() bool {
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
