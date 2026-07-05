package kafds

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func replicaSet(rs []int32) map[int32]bool {
	m := make(map[int32]bool, len(rs))
	for _, r := range rs {
		m[r] = true
	}
	return m
}

func TestComputeReassignment_Validation(t *testing.T) {
	current := [][]int32{{1, 2}, {2, 3}}
	leaders := []int32{1, 2}
	online := []int32{1, 2, 3}

	t.Run("equal factor rejected", func(t *testing.T) {
		_, err := computeReassignment(current, leaders, online, 2)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "already")
	})
	t.Run("factor below 1 rejected", func(t *testing.T) {
		_, err := computeReassignment(current, leaders, online, 0)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "at least 1")
	})
	t.Run("factor above online brokers rejected", func(t *testing.T) {
		_, err := computeReassignment(current, leaders, []int32{1, 2}, 3)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "exceeds")
	})
	t.Run("no partitions rejected", func(t *testing.T) {
		_, err := computeReassignment(nil, nil, online, 2)
		assert.Error(t, err)
	})
}

func TestComputeReassignment_Increase(t *testing.T) {
	// RF 2 -> 3 across 4 partitions, 3 online brokers.
	current := [][]int32{{1, 2}, {2, 3}, {3, 1}, {1, 2}}
	leaders := []int32{1, 2, 3, 1}
	online := []int32{1, 2, 3}

	result, err := computeReassignment(current, leaders, online, 3)
	require.NoError(t, err)
	require.Len(t, result, 4)

	load := map[int32]int{}
	for i, rs := range result {
		assert.Len(t, rs, 3, "partition %d should have RF 3", i)
		// Existing replicas preserved.
		for _, r := range current[i] {
			assert.Contains(t, rs, r)
		}
		// No duplicate replicas.
		assert.Len(t, replicaSet(rs), 3)
		// Only online brokers used.
		for _, r := range rs {
			assert.Contains(t, online, r)
		}
		for _, r := range rs {
			load[r]++
		}
	}
	// Balanced: 12 replica slots over 3 brokers = 4 each.
	for _, b := range online {
		assert.Equal(t, 4, load[b], "broker %d load", b)
	}
}

func TestComputeReassignment_Decrease_PreservesLeader(t *testing.T) {
	// RF 3 -> 2; leader must survive every partition.
	current := [][]int32{{1, 2, 3}, {2, 3, 1}, {3, 1, 2}}
	leaders := []int32{1, 2, 3}
	online := []int32{1, 2, 3}

	result, err := computeReassignment(current, leaders, online, 2)
	require.NoError(t, err)
	for i, rs := range result {
		assert.Len(t, rs, 2, "partition %d should have RF 2", i)
		assert.Contains(t, rs, leaders[i], "leader must be preserved for partition %d", i)
	}
}

func TestComputeReassignment_OfflineBrokerExcluded(t *testing.T) {
	// Broker 3 is offline; increasing RF must only add online brokers.
	current := [][]int32{{1}, {2}}
	leaders := []int32{1, 2}
	online := []int32{1, 2} // 3 is offline

	result, err := computeReassignment(current, leaders, online, 2)
	require.NoError(t, err)
	for _, rs := range result {
		for _, r := range rs {
			assert.NotEqual(t, int32(3), r, "offline broker 3 must never be assigned")
		}
	}
}
