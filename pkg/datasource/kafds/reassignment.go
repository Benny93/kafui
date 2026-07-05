package kafds

import (
	"fmt"
	"sort"
)

// computeReassignment computes a new per-partition replica assignment for a
// replication-factor change (TP-11). It is pure and exhaustively tested.
//
//   - current[i]  = the current replica broker ids of partition i
//   - leaders[i]  = the leader broker id of partition i (never removed)
//   - onlineBrokers = broker ids eligible to host replicas (offline brokers are
//     never chosen as new targets)
//   - newFactor    = the desired replication factor
//
// On increase, new replicas are placed on the least-loaded online brokers not
// already hosting the partition. On decrease, replicas are removed from the
// most-loaded brokers first, never removing the leader. Load is balanced across
// partitions as the plan is built.
func computeReassignment(current [][]int32, leaders []int32, onlineBrokers []int32, newFactor int) ([][]int32, error) {
	if len(current) == 0 {
		return nil, fmt.Errorf("topic has no partitions")
	}
	if newFactor < 1 {
		return nil, fmt.Errorf("replication factor must be at least 1")
	}
	if newFactor > len(onlineBrokers) {
		return nil, fmt.Errorf("replication factor %d exceeds the %d online broker(s)", newFactor, len(onlineBrokers))
	}
	currentRF := len(current[0])
	if newFactor == currentRF {
		return nil, fmt.Errorf("replication factor is already %d", newFactor)
	}

	onlineSet := make(map[int32]bool, len(onlineBrokers))
	for _, b := range onlineBrokers {
		onlineSet[b] = true
	}
	load := make(map[int32]int, len(onlineBrokers))
	for _, b := range onlineBrokers {
		load[b] = 0
	}

	result := make([][]int32, len(current))
	for i, replicas := range current {
		var leader int32 = -1
		if i < len(leaders) {
			leader = leaders[i]
		}
		var newReplicas []int32
		if newFactor > len(replicas) {
			newReplicas = increaseReplicas(replicas, onlineBrokers, onlineSet, load, newFactor)
		} else if newFactor < len(replicas) {
			newReplicas = decreaseReplicas(replicas, leader, load, newFactor)
		} else {
			newReplicas = append([]int32{}, replicas...)
		}
		result[i] = newReplicas
		for _, b := range newReplicas {
			load[b]++
		}
	}
	return result, nil
}

// increaseReplicas keeps the current replicas and appends least-loaded online
// brokers not already assigned until newFactor is reached.
func increaseReplicas(replicas []int32, onlineBrokers []int32, onlineSet map[int32]bool, load map[int32]int, newFactor int) []int32 {
	out := append([]int32{}, replicas...)
	assigned := make(map[int32]bool, len(out))
	for _, b := range out {
		assigned[b] = true
	}
	for len(out) < newFactor {
		best := int32(-1)
		bestLoad := int(^uint(0) >> 1)
		// Iterate a sorted copy for deterministic tie-breaking by broker id.
		for _, b := range sortedBrokers(onlineBrokers) {
			if assigned[b] {
				continue
			}
			if load[b] < bestLoad {
				best = b
				bestLoad = load[b]
			}
		}
		if best < 0 {
			break // no eligible online broker left
		}
		out = append(out, best)
		assigned[best] = true
	}
	return out
}

// decreaseReplicas removes the most-loaded non-leader replicas until newFactor is
// reached, always keeping the leader.
func decreaseReplicas(replicas []int32, leader int32, load map[int32]int, newFactor int) []int32 {
	removable := make([]int32, 0, len(replicas))
	for _, b := range replicas {
		if b != leader {
			removable = append(removable, b)
		}
	}
	// Most-loaded first; tie-break by higher broker id for determinism.
	sort.Slice(removable, func(i, j int) bool {
		if load[removable[i]] != load[removable[j]] {
			return load[removable[i]] > load[removable[j]]
		}
		return removable[i] > removable[j]
	})
	toRemove := len(replicas) - newFactor
	remove := make(map[int32]bool, toRemove)
	for i := 0; i < toRemove && i < len(removable); i++ {
		remove[removable[i]] = true
	}
	out := make([]int32, 0, newFactor)
	for _, b := range replicas {
		if !remove[b] {
			out = append(out, b)
		}
	}
	return out
}

func sortedBrokers(brokers []int32) []int32 {
	s := append([]int32{}, brokers...)
	sort.Slice(s, func(i, j int) bool { return s[i] < s[j] })
	return s
}
