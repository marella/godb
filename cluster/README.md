Package cluster
===============

A simple clustering library written in Go language.

## Install
Currently it has no dependencies. It can be installed directly using:
<pre>go get github.com/marella/godb/cluster</pre>

## Documentation
http://marella.github.io/godb/cluster/

## Testing
<pre>go test github.com/marella/godb/cluster</pre>

### Parameters
<table>
	<tr>
		<td>N</td><td>Number of servers in the cluster</td>
	</tr>
	<tr>
		<td>M</td><td>Number of messages to broadcast by each server</td>
	</tr>
	<tr>
		<td>TIMEDELAY</td><td>Timedelay in seconds between each broadcast of a server</td>
	</tr>
	<tr>
		<td>TIMEOUT</td><td>Timeout in seconds for receiving messages</td>
	</tr>
</table>

### Results
<table>
	<tr>
		<th>N</th><th>M</th><th>TIMEDELAY</th><th>TIMEOUT</th><th>Result</th>
	</tr>
	<tr>
		<td>100</td><td>1</td><td>0</td><td>10</td><td>PASS</td>
	</tr>
	<tr>
		<td>200</td><td>1</td><td>0</td><td>10</td><td>FAIL</td>
	</tr>
	<tr>
		<td>100</td><td>2</td><td>0</td><td>10</td><td>FAIL</td>
	</tr>
	<tr>
		<td>200</td><td>1</td><td>10</td><td>100</td><td>PASS</td>
	</tr>
	<tr>
		<td>100</td><td>2</td><td>10</td><td>100</td><td>FAIL</td>
	</tr>
	<tr>
		<td>2</td><td>400</td><td>0</td><td>10</td><td>PASS</td>
	</tr>
</table>