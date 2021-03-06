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
	"fmt"
	"math"
)

// AggregationValue is the interface for all types of aggregations values.
type AggregationValue interface {
	String() string
	equal(other AggregationValue) bool
	isAggregate() bool
	addSample(v interface{})
	multiplyByFraction(fraction float64) AggregationValue
	addToIt(other AggregationValue)
	clear()
}

// AggregationCountValue is the aggregated data for an AggregationCountInt64.
type AggregationCountValue int64

// NewTestingAggregationCountValue is used to facilitate testing only and
// should not be invoked in production.
func NewTestingAggregationCountValue(v int64) *AggregationCountValue {
	return newAggregationCountValue(v)
}

func newAggregationCountValue(v int64) *AggregationCountValue {
	tmp := AggregationCountValue(v)
	return &tmp
}

func (a *AggregationCountValue) isAggregate() bool { return true }

func (a *AggregationCountValue) addSample(v interface{}) {
	*a = *a + 1
}

func (a *AggregationCountValue) multiplyByFraction(fraction float64) AggregationValue {
	return newAggregationCountValue(int64(float64(int64(*a))*fraction + 0.5)) // adding 0.5 because go runtime will take floor instead of rounding

}

func (a *AggregationCountValue) addToIt(av AggregationValue) {
	other, ok := av.(*AggregationCountValue)
	if !ok {
		return
	}
	*a = *a + *other
}

func (a *AggregationCountValue) clear() {
	*a = 0
}

func (a *AggregationCountValue) equal(other AggregationValue) bool {
	a2, ok := other.(*AggregationCountValue)
	if !ok {
		return false
	}

	return int64(*a) == int64(*a2)
}

func (a *AggregationCountValue) String() string {
	return fmt.Sprintf("{%v}", *a)
}

// AggregationDistributionValue is the aggregated data for an
// AggregationDistributionFloat64  or AggregationDistributionInt64.
type AggregationDistributionValue struct {
	count    int64
	min, max float64

	// mean and sumOfSquaredDev are the Knuth's algorithm variables to compute
	// the online average and variance for streaming values in a stable manner.
	// When the first sample x arrives:
	//
	// mean =  x
	// sumOfSquaredDev = 0
	//
	// For each subsequent sample x:
	//
	// count++
	// oldMean = mean
	// mean = mean + (x-mean) / count
	// sumOfSquaredDev = sumOfSquaredDev+(x-oldMean)(x-mean)
	mean, sumOfSquaredDev float64

	// countPerBucket is the set of occurrences count per bucket. The buckets
	// bounds are the same as the ones setup in AggregationDistribution.
	countPerBucket []int64
	bounds         []float64
}

// NewDoNotUseTestingAggregationDistributionValue allows to initialize a new
// AggregationDistributionValue to some desired values. It is expected to be
// used to facilitate testing only. It should not be invoked in production.
func NewDoNotUseTestingAggregationDistributionValue(bounds []float64, countPerBucket []int64, count int64, min, max, mean, sumOfSquaredDev float64) *AggregationDistributionValue {
	return &AggregationDistributionValue{
		countPerBucket:  countPerBucket,
		bounds:          bounds,
		count:           count,
		min:             min,
		max:             max,
		mean:            mean,
		sumOfSquaredDev: sumOfSquaredDev,
	}
}

func newAggregationDistributionValue(bounds []float64) *AggregationDistributionValue {
	return &AggregationDistributionValue{
		countPerBucket: make([]int64, len(bounds)+1),
		bounds:         bounds,
		min:            math.MaxFloat64,
		max:            math.SmallestNonzeroFloat64,
	}
}

// Count returns the count of all samples collected.
func (a *AggregationDistributionValue) Count() int64 { return a.count }

// Min returns the min of all samples collected.
func (a *AggregationDistributionValue) Min() float64 { return a.min }

// Mean returns the mean of all samples collected.
func (a *AggregationDistributionValue) Mean() float64 { return a.mean }

// Max returns the max of all samples collected.
func (a *AggregationDistributionValue) Max() float64 { return a.max }

// Sum returns the sum of all samples collected.
func (a *AggregationDistributionValue) Sum() float64 { return a.mean * float64(a.count) }

func (a *AggregationDistributionValue) variance() float64 {
	if a.count <= 1 {
		return 0
	}
	return a.SumOfSquaredDeviation() / float64(a.count-1)
}

// SumOfSquaredDeviation returns the sum of all samples deviations from the
// mean squared. This the M2 variable in Knuth's online algorithm for variance
// calculation. https://en.wikipedia.org/wiki/Algorithms_for_calculating_variance
func (a *AggregationDistributionValue) SumOfSquaredDeviation() float64 { return a.sumOfSquaredDev }

func (a *AggregationDistributionValue) String() string {
	return fmt.Sprintf("{%v %v %v %v %v %v %v}", a.Count(), a.Min(), a.Max(), a.Mean(), a.variance(), a.countPerBucket, a.bounds)
}

// CountPerBucket returns count per bucket. The buckets bounds are the same as
// the ones setup in AggregationDistribution.
func (a *AggregationDistributionValue) CountPerBucket() []int64 {
	var ret []int64
	for _, c := range a.countPerBucket {
		ret = append(ret, c)
	}
	return ret
}

func (a *AggregationDistributionValue) isAggregate() bool { return true }

func (a *AggregationDistributionValue) addSample(v interface{}) {
	var f float64
	switch x := v.(type) {
	case int64:
		f = float64(x)
		break
	case float64:
		f = x
		break
	default:
		return
	}

	if f < a.min {
		a.min = f
	}
	if f > a.max {
		a.max = f
	}
	a.count++
	a.incrementBucketCount(f)

	if a.count == 1 {
		a.mean = f
		return
	}

	oldMean := a.mean
	a.mean = a.mean + (f-a.mean)/float64(a.count)
	a.sumOfSquaredDev = a.sumOfSquaredDev + (f-oldMean)*(f-a.mean)
}

func (a *AggregationDistributionValue) incrementBucketCount(f float64) {
	if len(a.bounds) == 0 {
		a.countPerBucket[0]++
		return
	}

	for i, b := range a.bounds {
		if f < b {
			a.countPerBucket[i]++
			return
		}
	}
	a.countPerBucket[len(a.bounds)]++
}

// AggregationDistributionValue will not multiply by the fraction for this type
// of aggregation. The 'fraction' argument is there just to satisfy the
// interface 'AggregationValue'. For simplicity, we include the oldest partial
// bucket in its entirety when the aggregation is a distribution. We do not try
//  to multiply it by the fraction as it would make the calculation too complex
// and will create inconsistencies between sumOfSquaredDev, min, max and the
// various buckets of the histogram.
func (a *AggregationDistributionValue) multiplyByFraction(fraction float64) AggregationValue {
	ret := newAggregationDistributionValue(a.bounds)
	for i, c := range a.countPerBucket {
		ret.countPerBucket[i] = c
	}
	ret.count = a.count
	ret.min = a.min
	ret.max = a.max
	ret.mean = a.mean
	ret.sumOfSquaredDev = a.sumOfSquaredDev

	return ret

}

func (a *AggregationDistributionValue) addToIt(av AggregationValue) {
	other, ok := av.(*AggregationDistributionValue)
	if !ok {
		return
	}

	if other.count == 0 {
		return
	}

	if other.min < a.min {
		a.min = other.min
	}
	if other.max > a.max {
		a.max = other.max
	}

	delta := other.mean - a.mean
	a.sumOfSquaredDev = a.sumOfSquaredDev + other.sumOfSquaredDev + math.Pow(delta, 2)*float64(a.count*other.count)/(float64(a.count+other.count))

	a.mean = (a.Sum() + other.Sum()) / float64(a.count+other.count)
	a.count = a.count + other.count
	for i := range other.countPerBucket {
		a.countPerBucket[i] = a.countPerBucket[i] + other.countPerBucket[i]
	}
}

func (a *AggregationDistributionValue) clear() {
	a.count = 0
	a.min = math.MaxFloat64
	a.max = math.SmallestNonzeroFloat64
	a.mean = 0
	a.sumOfSquaredDev = 0
	for i := range a.countPerBucket {
		a.countPerBucket[i] = 0
	}
}

func (a *AggregationDistributionValue) equal(other AggregationValue) bool {
	a2, ok := other.(*AggregationDistributionValue)
	if !ok {
		return false
	}

	if a2 == nil {
		return false
	}

	if len(a.countPerBucket) != len(a2.countPerBucket) {
		return false
	}

	for i := range a.countPerBucket {
		if a.countPerBucket[i] != a2.countPerBucket[i] {
			return false
		}
	}

	epsilon := math.Pow10(-9)
	return a.Count() == a2.Count() && a.Min() == a2.Min() && a.Max() == a2.Max() && math.Pow(a.Mean()-a2.Mean(), 2) < epsilon && math.Pow(a.variance()-a2.variance(), 2) < epsilon
}
