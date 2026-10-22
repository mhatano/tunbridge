package ui

import "embed"

//go:embed index.html styles.css i18n.js app.js
var Assets embed.FS
