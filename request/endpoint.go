package request

type Endpoint int

const (
	// POST (API) and GET (webpage).

	// GET.
	ENDPOINTSTRING_FAVICON = "favicon.ico"

	// GET (Monitoring).
	ENDPOINTSTRING_LIVEZ   = "livez"
	ENDPOINTSTRING_READYZ  = "readyz"
	ENDPOINTSTRING_METRICS = "metrics"
	ENDPOINTSTRING_BUILD   = "debug/build"
	ENDPOINTSTRING_CONFIG  = "debug/config"
)
