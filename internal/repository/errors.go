package repository

import "errors"

var ErrNotFound = errors.New("not found")
var ErrPostScanFailed = errors.New("parsing post failed")
var ErrRowsLoop = errors.New("row loop encountered an error")
