// Copyright 2017, OpenCensus Authors
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
//

package stats

import (
	"errors"
	"fmt"
	"testing"

	"github.com/census-instrumentation/opencensus-go/tags"
	"golang.org/x/net/context"
)

func Test_Worker_MeasureCreation(t *testing.T) {
	RestartWorker()

	if _, err := NewMeasureFloat64("MF1", "desc MF1", "unit"); err != nil {
		t.Errorf("NewMeasureFloat64(\"MF1\", \"desc MF1\") got error %v, want no error", err)
	}

	if _, err := NewMeasureFloat64("MF1", "Duplicate measure with same name as MF1.", "unit"); err == nil {
		t.Error("NewMeasureFloat64(\"MF1\", \"Duplicate MeasureFloat64 with same name as MF1.\") got no error, want no error")
	}

	if _, err := NewMeasureInt64("MF1", "Duplicate measure with same name as MF1.", "unit"); err == nil {
		t.Error("NewMeasureInt64(\"MF1\", \"Duplicate MeasureInt64 with same name as MF1.\") got no error, want no error")
	}

	if _, err := NewMeasureFloat64("MF2", "desc MF2", "unit"); err != nil {
		t.Errorf("NewMeasureFloat64(\"MF2\", \"desc MF2\") got error %v, want no error", err)
	}

	if _, err := NewMeasureInt64("MI1", "desc MI1", "unit"); err != nil {
		t.Errorf("NewMeasureInt64(\"MI1\", \"desc MI1\") got error %v, want no error", err)
	}

	if _, err := NewMeasureInt64("MI1", "Duplicate measure with same name as MI1.", "unit"); err == nil {
		t.Error("NewMeasureInt64(\"MI1\", \"Duplicate NewMeasureInt64 with same name as MI1.\") got no error, want no error")
	}

	if _, err := NewMeasureFloat64("MI1", "Duplicate measure with same name as MI1.", "unit"); err == nil {
		t.Error("NewMeasureFloat64(\"MI1\", \"Duplicate NewMeasureFloat64 with same name as MI1.\") got no error, want no error")
	}
}

func Test_Worker_MeasureByName(t *testing.T) {
	RestartWorker()

	someError := errors.New("some error")
	mf1, err := NewMeasureFloat64("MF1", "desc MF1", "unit")
	if err != nil {
		t.Errorf("NewMeasureFloat64(\"MF1\", \"desc MF1\") got error %v, want no error", err)
	}
	mf2, err := NewMeasureFloat64("MF2", "desc MF2", "unit")
	if err != nil {
		t.Errorf("NewMeasureFloat64(\"MF2\", \"desc MF2\") got error %v, want no error", err)
	}
	mi1, err := NewMeasureInt64("MI1", "desc MI1", "unit")
	if err != nil {
		t.Errorf("NewMeasureInt64(\"MI1\", \"desc MI1\") got error %v, want no error", err)
	}

	type testCase struct {
		label string
		name  string
		m     Measure
		err   error
	}

	tcs := []testCase{
		{
			"0",
			mf1.Name(),
			mf1,
			nil,
		},
		{
			"1",
			"MF1",
			mf1,
			nil,
		},
		{
			"2",
			mf2.Name(),
			mf2,
			nil,
		},
		{
			"3",
			"MF2",
			mf2,
			nil,
		},
		{
			"4",
			mi1.Name(),
			mi1,
			nil,
		},
		{
			"5",
			"MI1",
			mi1,
			nil,
		},
		{
			"6",
			"other",
			nil,
			someError,
		},
	}

	for _, tc := range tcs {
		m, err := GetMeasureByName(tc.name)
		if (err != nil) != (tc.err != nil) {
			t.Errorf("GetMeasureByName. got error %v, want %v. Test case: %v", err, tc.err, tc.label)
		}

		if m != tc.m {
			t.Errorf("GetMeasureByName. got measure %v, want measure %v. Test case: %v", m, tc.m, tc.label)
		}
	}
}

func Test_Worker_MeasureDelete(t *testing.T) {
	someError := errors.New("some error")

	registerViewFunc := func(viewName string) func(m Measure) error {
		return func(m Measure) error {
			switch x := m.(type) {
			case *MeasureInt64:
				v := NewView(viewName, "", nil, x, nil, nil)
				return RegisterView(v)
			case *MeasureFloat64:
				v := NewView(viewName, "", nil, x, nil, nil)
				return RegisterView(v)
			default:
				return fmt.Errorf("cannot create view '%v' with measure '%v'", viewName, m.Name())
			}
		}
	}

	type vRegistrations struct {
		measureName string
		regFunc     func(m Measure) error
	}

	type deletion struct {
		name      string
		getErr    error
		deleteErr error
	}

	type testCase struct {
		label         string
		measureNames  []string
		registrations []vRegistrations
		deletions     []deletion
	}

	tcs := []testCase{
		{
			"0",
			[]string{"mi1"},
			[]vRegistrations{},
			[]deletion{
				{"mi1", nil, nil},
			},
		},
		{
			"1",
			[]string{"mi1"},
			[]vRegistrations{
				{
					"mi1",
					registerViewFunc("vw1"),
				},
			},
			[]deletion{
				{"mi1", nil, someError},
				{"mi2", someError, nil},
			},
		},
		{
			"2",
			[]string{"mi1", "mi2"},
			[]vRegistrations{
				{
					"mi1",
					registerViewFunc("vw1"),
				},
			},
			[]deletion{
				{"mi1", nil, someError},
				{"mi2", nil, nil},
			},
		},
	}

	for _, tc := range tcs {
		RestartWorker()

		for _, n := range tc.measureNames {
			if _, err := NewMeasureInt64(n, "some desc", "unit"); err != nil {
				t.Errorf("Creating measure got error '%v'. test case: '%v'", err, tc.label)
			}
		}

		for _, r := range tc.registrations {
			m, err := GetMeasureByName(r.measureName)
			if err != nil {
				t.Errorf("Retrieving measure '%v' got '%v', want no error. test case: '%v'", r.measureName, err, tc.label)
				continue
			}
			if err = r.regFunc(m); err != nil {
				t.Errorf("Registring view got '%v', want no error. test case: '%v'", err, tc.label)
				continue
			}
		}

		for _, d := range tc.deletions {
			m, err := GetMeasureByName(d.name)
			if (err != nil) != (d.getErr != nil) {
				t.Errorf("Retrieving measure got '%v', want '%v'. test case: '%v'", err, d.getErr, tc.label)
				continue
			}

			if err != nil {
				// err was expected to be nil
				continue
			}

			err = DeleteMeasure(m)
			if (err != nil) != (d.deleteErr != nil) {
				t.Errorf("Deleting measure got '%v', want '%v'. test case: '%v'", err, d.deleteErr, tc.label)
			}

			var deleted bool
			if err == nil {
				deleted = true
			}

			if _, err := GetMeasureByName(d.name); deleted && (err == nil) {
				t.Errorf("Retrieving measure succedded after delete succeded. test case: '%v'", tc.label)
				continue
			}
		}
	}
}

func Test_Worker_ViewRegistration(t *testing.T) {
	someError := errors.New("some error")

	sc1 := make(chan *ViewData)

	type registerWant struct {
		vID                string
		err                error
		isForcedCollection bool
	}
	type unregisterWant struct {
		vID string
		err error
	}
	type byNameWant struct {
		name string
		vID  string
		err  error
	}

	type subscription struct {
		c   chan *ViewData
		vID string
		err error
	}
	type testCase struct {
		label         string
		regs          []registerWant
		subscriptions []subscription
		unregs        []unregisterWant
		bynames       []byNameWant
	}
	tcs := []testCase{
		{
			"0",
			[]registerWant{
				{
					"v1ID",
					nil,
					true,
				},
			},
			[]subscription{
				{
					sc1,
					"v1ID",
					nil,
				},
			},
			[]unregisterWant{
				{
					"v1ID",
					someError,
				},
			},
			[]byNameWant{
				{
					"VF1",
					"v1ID",
					nil,
				},
				{
					"VF2",
					"vNilID",
					someError,
				},
			},
		},
		{
			"1",
			[]registerWant{
				{
					"v1ID",
					nil,
					false,
				},
				{
					"v2ID",
					nil,
					true,
				},
			},
			[]subscription{
				{
					sc1,
					"v1ID",
					nil,
				},
			},
			[]unregisterWant{
				{
					"v1ID",
					someError,
				},
				{
					"v2ID",
					someError,
				},
			},
			[]byNameWant{
				{
					"VF1",
					"v1ID",
					nil,
				},
				{
					"VF2",
					"v2ID",
					nil,
				},
			},
		},
		{
			"2",
			[]registerWant{
				{
					"v1ID",
					nil,
					true,
				},
			},
			[]subscription{
				{
					sc1,
					"v1ID",
					nil,
				},
				{
					sc1,
					"v1SameNameID",
					someError,
				},
			},
			[]unregisterWant{
				{
					"v1ID",
					someError,
				},
				{
					"v1SameNameID",
					nil,
				},
			},
			[]byNameWant{
				{
					"VF1",
					"v1ID",
					nil,
				},
			},
		},
	}

	for _, tc := range tcs {
		RestartWorker()

		mf1, _ := NewMeasureFloat64("MF1", "desc MF1", "unit")
		mf2, _ := NewMeasureFloat64("MF2", "desc MF2", "unit")

		views := make(map[string]View)
		views["v1ID"] = NewView("VF1", "desc VF1", nil, mf1, nil, nil)
		views["v1SameNameID"] = NewView("VF1", "desc duplicate name VF1.", nil, mf1, nil, nil)
		views["v2ID"] = NewView("VF2", "desc VF2", nil, mf2, nil, nil)
		views["vNilID"] = nil

		for _, reg := range tc.regs {
			v := views[reg.vID]

			err := RegisterView(v)
			if (err != nil) != (reg.err != nil) {
				t.Errorf("RegisterView. got error %v, want %v. Test case: %v", err, reg.err, tc.label)
			}
			ForceCollection(v)
		}

		for _, s := range tc.subscriptions {
			v := views[s.vID]
			err := SubscribeToView(v, s.c)
			if (err != nil) != (s.err != nil) {
				t.Errorf("SubscribeToView. got error %v, want %v. Test case: %v", err, s.err, tc.label)
			}
		}

		for _, unreg := range tc.unregs {
			v := views[unreg.vID]
			err := UnregisterView(v)
			if (err != nil) != (unreg.err != nil) {
				t.Errorf("UnregisterView. got error %v, want %v. Test case: %v", err, unreg.err, tc.label)
			}
		}

		for _, byname := range tc.bynames {
			v, err := GetViewByName(byname.name)
			if (err != nil) != (byname.err != nil) {
				t.Errorf("GetViewByName. got error %v, want %v. Test case: %v", err, byname.err, tc.label)
			}

			wantV := views[byname.vID]
			if v != wantV {
				t.Errorf("GetViewByName. got view '%v' '%v', want view %v. Test case: %v", v.Name(), v.Description(), wantV, tc.label)
			}
		}
	}
}

func Test_Worker_RecordFloat64(t *testing.T) {
	RestartWorker()

	someError := errors.New("some error")
	m, err := NewMeasureFloat64("MF1", "desc MF1", "unit")
	if err != nil {
		t.Errorf("NewMeasureFloat64(\"MF1\", \"desc MF1\") got error '%v', want no error", err)
	}

	k1, _ := tags.CreateKeyString("k1")
	k2, _ := tags.CreateKeyString("k2")
	tagsSet := tags.NewTagSetBuilder(nil).
		InsertString(k1, "v1").
		InsertString(k2, "v2").
		Build()
	ctx := tags.NewContext(context.Background(), tagsSet)

	v1 := NewView("VF1", "desc VF1", []tags.Key{k1, k2}, m, NewAggregationCount(), NewWindowCumulative())
	v2 := NewView("VF2", "desc VF2", []tags.Key{k1, k2}, m, NewAggregationCount(), NewWindowCumulative())

	c1 := make(chan *ViewData)
	type subscription struct {
		v View
		c chan *ViewData
	}
	type want struct {
		v    View
		rows []*Row
		err  error
	}
	type testCase struct {
		label           string
		registrations   []View
		subscriptions   []subscription
		forcedCollected []View
		records         []float64
		wants           []want
	}

	tcs := []testCase{
		{
			"0",
			[]View{v1, v2},
			[]subscription{},
			[]View{},
			[]float64{1, 1},
			[]want{{v1, nil, someError}, {v2, nil, someError}},
		},
		{
			"1",
			[]View{v1, v2},
			[]subscription{},
			[]View{v1},
			[]float64{1, 1},
			[]want{
				{
					v1,
					[]*Row{
						{
							[]tags.Tag{{k1, []byte("v1")}, {k2, []byte("v2")}},
							newAggregationCountValue(2),
						},
					},
					nil,
				},
				{v2, nil, someError},
			},
		},
		{
			"2",
			[]View{v1, v2},
			[]subscription{},
			[]View{v1, v2},
			[]float64{1, 1},
			[]want{
				{
					v1,
					[]*Row{
						{
							[]tags.Tag{{k1, []byte("v1")}, {k2, []byte("v2")}},
							newAggregationCountValue(2),
						},
					},
					nil,
				},
				{
					v2,
					[]*Row{
						{
							[]tags.Tag{{k1, []byte("v1")}, {k2, []byte("v2")}},
							newAggregationCountValue(2),
						},
					},
					nil,
				},
			},
		},
		{
			"3",
			[]View{v1, v2},
			[]subscription{{v1, c1}},
			[]View{},
			[]float64{1, 1},
			[]want{
				{
					v1,
					[]*Row{
						{
							[]tags.Tag{{k1, []byte("v1")}, {k2, []byte("v2")}},
							newAggregationCountValue(2),
						},
					},
					nil,
				},
				{v2, nil, someError},
			},
		},
		{
			"4",
			[]View{v1, v2},
			[]subscription{{v1, c1}, {v2, c1}},
			[]View{},
			[]float64{1, 1},
			[]want{
				{
					v1,
					[]*Row{
						{
							[]tags.Tag{{k1, []byte("v1")}, {k2, []byte("v2")}},
							newAggregationCountValue(2),
						},
					},
					nil,
				},
				{
					v2,
					[]*Row{
						{
							[]tags.Tag{{k1, []byte("v1")}, {k2, []byte("v2")}},
							newAggregationCountValue(2),
						},
					},
					nil,
				},
			},
		},
		{
			"5",
			[]View{v1, v2},
			[]subscription{{v1, c1}},
			[]View{v2},
			[]float64{1, 1, 10},
			[]want{
				{
					v1,
					[]*Row{
						{
							[]tags.Tag{{k1, []byte("v1")}, {k2, []byte("v2")}},
							newAggregationCountValue(3),
						},
					},
					nil,
				},
				{
					v2,
					[]*Row{
						{
							[]tags.Tag{{k1, []byte("v1")}, {k2, []byte("v2")}},
							newAggregationCountValue(3),
						},
					},
					nil,
				},
			},
		},
	}

	for _, tc := range tcs {
		for _, v := range tc.registrations {
			if err := RegisterView(v); err != nil {
				t.Fatalf("RegisterView '%v' got error '%v', want no error for test case: '%v'", v.Name(), err, tc.label)
			}
		}

		for _, s := range tc.subscriptions {
			if err := SubscribeToView(s.v, s.c); err != nil {
				t.Fatalf("SubscribeToView '%v' got error '%v', want no error for test case: '%v'", s.v.Name(), err, tc.label)
			}
		}

		for _, v := range tc.forcedCollected {
			if err := ForceCollection(v); err != nil {
				t.Fatalf("ForceCollection '%v' got error '%v', want no error for test case: '%v'", v.Name(), err, tc.label)
			}
		}

		for _, value := range tc.records {
			RecordFloat64(ctx, m, value)
		}

		for _, w := range tc.wants {
			gotRows, err := RetrieveData(w.v)
			if (err != nil) != (w.err != nil) {
				t.Fatalf("RetrieveData '%v' got error '%v', want no error for test case: '%v'", w.v.Name(), err, tc.label)
			}
			for _, gotRow := range gotRows {
				if !ContainsRow(w.rows, gotRow) {
					t.Errorf("got unexpected row '%v' for test case: '%v'", gotRow, tc.label)
					break
				}
			}

			for _, wantRow := range w.rows {
				if !ContainsRow(gotRows, wantRow) {
					t.Errorf("want row '%v' for test case: '%v'. Not received", wantRow, tc.label)
					break
				}
			}
		}

		// cleaning up
		for _, v := range tc.forcedCollected {
			if err := StopForcedCollection(v); err != nil {
				t.Fatalf("StopForcedCollection '%v' got error '%v', want no error for test case: '%v'", v.Name(), err, tc.label)
			}
		}

		for _, s := range tc.subscriptions {
			if err := UnsubscribeFromView(s.v, s.c); err != nil {
				t.Fatalf("UnsubscribeFromView '%v' got error '%v', want no error for test case: '%v'", s.v.Name(), err, tc.label)
			}
		}

		for _, v := range tc.registrations {
			if err := UnregisterView(v); err != nil {
				t.Fatalf("UnregisterView '%v' got error '%v', want no error for test case: '%v'", v.Name(), err, tc.label)
			}
		}
	}
}
