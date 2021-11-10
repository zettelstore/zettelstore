//-----------------------------------------------------------------------------
// Copyright (c) 2021 Detlef Stern
//
// This file is part of zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

// Package manager coordinates the various boxes and indexes of a Zettelstore.
package manager

import (
	"testing"

	"zettelstore.de/z/domain/id"
)

func TestSimple(t *testing.T) {
	t.Parallel()
	ar := newAnterooms(2)
	ar.Enqueue(id.Zid(1), arUpdate)
	action, zid, rno := ar.Dequeue()
	if zid != id.Zid(1) || action != arUpdate || rno != 1 {
		t.Errorf("Expected arUpdate/1/1, but got %v/%v/%v", action, zid, rno)
	}
	action, zid, _ = ar.Dequeue()
	if zid != id.Invalid && action != arDelete {
		t.Errorf("Expected invalid Zid, but got %v", zid)
	}
	ar.Enqueue(id.Zid(1), arUpdate)
	ar.Enqueue(id.Zid(2), arUpdate)
	if ar.first != ar.last {
		t.Errorf("Expected one room, but got more")
	}
	ar.Enqueue(id.Zid(3), arUpdate)
	if ar.first == ar.last {
		t.Errorf("Expected more than one room, but got only one")
	}

	count := 0
	for ; count < 1000; count++ {
		action, _, _ = ar.Dequeue()
		if action == arNothing {
			break
		}
	}
	if count != 3 {
		t.Errorf("Expected 3 dequeues, but got %v", count)
	}
}

func TestReset(t *testing.T) {
	t.Parallel()
	ar := newAnterooms(1)
	ar.Enqueue(id.Zid(1), arUpdate)
	ar.Reset()
	action, zid, _ := ar.Dequeue()
	if action != arReload || zid != id.Invalid {
		t.Errorf("Expected reload & invalid Zid, but got %v/%v", action, zid)
	}
	ar.Reload(id.NewSet(3, 4))
	ar.Enqueue(id.Zid(5), arUpdate)
	ar.Enqueue(id.Zid(5), arDelete)
	ar.Enqueue(id.Zid(5), arDelete)
	ar.Enqueue(id.Zid(5), arUpdate)
	if ar.first == ar.last || ar.first.next != ar.last /*|| ar.first.next.next != ar.last*/ {
		t.Errorf("Expected 2 rooms")
	}
	action, zid1, _ := ar.Dequeue()
	if action != arUpdate {
		t.Errorf("Expected arUpdate, but got %v", action)
	}
	action, zid2, _ := ar.Dequeue()
	if action != arUpdate {
		t.Errorf("Expected arUpdate, but got %v", action)
	}
	if !(zid1 == id.Zid(3) && zid2 == id.Zid(4) || zid1 == id.Zid(4) && zid2 == id.Zid(3)) {
		t.Errorf("Zids must be 3 or 4, but got %v/%v", zid1, zid2)
	}
	action, zid, _ = ar.Dequeue()
	if zid != id.Zid(5) || action != arUpdate {
		t.Errorf("Expected 5/arUpdate, but got %v/%v", zid, action)
	}
	action, zid, _ = ar.Dequeue()
	if action != arNothing || zid != id.Invalid {
		t.Errorf("Expected nothing & invalid Zid, but got %v/%v", action, zid)
	}

	ar = newAnterooms(1)
	ar.Reload(id.NewSet(id.Zid(6)))
	action, zid, _ = ar.Dequeue()
	if zid != id.Zid(6) || action != arUpdate {
		t.Errorf("Expected 6/arUpdate, but got %v/%v", zid, action)
	}
	action, zid, _ = ar.Dequeue()
	if action != arNothing || zid != id.Invalid {
		t.Errorf("Expected nothing & invalid Zid, but got %v/%v", action, zid)
	}

	ar = newAnterooms(1)
	ar.Enqueue(id.Zid(8), arUpdate)
	ar.Reload(nil)
	action, zid, _ = ar.Dequeue()
	if action != arNothing || zid != id.Invalid {
		t.Errorf("Expected nothing & invalid Zid, but got %v/%v", action, zid)
	}
}
