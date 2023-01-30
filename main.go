package main

import (
	"embed"
	"encoding/json"
	"flag"
	"fmt"
	"html/template"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/djhworld/theunwrapper/chain"
	"github.com/djhworld/theunwrapper/queryparam"
	"github.com/djhworld/theunwrapper/unwrap"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

var flagPort = flag.Uint("port", 8080, "port")
var flagUpstreamDNS = flag.String("upstream-dns", "1.1.1.1:53", "upstream dns IP:Port, defaults to cloudflare")
var flagLogFormat = flag.String("log-format", "json", "log format, options are [pretty,json]")
var flagLogDebug = flag.Bool("debug", false, "turn on debug logging")

var knownUnwrappers map[string]*unwrap.Unwrapper

// content holds our static index.html page and configurations
//
//go:embed templates/*
//go:embed config/*
var embedFS embed.FS

var tmpl = template.Must(template.New("index.html").Funcs(template.FuncMap{
	"stripParams": queryparam.Strip,
	"toString":    toString,
	"ellipsis":    ellipsis,
}).ParseFS(embedFS, "templates/*.html"))

type Output struct {
	Visited []chain.ChainEntry
	Result  *url.URL
	Err     error
}

func toString(stringer fmt.Stringer) string {
	return stringer.String()
}
func ellipsis(input string) string {
	if len(input) > 35 {
		return fmt.Sprintf("%s...(truncated)", input[0:35])
	}
	return input
}

func handler(w http.ResponseWriter, r *http.Request) {
	var output Output
	chained, err := chain.New(r, knownUnwrappers)
	if err != nil {
		log.Error().Err(err).Send()
		w.WriteHeader(http.StatusBadRequest)
		output.Err = err
		tmpl.Execute(w, output)
		return
	}

	for chained.Next() {
	}

	output.Visited = chained.Visited()

	if chained.Err() != nil || chained.Last() == nil {
		w.WriteHeader(http.StatusInternalServerError)
		output.Err = chained.Err()
		tmpl.Execute(w, output)
		return
	}

	output.Result = chained.Last()
	output.Err = nil

	if err := tmpl.Execute(w, output); err != nil {
		log.Error().Err(err).Send()
	} else {
		log.Info().Msg("completed processing request")
	}
}

func main() {
	flag.Parse()
	configureLogging()

	log.Info().Msgf("starting unwrapper service on port: %d", *flagPort)
	loadUnwrappers()
	http.HandleFunc("/", handler)
	http.ListenAndServe(fmt.Sprintf(":%d", *flagPort), nil)
}

func configureLogging() {
	switch *flagLogFormat {
	case "pretty":
		log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339})
	case "json":
	default:
		log.Fatal().Msgf("unknown log format: %s", *flagLogFormat)
	}

	zerolog.SetGlobalLevel(zerolog.InfoLevel)

	if *flagLogDebug {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	}
}

type unwrapperDef struct {
	Host        string
	Description string
}

func loadUnwrappers() {
	f, err := embedFS.Open("config/unwrappers.json")
	if err != nil {
		log.Fatal().Err(err).Send()
	}
	defer f.Close()

	decoder := json.NewDecoder(f)
	var unwrapperDefs []unwrapperDef
	if err := decoder.Decode(&unwrapperDefs); err != nil {
		log.Fatal().Err(err).Send()
	}

	knownUnwrappers = make(map[string]*unwrap.Unwrapper)
	for _, d := range unwrapperDefs {
		log.Debug().Msgf("creating unwrapper for: %s (%s)", d.Host, d.Description)
		knownUnwrappers[d.Host] = unwrap.New(d.Host, d.Description, *flagUpstreamDNS)
	}
	log.Info().Msgf("loaded %d link unwrappers", len(knownUnwrappers))
}
