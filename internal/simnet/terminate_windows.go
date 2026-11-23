package simnet

import "os"

// terminate stops the process; Windows has no SIGTERM to deliver, so a
// machine stopped this way reports a non-zero exit and is not clean.
func terminate(p *os.Process) { p.Kill() }
