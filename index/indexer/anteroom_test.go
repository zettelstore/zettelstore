//-----------------------------------------------------------------------------
// Copyright (c) 2021 Detlef Stern
//
// This file is part of zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

// Package indexer allows to search for metadata and content.
package indexer

import (
	"testing"

	"zettelstore.de/z/domain/id"
)

func TestSimple(t *testing.T) {
	ar := newAnterooms(2)
	ar.Enqueue(id.Zid(1), true)
	zid, val := ar.Dequeue()
	if zid != id.Zid(1) || val != true {
		t.Errorf("Expected 1/true, but got %v/%v", zid, val)
	}
	zid, val = ar.Dequeue()
	if zid != id.Invalid && val != false {
		t.Errorf("Expected invalid Zid, but got %v", zid)
	}
	ar.Enqueue(id.Zid(1), true)
	ar.Enqueue(id.Zid(2), true)
	if ar.first != ar.last {
		t.Errorf("Expected one room, but got more")
	}
	ar.Enqueue(id.Zid(3), true)
	if ar.first == ar.last {
		t.Errorf("Expected more than one room, but got only one")
	}

	count := 0
	for ; ; count++ {
		zid, val := ar.Dequeue()
		if zid == id.Invalid && val == false {
			break
		}
	}
	if count != 3 {
		t.Errorf("Expected 3 dequeues, but got %v", count)
	}
}

func TestReset(t *testing.T) {
	ar := newAnterooms(1)
	ar.Enqueue(id.Zid(1), true)
	ar.Reset()
	zid, val := ar.Dequeue()
	if zid != id.Invalid && val != true {
		t.Errorf("Expected invalid Zid, but got %v/%v", zid, val)
	}
	ar.Reload([]id.Zid{id.Zid(2)}, map[id.Zid]bool{id.Zid(3): true, id.Zid(4): false})
	ar.Enqueue(id.Zid(5), true)
	ar.Enqueue(id.Zid(5), false)
	ar.Enqueue(id.Zid(5), false)
	ar.Enqueue(id.Zid(5), true)
	if ar.first == ar.last || ar.first.next == ar.last || ar.first.next.next != ar.last {
		t.Errorf("Expected 3 rooms")
	}
	zid, val = ar.Dequeue()
	if zid != id.Zid(2) || val != false {
		t.Errorf("Expected 2/false, but got %v/%v", zid, val)
	}
	zid1, val := ar.Dequeue()
	if val != true {
		t.Errorf("Expected true, but got %v", val)
	}
	zid2, val := ar.Dequeue()
	if val != true {
		t.Errorf("Expected true, but got %v", val)
	}
	if !(zid1 == id.Zid(3) && zid2 == id.Zid(4) || zid1 == id.Zid(4) && zid2 == id.Zid(3)) {
		t.Errorf("Zids must be 3 or 4, but got %v/%v", zid1, zid2)
	}
	zid, val = ar.Dequeue()
	if zid != id.Zid(5) || val != true {
		t.Errorf("Expected 5/true, but got %v/%v", zid, val)
	}
	zid, val = ar.Dequeue()
	if zid != id.Invalid && val != false {
		t.Errorf("Expected invalid Zid, but got %v", zid)
	}

	ar = newAnterooms(1)
	ar.Reload(nil, map[id.Zid]bool{id.Zid(6): true})
	zid, val = ar.Dequeue()
	if zid != id.Zid(6) || val != true {
		t.Errorf("Expected 6/true, but got %v/%v", zid, val)
	}
	zid, val = ar.Dequeue()
	if zid != id.Invalid && val != false {
		t.Errorf("Expected invalid Zid, but got %v", zid)
	}

	ar = newAnterooms(1)
	ar.Reload([]id.Zid{id.Zid(7)}, nil)
	zid, val = ar.Dequeue()
	if zid != id.Zid(7) || val != false {
		t.Errorf("Expected 7/false, but got %v/%v", zid, val)
	}
	zid, val = ar.Dequeue()
	if zid != id.Invalid && val != false {
		t.Errorf("Expected invalid Zid, but got %v", zid)
	}

	ar = newAnterooms(1)
	ar.Enqueue(id.Zid(8), true)
	ar.Reload(nil, nil)
	zid, val = ar.Dequeue()
	if zid != id.Invalid && val != false {
		t.Errorf("Expected invalid Zid, but got %v", zid)
	}
}
