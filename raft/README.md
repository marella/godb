Package raft
============

Implements the Raft Leader Election algorithm in Ongaro, Diego, and John Ousterhout. "In search of an understandable consensus algorithm." Draft of October 7 (2013).

## Install
Package raft uses the cluster package. It can be installed directly using:
<pre>go get github.com/marella/godb/raft</pre>

## Documentation
http://marella.github.io/godb/raft/

##Configuration
This package requires two config files - cluster_name.config (for raft configuration like Terms and Timeouts) and cluster_name.cluster.config (used by the cluster package to load the cluster configuration).

## Testing
Basic: After startup, one and at most one, leader is nominated. It is an error to have 0 leaders after a "sufficiently long time", and to have more than 1 leader at any time. The "sufficiently long time" is set as the parameter <code>WAITTIME</code> in milliseconds. Another parameter N denotes the N*WAITTIME milliseconds the test has to be run.
<pre>go test github.com/marella/godb/raft -run Basic</pre>
MinorityFailures: Only 3 out of 5 servers are started and checks if above condition is satisfied.
<pre>go test github.com/marella/godb/raft -run MinorityFailures</pre>
MajorityFailures: Only 2 out of 5 servers are started and checks if no leader is elected.
<pre>go test github.com/marella/godb/raft -run MajorityFailures</pre>
Note: Please run these tests separately and don't use <code>go test</code> as after a test finishes the ports are not yet freed!