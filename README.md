# bidirectional-bfs
Parallel bi-directional BFS in Golang for fewest-hops paths (i.e. unweighted shortest paths)

A simple implementation of parallel [bidirectional BFS](https://en.wikipedia.org/wiki/Bidirectional_search) for point-to-point queries in graphs.
This mostly exists to help me test my newfound knowledge of Go's concurrency patterns, but it's pretty fast too.
The included data is one of the smaller sets (15 MB) from [the 9th DIMACs implementation challenge](http://users.diag.uniroma1.it/challenge9/download.shtml). The link contains bigger graphs you can try for yourself.

### Explanation

The search starts two go-routines, one searching from the source and the second searching from the destination. Each
goroutine sends newly found nodes to the other along a channel. When a node the other has visited is found, each 
goroutine reconstructs their half of the path and sends it to the main thread, which combines the data appropriately.

### Example usage and expected output
```
[12:12] bidirectional-bfs> go run . 12 251243
Data loaded; graph has 264346 nodes
Searching for fewest-hops path from node 12 to node 251243
BFS starting from 251243 visited 14952 nodes and 22504 edges
BFS starting from 12 visited 14683 nodes and 18252 edges
Search took 29.6079ms
[12 1 1363 1358 1355 1274 1143 1264 1239 1265 1229 1228 1227 941 936 933 935 923 930 919 927 925 924 906 901 755 677 663 660 659 366 365 364 349 347 345 342 320 343 297 294 295 196 202 181 4503 4502 4739 4735 4733 4730 4725 4724 4723 4701 4699 4639 4638 4640 4617 4481 4482 4480 4467 4478 4458 4457 4456 4429 4422 4428 4430 4420 4427 4426 4423 4425 4412 4436 260337 259696 259692 259691 258332 258330 258314 258313 258312 244615 244579 244582 244581 244592 244589 244555 244551 244552 244553 244567 244561 244562 244536 244560 244558 244533 244532 247201 247203 247198 247205 247215 247209 247212 247279 247280 247283 247282 245555 245549 245547 245546 245532 245534 245552 245537 245747 245699 245698 245573 245574 245572 245235 245229 245205 245204 245193 245188 245187 245186 245121 245120 244809 245119 245116 245115 261251 261253 261254 245136 241450 261256 261255 245152 242255 263784 242258 242257 263783 242263 242262 242267 242272 242211 242210 242335 242339 242338 250179 250178 250177 250190 249596 249580 249577 249573 249571 249572 249574 250203 250202 249728 249727 249730 249744 249748 238158 238157 249988 249989 250072 250069 249916 249915 249920 249877 249893 249925 249926 249928 249927 251243]
```
You can see that the search visits only about half of the graph's nodes, which is the reason to use bidirectional search.


##### And what about bigger graphs?

The largest graph in the DIMACs dataset contains 23 million nodes. Serialized, the file weighs in at 13 GB. My highly anecdotal and not-at-all scientific method of randomly typing in numbers (I did it twice!) yields search times around 13 seconds, which only visit less than 1% of the nodes in the graph. If I become interested again I might try something more scientific. The vast majority of that time is spent copying the graph in memory (the present implementation deletes visited nodes from the graph).
