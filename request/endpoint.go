package request

type Endpoint int

const (
	// POST (API) and GET (webpage).

	// GET.
	ENDPOINTSTRING_FAVICON      = "favicon.ico"
	ENDPOINTSTRING_ROBOTS       = "robots.txt"
	ENDPOINTSTRING_SECTIGO_LOGO = "sectigo_s.png"
	ENDPOINTSTRING_GITHUB_MARK  = "github-mark-32px.png"

	// GET (Monitoring).
	ENDPOINTSTRING_LIVEZ   = "livez"
	ENDPOINTSTRING_READYZ  = "readyz"
	ENDPOINTSTRING_METRICS = "metrics"
	ENDPOINTSTRING_BUILD   = "debug/build"
	ENDPOINTSTRING_CONFIG  = "debug/config"
)
