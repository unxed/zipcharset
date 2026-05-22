module github.com/unxed/zipcharset

go 1.25.5

require (
	github.com/klauspost/compress v1.18.7-0.20260521203646-ecdb779d8745
	github.com/unxed/localecp v0.1.0
	golang.org/x/text v0.21.0
)

replace github.com/unxed/localecp => ../localecp
