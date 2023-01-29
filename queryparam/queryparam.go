package queryparam

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/rs/zerolog/log"
)

var badQueryParams = map[string]struct{}{
	"cmp":         {},
	"d_id":        {},
	"ref":         {},
	"source":      {},
	"utm":         {},
	"xtor":        {},
	"ito":         {},
	"ocid":        {},
	"ftag":        {},
	"linkid":      {},
	"cid":         {},
	"smid":        {},
	"smtyp":       {},
	"taid":        {},
	"uniqueid":    {}, // seen on: cnet
	"servicetype": {}, // seen on: cnet
	"posttype":    {}, // seen on: cnet
	"thetime":     {}, // seen on: cnet
	"mkevt":       {},
	"siteid":      {},
	"fscl_post":   {},
	"norover":     {},
	"mkrid":       {},
	"mkcid":       {},
	"amdata":      {},
	"hash":        {},
	"cm_sp":       {},
	"tpcc":        {}, // seen on: techcrunch
	"cmpid":       {}, // seen on: bloomberg
	"reflink":     {}, // seen on: Wall Street Journal/WSJ
	"dcmp":        {}, // seen on: skynews
	"sh":          {}, // seen on: forbes
	"tag":         {}, // seen on: amazon
	"linkcode":    {}, // seen on: amazon
	"ref_":        {}, // seen on: amazon
	"psc":         {}, // seen on: amazon
	"th":          {}, // seen on: amazon
	"keywords":    {}, // seen on: amazon
	"sprefix":     {}, // seen on: amazon
	"sr":          {}, // seen on: amazon
	"qid":         {}, // seen on: amazon
	"crid":        {}, // seen on: amazon
}

var badQueryParamPrefixes = []string{
	"utm_", // google analytics?
	"at_",
	"ns_",
	"WT.", // seen on: gatesnotes
}

func Strip(ur url.URL) *url.URL {
	q := ur.Query()

	if len(q) > 0 {
		var stripCount int = 0
		for k, v := range q {
			msg := fmt.Sprintf("query parameter [%s=%s]", k, v)
			if _, ok := badQueryParams[strings.ToLower(k)]; ok {
				q.Del(k)
				msg = msg + "...stripping"
				stripCount++
				log.Debug().Msg(msg)
				continue
			}

			var stripped bool
			for _, prefix := range badQueryParamPrefixes {
				if strings.HasPrefix(k, prefix) {
					q.Del(k)
					msg = msg + "...stripping"
					log.Debug().Msg(msg)
					stripCount++
					stripped = true
					break
				}
			}
			if !stripped {
				log.Debug().Msg(msg + "...keeping")
			}
		}
		ur.RawQuery = q.Encode()
		log.Debug().Msgf("stripped %d query params from link", stripCount)
	}

	if ur.Fragment != "" {
		ur.Fragment = ""
	}
	if ur.RawFragment != "" {
		ur.RawFragment = ""
	}

	return &ur
}
