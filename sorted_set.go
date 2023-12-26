// Copyright 2023 The nutsdb Author. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package nutsdb

import (
	"errors"
)

var (
	ErrSortedSetNotFound = errors.New("the sortedSet does not exist")

	ErrSortedSetMemberNotExist = errors.New("the member of sortedSet does not exist")

	ErrSortedSetIsEmpty = errors.New("the sortedSet if empty")
)

type SortedSet struct {
	db *DB
	M  map[string]*SkipList
}

func NewSortedSet(db *DB) *SortedSet {
	return &SortedSet{
		db: db,
		M:  map[string]*SkipList{},
	}
}

func (z *SortedSet) ZAdd(key string, score SCORE, value []byte, record *Record) error {
	sortedSet, ok := z.M[key]
	if !ok {
		z.M[key] = newSkipList(z.db)
		sortedSet = z.M[key]
	}

	return sortedSet.Put(score, value, record)
}

func (z *SortedSet) ZMembers(key string) (map[*Record]SCORE, error) {
	sortedSet, ok := z.M[key]

	if !ok {
		return nil, ErrSortedSetNotFound
	}

	nodes := sortedSet.dict

	members := make(map[*Record]SCORE, len(nodes))
	for _, node := range nodes {
		members[node.record] = node.score
	}

	return members, nil
}

func (z *SortedSet) ZCard(key string) (int, error) {
	if sortedSet, ok := z.M[key]; ok {
		return int(sortedSet.length), nil
	}

	return 0, ErrSortedSetNotFound
}

func (z *SortedSet) ZCount(key string, start SCORE, end SCORE, opts *GetByScoreRangeOptions) (int, error) {
	if sortedSet, ok := z.M[key]; ok {
		return len(sortedSet.GetByScoreRange(start, end, opts)), nil
	}
	return 0, ErrSortedSetNotFound
}

func (z *SortedSet) ZPeekMax(key string) (*Record, SCORE, error) {
	if sortedSet, ok := z.M[key]; ok {
		node := sortedSet.PeekMax()
		if node != nil {
			return node.record, node.score, nil
		}
		return nil, 0, ErrSortedSetIsEmpty
	}

	return nil, 0, ErrSortedSetNotFound
}

func (z *SortedSet) ZPopMax(key string) (*Record, SCORE, error) {
	if sortedSet, ok := z.M[key]; ok {
		node := sortedSet.PopMax()
		if node != nil {
			return node.record, node.score, nil
		}
		return nil, 0, ErrSortedSetIsEmpty
	}

	return nil, 0, ErrSortedSetNotFound
}

func (z *SortedSet) ZPeekMin(key string) (*Record, SCORE, error) {
	if sortedSet, ok := z.M[key]; ok {
		node := sortedSet.PeekMin()
		if node != nil {
			return node.record, node.score, nil
		}
		return nil, 0, ErrSortedSetIsEmpty
	}

	return nil, 0, ErrSortedSetNotFound
}

func (z *SortedSet) ZPopMin(key string) (*Record, SCORE, error) {
	if sortedSet, ok := z.M[key]; ok {
		node := sortedSet.PopMin()
		if node != nil {
			return node.record, node.score, nil
		}
		return nil, 0, ErrSortedSetIsEmpty
	}

	return nil, 0, ErrSortedSetNotFound
}

func (z *SortedSet) ZRangeByScore(key string, start SCORE, end SCORE, opts *GetByScoreRangeOptions) ([]*Record, []float64, error) {
	if sortedSet, ok := z.M[key]; ok {

		nodes := sortedSet.GetByScoreRange(start, end, opts)

		records := make([]*Record, len(nodes))
		scores := make([]float64, len(nodes))

		for i, node := range nodes {
			records[i] = node.record
			scores[i] = float64(node.score)
		}

		return records, scores, nil
	}

	return nil, nil, ErrSortedSetNotFound
}

func (z *SortedSet) ZRangeByRank(key string, start int, end int) ([]*Record, []float64, error) {
	if sortedSet, ok := z.M[key]; ok {

		nodes := sortedSet.GetByRankRange(start, end, false)

		records := make([]*Record, len(nodes))
		scores := make([]float64, len(nodes))

		for i, node := range nodes {
			records[i] = node.record
			scores[i] = float64(node.score)
		}

		return records, scores, nil
	}

	return nil, nil, ErrSortedSetNotFound
}

func (z *SortedSet) ZRem(key string, value []byte) (*Record, error) {
	if sortedSet, ok := z.M[key]; ok {
		hash, err := getFnv32(value)
		if err != nil {
			return nil, err
		}
		node := sortedSet.Remove(hash)
		if node != nil {
			return node.record, nil
		}
		return nil, ErrSortedSetMemberNotExist
	}

	return nil, ErrSortedSetNotFound
}

func (z *SortedSet) ZRemRangeByRank(key string, start int, end int) error {
	if sortedSet, ok := z.M[key]; ok {

		_ = sortedSet.GetByRankRange(start, end, true)
		return nil
	}

	return ErrSortedSetNotFound
}

func (z *SortedSet) getZRemRangeByRankNodes(key string, start int, end int) ([]*SkipListNode, error) {
	if sortedSet, ok := z.M[key]; ok {
		return sortedSet.GetByRankRange(start, end, false), nil
	}

	return []*SkipListNode{}, nil
}

func (z *SortedSet) ZRank(key string, value []byte) (int, error) {
	if sortedSet, ok := z.M[key]; ok {
		hash, err := getFnv32(value)
		if err != nil {
			return 0, err
		}
		rank := sortedSet.FindRank(hash)
		if rank == 0 {
			return 0, ErrSortedSetMemberNotExist
		}
		return rank, nil
	}
	return 0, ErrSortedSetNotFound
}

func (z *SortedSet) ZRevRank(key string, value []byte) (int, error) {
	if sortedSet, ok := z.M[key]; ok {
		hash, err := getFnv32(value)
		if err != nil {
			return 0, err
		}
		rank := sortedSet.FindRevRank(hash)
		if rank == 0 {
			return 0, ErrSortedSetMemberNotExist
		}
		return rank, nil
	}
	return 0, ErrSortedSetNotFound
}

func (z *SortedSet) ZScore(key string, value []byte) (float64, error) {
	if sortedSet, ok := z.M[key]; ok {
		node := sortedSet.GetByValue(value)
		if node != nil {
			return float64(sortedSet.GetByValue(value).score), nil
		}
		return 0, ErrSortedSetMemberNotExist
	}
	return 0, ErrSortedSetNotFound
}

func (z *SortedSet) ZExist(key string, value []byte) (bool, error) {
	if sortedSet, ok := z.M[key]; ok {
		hash, err := getFnv32(value)
		if err != nil {
			return false, err
		}
		_, ok := sortedSet.dict[hash]
		return ok, nil
	}
	return false, ErrSortedSetNotFound
}

// SCORE represents the score type.
type SCORE float64
