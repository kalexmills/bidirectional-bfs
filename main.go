package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"strconv"
	"strings"
	"time"
)

const DimacsFile = "data/USA-road-d.NY.gr"

func main() {
	if len(os.Args) != 3 {
		fmt.Println("usage: par-graph [start_id] [start_id]")
		os.Exit(1)
	}
	startV, err := strconv.Atoi(os.Args[1])
	if err != nil {
		fmt.Println("start_id must be an integer")
		os.Exit(1)
	}
	endV, err := strconv.Atoi(os.Args[2])
	if err != nil {
		fmt.Println("end_id must be an integer")
		os.Exit(1)
	}

	graph := loadDimacs(lines(DimacsFile))
	fmt.Println("Data loaded; graph has", len(graph), "nodes")

	fmt.Println("Searching for fewest-hops path from node", startV, "to node", endV)

	start := time.Now()
	fmt.Println(Bfs(graph, startV, endV))
	fmt.Println("Search took", time.Since(start))
}

// Bfs performs a parallel, cooperative breadth-first search of the provided graph. Two peers begin their search
// from the source and target nodes, communicating nodes as they are visited. When peer A learns that peer B has
// visited a node that A has already seen, the search frontiers have collided, and the fewest-hop path is returned.
func Bfs(graph map[int][]int, u int, v int) []int {
	done := make(chan int, 1)
	uChan := make(chan int, 64*1024) // these sizes are arbitrary
	vChan := make(chan int, 64*1024)
	graphCopy := copyMap(graph)
	uResult := bfsPeer(u, done, vChan, uChan, graph)
	vResult := bfsPeer(v, done, uChan, vChan, graphCopy)

	var uPath, vPath []int
	select {
	case uPath = <-uResult:
		if uPath == nil {
			return nil
		}
		vPath = <-vResult
	case vPath = <-vResult:
		if vPath == nil {
			return nil
		}
		uPath = <-uResult
	}

	return append(reverse(uPath), vPath[1:]...)
}

func copyMap(source map[int][]int) map[int][]int {
	result := make(map[int][]int)
	for k, v := range source {
		result[k] = append(result[k], v...)
	}
	return result
}

// reverse reverses the provided slice in place.
func reverse(slice []int) []int {
	n := len(slice)
	for i := 0; i < n/2; i++ {
		slice[i], slice[n-i-1] = slice[n-i-1], slice[i]
	}
	return slice
}

// bfsPeer performs a bfs from the start node. As they are visited, node ids are communicated to a peer, which is
// presumably starting from the other end of the path. Closes peerOut to indicate that the search has terminated.
func bfsPeer(start int, done chan int, peerOut chan<- int, peerIn <-chan int, graph map[int][]int) <-chan []int {
	result := make(chan []int, 1)
	go func() {
		frontier := []int{start}      // my search frontier
		visited := make(map[int]bool) // my visited array
		pred := make(map[int]int)     // predecessors I know about
		edgeCount := 0
		defer close(result)
		defer close(peerOut)
	finish:
		for {
			select {
			case meetNode := <-done:
				// peer found the meeting point, send back our half of the path; note pred is guaranteed
				// to contain a valid path, since peer found a meeting point based on a node we visited.
				result <- pathFrom(pred, meetNode, start)
				break finish
			case other := <-peerIn: // peer has visited another node
				if visited[other] {
					// search frontiers have merged, send back our half of the path.
					done <- other
					result <- pathFrom(pred, other, start)
					break finish
				}
			default:
				// expand search and send peer the visited node
				next := frontier[0]
				frontier = frontier[1:]
				neighbors := graph[next]
				for i := 0; i < len(neighbors); i++ {
					if _, iVisited := visited[neighbors[i]]; !iVisited {
						edgeCount++
						pred[neighbors[i]] = next
						frontier = append(frontier, neighbors[i])
						peerOut <- neighbors[i]
						delete(graph, next) // remove this node's outgoing edges to avoid much useless work
					}
				}
				visited[next] = true

				if len(frontier) == 0 {
					result <- nil // no path exists; report this fact
					break finish  // terminate search once the frontier is empty
				}
			}
		}
		fmt.Println("BFS starting from", start, "visited", len(visited), "nodes and", edgeCount, "edges")
	}()
	return result
}

// pathFrom reads the provided predecessor map to build a path from u to v. If pred contains
// a cycle, this method will detect it and return an error.
func pathFrom(pred map[int]int, u int, v int) []int {
	var result []int
	seen := make(map[int]bool, len(pred))
	curr := u
	for {
		if seen[curr] {
			log.Panicln("predecessor array contained a cycle")
		}
		result = append(result, curr)
		if curr == v {
			break
		}
		seen[curr] = true
		curr = pred[curr]
	}
	return result
}

// loadDimacs loads an undirected graph in the format defined by the 9th DIMACS implementation challenge.
// http://www.dis.uniroma1.it/~challenge9
func loadDimacs(lines <-chan string) map[int][]int {
	result := make(map[int][]int)
	lineNum := 0
	for s := range lines {
		lineNum++
		if len(s) > 0 {
			if s[0] == 'a' {
				tokens := strings.Fields(s)
				u := convToken(tokens[1], lineNum, 1)
				v := convToken(tokens[2], lineNum, 2)
				result[u] = append(result[u], v)
			}
		}
	}
	return result
}

// toSlice drains a channel of strings and returns them in a new slice.
func toSlice(lines <-chan string) []string {
	var linesSlice []string
	for s := range lines {
		linesSlice = append(linesSlice, s)
	}
	return linesSlice
}

// loadGraph parses a slice of space-delimited integers into a map of adjacency lists. Each
// line starts with the id of the source node, followed by its neighbors. An example file might
// contain
//
//   1 2 3
//   2 4 1
//   3 4 1
//   4 2 3
//
// which describes the undirected graph pictured below.
//
//    2
//   / \
//  1   4
//   \ /
//    3
//
func loadGraph(lines []string) map[int][]int {
	result := make(map[int][]int)
	for i := 0; i < len(lines); i++ {
		tokens := strings.Fields(lines[i])

		if len(tokens) != 0 {
			key := convToken(tokens[0], i, 0)
			for j := 1; j < len(tokens); j++ {
				token := convToken(tokens[j], i, j)
				result[key] = append(result[key], token)
			}
		}
	}
	return result
}

// convToken is a helper for loadGraph, which converts tokens to integers, or panics with
// a helpful error.
func convToken(str string, line, token int) int {
	result, err := strconv.Atoi(str)
	if err != nil {
		log.Panicf("Could not parse token %v at line %v as number", token, line)
	}
	return result
}

// lines parses a file representing a graph into a slice of lines
func lines(filename string) <-chan string {
	out := make(chan string)
	go func() {
		defer close(out)
		// Stream from file, so we know that we can
		file, err := os.Open(filename)
		if err != nil {
			log.Panic(err)
		}
		defer file.Close()

		buf := make([]byte, 64*1024)
		var next []byte = nil
		for {
			n, err := file.Read(buf)
			if err != nil {
				if err != io.EOF {
					log.Panic(err)
				}
				break
			}
			j := 0
			for i := 0; i < n; i++ {
				if buf[i] == '\n' {
					next = append(next, buf[j:i]...)
					out <- string(next)
					j = i + 1
					next = nil
				}
			}
			next = append(next, buf[j:n]...)
		}
		out <- string(next)
		return
	}()
	return out
}
