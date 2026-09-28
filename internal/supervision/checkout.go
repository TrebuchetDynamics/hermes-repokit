package supervision

import _ "embed"

// CheckoutScript provides verify_checkout(plugin, root, revision), a read-only
// checker shared by native setup and observation. It never imports plugin code.
//
//go:embed checkout.py
var CheckoutScript string
