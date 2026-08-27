package hostruntime

import (
	"net/http"

	"m31labs.dev/gosx-studio/mediaruntime"
)

const MediaRuntimePath = RuntimeRoot + "/media-runtime.js"

// MediaRuntimeScript returns the shared Studio media picker runtime.
func MediaRuntimeScript() []byte {
	return mediaruntime.Script()
}

func MediaRuntimeHandler() http.Handler {
	return mediaruntime.Handler()
}
